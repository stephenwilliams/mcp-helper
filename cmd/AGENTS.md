<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# cmd

## Purpose

CLI command implementations using Cobra. This package defines all user-facing commands for mcp-helper, including server management, configuration initialization, and the interactive browser.

## Key Files

| File | Description |
|------|-------------|
| `root.go` | Root command setup, config loading, version info, and `browse` subcommand |
| `add.go` | `add` command for installing MCP servers to Claude Code |
| `list.go` | `list` command for displaying configured servers (table/JSON) |
| `init.go` | `init` command for creating sample configuration files |
| `info.go` | `info` command for showing detailed server information |
| `cmd_test.go` | Command tests |

## For AI Agents

### Working In This Directory

- Each command is in its own file with `init()` registering it to `rootCmd`
- Global flags go on `rootCmd.PersistentFlags()`
- Command-specific flags use `cmd.Flags()` in the command's `init()`
- Use `GetConfig()` to access the loaded configuration

### Command Structure Pattern

```go
var myCmd = &cobra.Command{
    Use:   "mycommand <args>",
    Short: "One-line description",
    Long:  `Detailed description with examples.`,
    Args:  cobra.ExactArgs(1),  // or other validators
    RunE:  runMyCommand,
}

func init() {
    rootCmd.AddCommand(myCmd)
    myCmd.Flags().BoolVar(&myFlag, "flag", false, "description")
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    // Implementation
}
```

### Testing Requirements

- Test commands using `cmd.Execute()` with captured output
- Mock adapters when testing installation commands
- Use temporary config files for config-dependent tests

### Common Patterns

- JSON output via `--json` flag using `encoding/json`
- Tabular output via `text/tabwriter`
- Scope parsing via `adapter.ParseScope()`
- Template processing before env var collection

## Dependencies

### Internal

- `internal/config` - Configuration loading and types
- `internal/adapter` - Adapter interface and scope types
- `internal/adapter/claudecode` - Claude Code adapter
- `internal/env` - Environment variable collection
- `internal/template` - Template processing
- `tui` - Interactive browser UI

### External

- `github.com/spf13/cobra` - Command framework

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
