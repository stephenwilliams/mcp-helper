<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# config

## Purpose

Configuration management for mcp-helper. Defines configuration types (`Config`, `Server`, `EnvVar`), provides XDG-compliant path utilities, and loads/validates YAML configuration using Viper.

## Key Files

| File | Description |
|------|-------------|
| `types.go` | Core types: `Config`, `Server`, `EnvVar` structs with YAML/mapstructure tags |
| `config.go` | XDG path utilities: `GetConfigDir()`, `GetDataDir()`, `GetCacheDir()`, etc. |
| `loader.go` | Configuration loading via Viper: `Load()`, `LoadFromPath()`, validation |
| `config_test.go` | Tests for XDG path functions |
| `loader_test.go` | Tests for config loading and validation |

## For AI Agents

### Working In This Directory

- Types use both `yaml` and `mapstructure` struct tags for Viper compatibility
- Configuration is optional - `Load()` returns nil config (not error) if no file found
- Validation happens during load via `Validate()` method

### Type Definitions

```go
type Config struct {
    DefaultScope string             `yaml:"default_scope" mapstructure:"default_scope"`
    Servers      map[string]*Server `yaml:"servers" mapstructure:"servers"`
}

type Server struct {
    Description string            `yaml:"description" mapstructure:"description"`
    Transport   string            `yaml:"transport" mapstructure:"transport"`  // "stdio" or "http"
    Command     string            `yaml:"command" mapstructure:"command"`      // For stdio
    Args        []string          `yaml:"args" mapstructure:"args"`            // For stdio
    URL         string            `yaml:"url" mapstructure:"url"`              // For http
    Env         map[string]EnvVar `yaml:"env" mapstructure:"env"`
}

type EnvVar struct {
    Required    bool   `yaml:"required" mapstructure:"required"`
    Description string `yaml:"description" mapstructure:"description"`
    Default     string `yaml:"default" mapstructure:"default"`
}
```

### Config Search Order

1. `MCP_HELPER_CONFIG` environment variable (explicit path)
2. `./.mcp-helper.yaml` (current directory)
3. `$XDG_CONFIG_HOME/mcp-helper/config.yaml`

### XDG Paths

- Config: `$XDG_CONFIG_HOME/mcp-helper/` (default: `~/.config/mcp-helper/`)
- Data: `$XDG_DATA_HOME/mcp-helper/`
- Cache: `$XDG_CACHE_HOME/mcp-helper/`

### Testing Requirements

- Test config loading from various paths
- Test validation errors for invalid transport values
- Test missing required fields
- Use `testdata/` fixtures

### Common Patterns

- `EnsureConfigDir()` creates directory lazily when needed
- `ListServers()` returns sorted server names
- `GetServer(name)` returns error if server not found

## Dependencies

### External

- `github.com/spf13/viper` - Configuration loading
- `github.com/adrg/xdg` - XDG Base Directory paths

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
