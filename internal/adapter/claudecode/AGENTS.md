<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# claudecode

## Purpose

Claude Code adapter implementation. Uses the `claude` CLI to add MCP servers via `claude mcp add` command. Supports both stdio and HTTP transports, environment variable configuration, and dry-run mode.

## Key Files

| File | Description |
|------|-------------|
| `claudecode.go` | `ClaudeCode` adapter implementing `adapter.Adapter` interface |
| `claudecode_test.go` | Unit tests for adapter functionality |

## For AI Agents

### Working In This Directory

- This adapter shells out to the `claude` CLI
- Command building is in `buildArgs()` method
- Supports custom claude binary path via `NewWithPath()`

### Command Format

**Stdio transport:**
```bash
claude mcp add --scope <scope> [-e KEY=val]... <name> -- <command> [args...]
```

**HTTP transport:**
```bash
claude mcp add --transport http --scope <scope> [-e KEY=val]... <name> <url>
```

### Key Methods

- `New()` - Creates adapter with default `claude` path
- `NewWithPath(path)` - Creates adapter with custom claude binary path
- `AddServer()` - Executes the claude mcp add command
- `DryRun()` - Returns the command string without executing
- `buildArgs()` - Constructs command-line arguments

### Testing Requirements

- Test argument building for both transports
- Test environment variable merging
- Test dry-run output formatting
- Mock exec.Command for integration tests

### Common Patterns

- Error messages include context about missing claude CLI
- Arguments with spaces are quoted in dry-run output
- Environment variables are merged: provided > defaults from config

## Dependencies

### Internal

- `internal/adapter` - Adapter interface and Scope type
- `internal/config` - Server configuration types

### External

- `os/exec` - Command execution

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
