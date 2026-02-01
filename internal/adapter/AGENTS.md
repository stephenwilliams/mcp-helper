<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# adapter

## Purpose

Defines the `Adapter` interface for managing MCP server configurations across different target environments. Provides scope types (`local`, `user`, `project`) and a common abstraction for adding servers with environment variables.

## Key Files

| File | Description |
|------|-------------|
| `adapter.go` | `Adapter` interface, `Scope` type, and `ParseScope()` function |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `claudecode/` | Claude Code adapter implementation (see `claudecode/AGENTS.md`) |

## For AI Agents

### Working In This Directory

- This package defines the interface, implementations are in subdirectories
- `Scope` is a string type with constants: `ScopeLocal`, `ScopeUser`, `ScopeProject`
- New adapters should implement the `Adapter` interface

### Adapter Interface

```go
type Adapter interface {
    Name() string
    AddServer(name string, server *config.Server, scope Scope, env map[string]string) error
    DryRun(name string, server *config.Server, scope Scope, env map[string]string) string
}
```

### Adding New Adapters

1. Create a new subdirectory (e.g., `vscode/`)
2. Implement the `Adapter` interface
3. Provide a constructor function (e.g., `New()`)
4. Create AGENTS.md in the subdirectory

### Testing Requirements

- Test scope parsing with valid and invalid inputs
- Mock adapters for testing commands

## Dependencies

### Internal

- `internal/config` - Server configuration types

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
