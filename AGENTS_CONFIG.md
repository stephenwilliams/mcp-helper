# MCP-Helper Config.yaml Reference

This document describes the `config.yaml` file format for mcp-helper.

## File Location

The config file is searched in this order:
1. `$MCP_HELPER_CONFIG` environment variable (if set)
2. `.mcp-helper.yaml` in current directory
3. `~/.config/mcp-helper/config.yaml`

---

## Schema

```yaml
# Default scope for add operations: "local" | "user" | "project"
# Optional. Defaults to "local" for most commands.
default_scope: user

# Server definitions (required)
servers:
  <server-name>:
    description: <string>        # Optional: human-readable description
    transport: <string>          # Required: "stdio" or "http"

    # For stdio transport:
    command: <string>            # Required: executable to run
    args: [<string>, ...]        # Optional: command arguments

    # For http transport:
    url: <string>                # Required: HTTP endpoint URL
    headers:                     # Optional: HTTP headers
      <Header-Name>: <string>    # Header value (supports templates)

    # Environment variables (optional):
    env:
      <VAR_NAME>:
        required: <bool>         # Optional: default false
        description: <string>    # Optional: shown during prompts
        default: <string>        # Optional: default value (supports templates)
```

---

## Server Types

### Stdio Server

Runs as a subprocess, communicates via stdin/stdout.

```yaml
servers:
  github:
    description: GitHub API integration
    transport: stdio
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-github"
    env:
      GITHUB_TOKEN:
        required: true
        description: Personal access token with repo scope
```

### HTTP Server

Connects to an HTTP endpoint. Supports custom headers for authentication.

```yaml
servers:
  my-api:
    description: Custom API server
    transport: http
    url: http://localhost:8080/mcp
    headers:
      Authorization: "Bearer {{ env \"API_TOKEN\" }}"
      X-Custom-Header: "static-value"
    env:
      API_TOKEN:
        required: true
        description: API authentication token
```

---

## Environment Variables

Each env var can have these properties:

| Property | Type | Default | Description |
|----------|------|---------|-------------|
| `required` | bool | false | Must be provided before server can be added |
| `description` | string | - | Help text shown during interactive prompts |
| `default` | string | - | Default value if not provided (supports templates) |

Variables and header names with `TOKEN`, `SECRET`, `KEY`, `PASSWORD`, `CREDENTIAL`, or `AUTHORIZATION` in their name are treated as secrets (masked in output).

---

## Template Support

The following fields support Go templates:
- `command`
- `args` (each element)
- `url`
- `headers.*` (header values, not names)
- `env.*.default`

### Available Functions

- All [slim-sprig functions](https://github.com/go-task/slim-sprig)
- `onePasswordRead "op://vault/item/field"` - Read from 1Password

### Template Context

```yaml
.Env    # Map of all environment variables
```

### Examples

```yaml
servers:
  templated:
    transport: stdio
    command: "{{ env \"HOME\" }}/bin/server"
    args:
      - --config={{ env "XDG_CONFIG_HOME" }}/app/config.yaml
    env:
      SECRET:
        default: "{{ onePasswordRead \"op://Private/my-secret/field\" }}"
```

---

## Complete Example

```yaml
default_scope: user

servers:
  github:
    description: GitHub API for repos, issues, and PRs
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN:
        required: true
        description: Personal access token with repo scope

  filesystem:
    description: Local filesystem access
    transport: stdio
    command: npx
    args:
      - "-y"
      - "@modelcontextprotocol/server-filesystem"
      - "{{ env \"HOME\" }}/projects"

  custom-api:
    description: Custom HTTP server with authentication
    transport: http
    url: http://localhost:3000/mcp
    headers:
      Authorization: "Bearer {{ env \"API_KEY\" }}"
    env:
      API_KEY:
        required: true
        description: API authentication key
      DEBUG:
        required: false
        default: "false"
```
