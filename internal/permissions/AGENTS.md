# Package permissions

The `permissions` package provides an abstraction layer for managing MCP tool permissions across different AI agent environments (Claude Code, OpenCode, etc.).

## Architecture

This package follows the **Adapter Pattern** to support multiple agent types:

```
permissions/
├── adapter.go         # Adapter interface definition
├── registry.go        # Adapter registration and lookup
├── permissions.go     # Common permission utilities
└── claudecode/        # Claude Code implementation
    └── claudecode.go
```

## Key Concepts

### Adapter Interface

Each AI agent environment implements the `Adapter` interface:

```go
type Adapter interface {
    Name() string
    GetMCPServers() ([]mcp.ServerConfig, error)
    GetSettingsPaths() []SettingsPath
    LoadPermissions(path string) ([]PermissionRule, error)
    SavePermissions(path string, rules []PermissionRule) error
    FormatToolRule(serverName, toolName string) PermissionRule
    FormatWildcardRule(serverName string) PermissionRule
}
```

### Permission Rules

Permission rules follow agent-specific formats:

- **Claude Code**: `mcp__server__tool` or `mcp__server__*`
- **Future agents**: May use different formats (handled by adapter)

### Registry Pattern

Adapters register themselves at initialization:

```go
func init() {
    permissions.Register("claudecode", func() permissions.Adapter {
        return &Adapter{}
    })
}
```

## Usage

### Get an Adapter

```go
import "github.com/stephenwilliams/mcp-helper/internal/permissions"
import _ "github.com/stephenwilliams/mcp-helper/internal/permissions/claudecode"

// Get adapter by name
adapter, err := permissions.Get("claudecode")

// Get with fallback to default
adapter, err := permissions.GetWithDefault(flagValue, "claudecode")
```

### Load and Save Permissions

```go
// Load existing permissions
rules, err := adapter.LoadPermissions("~/.claude/settings.json")

// Add new rules (with duplicate and wildcard coverage checking)
newRules := []permissions.PermissionRule{
    adapter.FormatToolRule("github", "search_repositories"),
    adapter.FormatWildcardRule("filesystem"),
}
merged := permissions.MergeRules(rules, newRules)

// Save back
err = adapter.SavePermissions("~/.claude/settings.json", merged)
```

### Discover MCP Servers

```go
servers, err := adapter.GetMCPServers()
// Returns []mcp.ServerConfig with Name, Scope, Command, Args, Env
```

## Claude Code Adapter

### File Locations

- **MCP Servers**: `~/.claude.json` → `mcpServers` key
- **User Permissions**: `~/.claude/settings.json` → `permissions.allow`
- **Project Permissions**: `.claude/settings.json` → `permissions.allow`
- **Local Permissions**: `.claude/settings.local.json` → `permissions.allow`

### Permission Format

```json
{
  "permissions": {
    "allow": [
      "mcp__github__*",
      "mcp__filesystem__read_file",
      "mcp__filesystem__write_file"
    ],
    "deny": [
      "mcp__filesystem__delete_file"
    ]
  }
}
```

### Features

- **Field Preservation**: Unknown JSON fields are preserved when saving
- **Atomic Writes**: Uses temp file + rename for safety
- **Duplicate Detection**: Skips rules that already exist
- **Wildcard Coverage**: Doesn't add `mcp__server__tool` if `mcp__server__*` exists
- **Directory Creation**: Automatically creates parent directories

## Utilities

### Format Rules

```go
// Specific tool
rule := permissions.FormatMCPToolRule("github", "search_repositories")
// Returns: "mcp__github__search_repositories"

// Wildcard
rule := permissions.FormatMCPWildcardRule("github")
// Returns: "mcp__github__*"
```

### Parse Rules

```go
server, tool, ok := permissions.ParseMCPRule("mcp__github__search_repositories")
// Returns: "github", "search_repositories", true

server, tool, ok := permissions.ParseMCPRule("mcp__github__*")
// Returns: "github", "*", true
```

### Check Coverage

```go
toolRule := permissions.FormatMCPToolRule("github", "search_repositories")
rules := []permissions.PermissionRule{"mcp__github__*"}

covered := permissions.IsCoveredByWildcard(toolRule, rules)
// Returns: true
```

### Merge Rules

```go
existing := []permissions.PermissionRule{
    "mcp__github__search_repositories",
    "mcp__filesystem__*",
}

newRules := []permissions.PermissionRule{
    "mcp__github__search_repositories",  // Duplicate - skipped
    "mcp__filesystem__read_file",        // Covered by wildcard - skipped
    "mcp__slack__send_message",          // New - added
}

merged := permissions.MergeRules(existing, newRules)
// Result: existing + ["mcp__slack__send_message"]
```

## Adding a New Adapter

To add support for a new AI agent:

1. Create a new package: `internal/permissions/newagent/`
2. Implement the `Adapter` interface
3. Register in `init()` function:

```go
package newagent

import "github.com/stephenwilliams/mcp-helper/internal/permissions"

type Adapter struct{}

func init() {
    permissions.Register("newagent", func() permissions.Adapter {
        return &Adapter{}
    })
}

func (a *Adapter) Name() string { return "newagent" }

// Implement other interface methods...
func (a *Adapter) FormatToolRule(server, tool string) permissions.PermissionRule {
    // Use agent-specific format
    return permissions.PermissionRule(fmt.Sprintf("custom:%s:%s", server, tool))
}
```

4. Import in your command code:

```go
import _ "github.com/stephenwilliams/mcp-helper/internal/permissions/newagent"
```

## Design Decisions

### Why Adapters?

Different AI agents use different:
- Configuration file locations
- Permission rule formats
- JSON structures
- Server discovery mechanisms

Adapters isolate these differences.

### Why Registry Pattern?

- **Extensibility**: New adapters can be added without modifying core code
- **Lazy Initialization**: Adapters are created on-demand via factory functions
- **Type Safety**: Central registry ensures interface compliance
- **Decoupling**: Commands don't need to know about specific adapter implementations

### Why Custom JSON Marshal/Unmarshal?

Settings files may contain:
- Unknown fields (future features)
- Custom extensions
- User-specific configurations

Custom marshaling preserves these fields when updating permissions.

## Testing

Run tests:

```bash
# All tests
go test ./internal/permissions/...

# Specific package
go test ./internal/permissions
go test ./internal/permissions/claudecode

# With coverage
go test -cover ./internal/permissions/...
```

## Thread Safety

The adapter registry uses `sync.RWMutex` for concurrent access:
- Multiple goroutines can read adapters simultaneously
- Write operations (registration) are serialized

Individual adapter instances are **not** thread-safe. Create separate instances for concurrent use.
