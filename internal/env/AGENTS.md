<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# env

## Purpose

Environment variable management for MCP server configurations. Handles validation of required variables, collection from multiple sources (provided, environment, defaults), and interactive prompting for missing values with secret detection.

## Key Files

| File | Description |
|------|-------------|
| `env.go` | Core functions: `IsSecret()`, `ValidateMissing()`, `CollectEnvVars()`, `Prompt()` |
| `env_test.go` | Unit tests for env utilities |
| `prompt_test.go` | Tests for prompting functionality |
| `integration_test.go` | Integration tests |

## For AI Agents

### Working In This Directory

- Secret detection is name-based (TOKEN, KEY, SECRET, PASSWORD, CREDENTIAL)
- Collection follows priority: provided > os.Getenv > config defaults
- Interactive mode prompts for missing required vars

### Key Functions

```go
// Check if variable name suggests sensitive data
func IsSecret(name string) bool

// Find required vars missing from provided map
func ValidateMissing(server *config.Server, provided map[string]string) []string

// Collect all env vars with fallback priority
func CollectEnvVars(server *config.Server, provided map[string]string, interactive bool) (map[string]string, error)

// Prompt user for a value (hides input for secrets)
func Prompt(name, description string) (string, error)
```

### Collection Priority

1. Values explicitly provided (e.g., from `--env` flags)
2. Values from current environment (`os.Getenv`)
3. Default values from server configuration
4. Interactive prompt (if enabled and value still missing)

### Testing Requirements

- Test secret detection with various patterns
- Test collection priority order
- Mock stdin for prompt tests
- Test error cases for missing required vars

### Common Patterns

- `term.ReadPassword()` for secret input (no echo)
- `bufio.Reader` for normal input
- Descriptive prompts with env var description

## Dependencies

### Internal

- `internal/config` - Server and EnvVar types

### External

- `golang.org/x/term` - Terminal password input

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
