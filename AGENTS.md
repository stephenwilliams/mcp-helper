<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# mcp-helper

## Purpose

A CLI tool for managing and configuring MCP (Model Context Protocol) servers. It provides an interactive TUI for browsing available servers, configuring environment variables, and installing servers to Claude Code. The tool supports Go template processing for dynamic configuration values, including 1Password secret retrieval.

## Key Files

| File | Description |
|------|-------------|
| `main.go` | Application entry point, delegates to cmd.Execute() |
| `go.mod` | Go module definition with dependencies (cobra, viper, bubbletea, lipgloss) |
| `.goreleaser.yaml` | GoReleaser configuration for builds and releases |
| `mise.toml` | Development environment tool versions |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `cmd/` | CLI commands using Cobra (see `cmd/AGENTS.md`) |
| `internal/` | Internal packages (see `internal/AGENTS.md`) |
| `tui/` | Terminal UI using Bubbletea (see `tui/AGENTS.md`) |
| `testdata/` | Test fixtures and sample configs (see `testdata/AGENTS.md`) |

## For AI Agents

### Working In This Directory

- This is a Go 1.25 project using standard Go module layout
- Follow Go idioms: short variable names, error handling with early returns
- Use `internal/` for packages that should not be imported externally
- All CLI commands are in `cmd/` and registered via Cobra

### Build and Test Commands

```bash
# Build the binary
go build -o mcp-helper .

# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run a specific package's tests
go test ./internal/config/...
```

### Testing Requirements

- Write table-driven tests for new functionality
- Use `testdata/` for test fixtures
- Integration tests use `_test.go` suffix with build tag `// +build integration` if needed

### Common Patterns

- **Configuration**: Uses Viper for YAML config loading with XDG paths
- **CLI**: Cobra commands with persistent flags on root
- **Adapters**: Interface pattern for supporting multiple target environments
- **Templates**: Go text/template with slim-sprig functions

### Architecture Overview

```
User → CLI (cmd/) → Config (internal/config/)
                  → TUI (tui/) → Adapter (internal/adapter/)
                               → Env collection (internal/env/)
                               → Template processing (internal/template/)
```

**HTTP Transport Features:**
- Headers support via `-H` flags (e.g., `Authorization: Bearer ...`)
- Template processing in header values
- Secret header values masked in dry-run output

## Dependencies

### External

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - TUI styling
- `github.com/adrg/xdg` - XDG Base Directory paths
- `github.com/go-task/slim-sprig/v3` - Template functions
- `golang.org/x/term` - Terminal password input

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
