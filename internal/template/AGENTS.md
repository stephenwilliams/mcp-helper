<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# template

## Purpose

Go template processing for MCP server configurations. Supports slim-sprig template functions and custom functions like `onePasswordRead` for 1Password secret retrieval. Enables dynamic configuration values that are resolved at runtime.

## Key Files

| File | Description |
|------|-------------|
| `template.go` | Core template functions: `HasTemplateSyntax()`, `ProcessString()`, `onePasswordRead()` |
| `data.go` | `TemplateData` struct providing template context (environment variables) |
| `server.go` | `ProcessServer()` for processing all template fields in a server config |
| `template_test.go` | Unit tests for template processing |
| `onepassword_test.go` | Tests for 1Password integration |
| `server_test.go` | Tests for server processing |
| `integration_test.go` | Integration tests |

## For AI Agents

### Working In This Directory

- Template syntax detection uses regex for `{{ }}` patterns
- Slim-sprig provides 70+ template functions (env, default, etc.)
- 1Password integration shells out to `op` CLI
- Template processing happens BEFORE env var collection

### Template Execution Flow

1. `HasTemplateSyntax()` - Fast check for `{{ }}` patterns
2. `ProcessString()` - Process single string if template syntax found
3. `ProcessServer()` - Process all templatable fields in server config

### Templated vs Non-Templated Fields

**Templated:**
- `Command`, `Args[]`, `URL`, `EnvVar.Default`

**Not Templated (static):**
- `Description`, `Transport`, `EnvVar.Description`, `EnvVar.Required`

### Template Data Context

```go
type TemplateData struct {
    Env map[string]string  // Current environment variables
}
```

Access in templates: `{{ .Env.HOME }}`, `{{ .Env.USER }}`

### Custom Functions

```go
// Retrieve secret from 1Password
onePasswordRead "op://vault/item/field" ["account"]
```

### Common slim-sprig Functions

- `{{ env "VAR" }}` - Get environment variable
- `{{ default "fallback" .Env.VAR }}` - Default value
- `{{ required "msg" .Env.VAR }}` - Fail if empty
- `{{ upper "text" }}`, `{{ lower "text" }}` - Case conversion

### Testing Requirements

- Test template syntax detection edge cases
- Test slim-sprig function availability
- Mock `op` CLI for 1Password tests
- Test error handling for invalid templates

### Common Patterns

- Fast path: skip template execution if no `{{ }}` found
- New server returned (original unchanged) by `ProcessServer()`
- Errors include context about which field failed

## Dependencies

### Internal

- `internal/config` - Server configuration types

### External

- `github.com/go-task/slim-sprig/v3` - Template functions
- `text/template` - Go standard library templates
- `os/exec` - 1Password CLI execution

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
