# Package mcp

The `mcp` package provides a client for discovering tools from MCP (Model Context Protocol) servers using the JSON-RPC 2.0 protocol with stdio transport.

## Architecture

```
mcp/
├── types.go          # JSON-RPC types and MCP data structures
├── client.go         # MCP client with handshake and tool discovery
├── cache.go          # Tool discovery cache with TTL
├── client_test.go    # Unit tests
└── cache_test.go     # Cache tests
```

## Key Concepts

### MCP Handshake Sequence

The client performs a complete MCP protocol handshake before discovering tools:

1. **Initialize**: Client sends initialize request with protocol version and client info
2. **Initialized**: Client sends initialized notification after receiving response
3. **Tools List**: Client requests tools/list to discover available tools
4. **Cleanup**: Client gracefully terminates the server process

```go
// Example handshake flow
ctx := context.Background()
client := mcp.NewClient(10*time.Second, cache)

tools, err := client.ListToolsStdio(ctx, "npx", []string{"@modelcontextprotocol/server-github"}, env)
```

### Tool Discovery with Caching

Tools are cached per server with a 1-hour TTL to speed up repeated discovery:

- **Cache Key Format**: `scope:serverName` (e.g., `user:github`)
- **Default TTL**: 1 hour
- **Bypass Cache**: Pass `--no-cache` flag to force fresh discovery

```go
cache := mcp.NewCache(mcp.DefaultCacheTTL)
client := mcp.NewClient(10*time.Second, cache)

// Cached access
tools := client.DiscoverTools(ctx, servers)

// Cache is transparent - hits returned without server connection
```

### JSON-RPC Protocol

The client communicates with MCP servers via JSON-RPC 2.0 over stdin/stdout:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "mcp-helper",
      "version": "1.0.0"
    }
  }
}
```

## Usage

### Basic Tool Discovery

```go
import "github.com/stephenwilliams/mcp-helper/internal/mcp"

// Create client with cache
cache := mcp.NewCache(mcp.DefaultCacheTTL)
client := mcp.NewClient(10*time.Second, cache)

// Discover tools from a single server
ctx := context.Background()
tools, err := client.ListToolsStdio(
    ctx,
    "npx",
    []string{"@modelcontextprotocol/server-github"},
    map[string]string{"GITHUB_TOKEN": "ghp_xxx"},
)
if err != nil {
    log.Fatal(err)
}

for _, tool := range tools {
    fmt.Printf("Tool: %s\n", tool.Name)
    fmt.Printf("  Description: %s\n", tool.Description)
}
```

### Discover from Multiple Servers

```go
// Discover from multiple servers concurrently
servers := []mcp.ServerConfig{
    {
        Name:    "github",
        Scope:   "user",
        Command: "npx",
        Args:    []string{"@modelcontextprotocol/server-github"},
        Env:     map[string]string{"GITHUB_TOKEN": "ghp_xxx"},
    },
    {
        Name:    "filesystem",
        Scope:   "project",
        Command: "node",
        Args:    []string{"/path/to/filesystem-server.js"},
        Env:     map[string]string{},
    },
}

results := client.DiscoverTools(ctx, servers)

for _, result := range results {
    if result.Error != nil {
        fmt.Printf("Server %s failed: %v\n", result.Name, result.Error)
        continue
    }
    fmt.Printf("Server %s has %d tools\n", result.Name, len(result.Tools))
}
```

### Handle Pagination

Tools lists can be paginated. The client handles this automatically:

```go
// Large tool lists are paginated transparently
// The client follows nextCursor until all tools are retrieved
tools, err := client.ListToolsStdio(ctx, cmd, args, env)

// All tools returned in a single slice, pagination handled internally
fmt.Printf("Total tools: %d\n", len(tools))
```

## Cache Management

### Cache Lifecycle

```go
cache := mcp.NewCache(1 * time.Hour)

// Tools are cached automatically on first discovery
results := client.DiscoverTools(ctx, servers)

// Subsequent calls hit cache (within TTL)
results = client.DiscoverTools(ctx, servers)

// Clear cache for specific server
cache.ClearServer("github")

// Clear all cached entries
cache.Clear()
```

### Cache Key Format

Cache keys prevent collisions between servers with the same name in different scopes:

```
scope:serverName

Examples:
- user:github       # GitHub server in user scope
- project:github    # GitHub server in project scope
- local:custom      # Custom server in local scope
```

## Error Handling

### Connection Failures

The client handles various error scenarios gracefully:

```go
results := client.DiscoverTools(ctx, servers)

for _, result := range results {
    if result.Error != nil {
        switch result.Error.Error() {
        case "connection timeout after 10s":
            // Server didn't respond in time
            log.Printf("Server %s timed out", result.Name)
        case "command not found":
            // Server process couldn't be started
            log.Printf("Server %s not installed", result.Name)
        default:
            // Other errors
            log.Printf("Server %s error: %v", result.Name, result.Error)
        }
    }
}
```

### Timeout Handling

Each server connection has a configurable timeout (default 10 seconds):

```go
// Create client with custom timeout
client := mcp.NewClient(5*time.Second, cache)

// Timeout triggers graceful shutdown:
// 1. Send SIGTERM to process
// 2. Wait 2 seconds for graceful exit
// 3. Force SIGKILL if still running
```

### Malformed Responses

The client validates JSON-RPC responses:

```go
// Invalid JSON is rejected
// Missing required fields are reported as errors
// Out-of-order notifications are handled gracefully
```

## Types

### Tool

```go
type Tool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description,omitempty"`
    InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}
```

### ServerInfo

```go
type ServerInfo struct {
    Name      string       // Server name
    Scope     string       // "user", "project", or "local"
    Transport string       // "stdio" or "http"
    Tools     []Tool       // Discovered tools (nil if error)
    Error     error        // Non-nil if connection failed
}
```

### ServerConfig

```go
type ServerConfig struct {
    Name    string
    Scope   string
    Command string
    Args    []string
    Env     map[string]string
}
```

## Process Management

### Safe Process Cleanup

The client uses `exec.CommandContext` for automatic cleanup:

- **Timeout**: Context cancellation triggers SIGKILL automatically
- **Graceful Shutdown**: SIGTERM sent first, 2s grace period, then SIGKILL
- **Stdio Separation**: stdout (JSON-RPC) and stderr (logging) are separate streams

```go
// Processes are cleaned up automatically
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

tools, err := client.ListToolsStdio(ctx, cmd, args, env)
// Context cleanup ensures process terminates
```

### Environment Variables

Environment variables are passed to server processes:

```go
env := map[string]string{
    "GITHUB_TOKEN": "ghp_xxx",
    "HOME":         "/home/user",
}

tools, err := client.ListToolsStdio(ctx, "npx", args, env)
```

## Testing

Run tests:

```bash
# All tests
go test ./internal/mcp/...

# Specific package
go test ./internal/mcp
go test ./internal/mcp -v

# With coverage
go test -cover ./internal/mcp/...

# Run with race detector
go test -race ./internal/mcp/...
```

## Thread Safety

The MCP client itself is **not** thread-safe. Create separate client instances for concurrent use. The cache **is** thread-safe and can be shared across goroutines:

```go
// Safe - shared cache, separate clients
cache := mcp.NewCache(1 * time.Hour)

client1 := mcp.NewClient(10*time.Second, cache)
client2 := mcp.NewClient(10*time.Second, cache)

// Each client runs independently
// Both benefit from shared cache
go func() { client1.DiscoverTools(ctx, servers1) }()
go func() { client2.DiscoverTools(ctx, servers2) }()
```

## Design Decisions

### Why Stdio Transport First?

Stdio is the most common MCP transport in current use:
- No network setup required
- Works in restricted environments
- Simple process management
- HTTP deferred to Phase 2

### Why Caching?

Tool discovery can be slow (network requests, subprocess startup). Caching with TTL:
- Improves CLI performance for repeated runs
- Can be bypassed with `--no-cache` flag
- Per-scope cache keys prevent collisions

### Why Separate stdout/stderr?

MCP protocol sends JSON-RPC over stdout. Logging/debugging goes to stderr:
- stdout must be pure JSON-RPC for protocol compliance
- stderr can be discarded or logged separately
- Mixed streams would corrupt protocol messages

## References

- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
