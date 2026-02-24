# mcp-helper

A CLI tool for managing and configuring MCP (Model Context Protocol) servers for Claude Code.

## Features

- **Interactive TUI Browser** - Browse available servers with a terminal UI
- **Configuration Management** - YAML-based server configs with XDG-compliant paths
- **Environment Variable Handling** - Interactive prompting with automatic secret detection
- **Template Processing** - Go templates with 70+ functions plus 1Password integration
- **Claude Code Integration** - Install servers via `claude mcp add` commands
- **AWS Profile Discovery** - Auto-detect AWS profiles with `-mcpro`/`-mcprw` suffixes
- **Multiple Transports** - Support for both stdio and HTTP MCP servers
- **Dry-Run Mode** - Preview commands without executing

## Installation

### Homebrew

```bash
brew install stephenwilliams/tap/mcp-helper
```

### From Source

```bash
go install github.com/stephenwilliams/mcp-helper@latest
```

### Build Locally

```bash
git clone https://github.com/stephenwilliams/mcp-helper.git
cd mcp-helper
go build -o mcp-helper .
```

## Quick Start

1. Initialize a sample configuration:

```bash
mcp-helper init
```

2. Browse and install servers interactively:

```bash
mcp-helper browse
```

3. Or add a specific server:

```bash
mcp-helper add my-server
```

## Configuration

Configuration files are loaded from XDG-compliant paths:

- `$XDG_CONFIG_HOME/mcp-helper/config.yaml` (default: `~/.config/mcp-helper/config.yaml`)
- Or specify with `--config`

### Example Configuration

```yaml
default_scope: user

servers:
  github:
    description: GitHub MCP server
    transport: stdio
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-github"
    env:
      GITHUB_TOKEN:
        required: true
        description: GitHub personal access token

  my-api:
    description: Custom HTTP MCP server
    transport: http
    url: http://localhost:8080
    env:
      API_KEY:
        required: true
        description: API key for authentication
      DEBUG:
        required: false
        description: Enable debug mode
        default: "false"
```

### Server Configuration Fields

| Field | Description |
|-------|-------------|
| `description` | Human-readable description |
| `transport` | `stdio` or `http` |
| `command` | Command to run (stdio only) |
| `args` | Command arguments (stdio only) |
| `url` | Server URL (http only) |
| `env` | Environment variables map |

### Environment Variable Options

| Field | Description |
|-------|-------------|
| `required` | Whether the variable is required |
| `description` | Description shown during prompts |
| `default` | Default value (supports templates) |

## Commands

### `browse`

Launch the interactive TUI to browse and install servers.

```bash
mcp-helper browse
```

Aliases: `interactive`, `ui`

### `list`

List all configured servers.

```bash
mcp-helper list
mcp-helper list --json
```

### `add`

Add a server to Claude Code.

```bash
# Interactive prompts for env vars
mcp-helper add github

# Specify scope
mcp-helper add github --scope project

# Provide env vars via flags
mcp-helper add github --env GITHUB_TOKEN=ghp_xxx

# Preview without executing
mcp-helper add github --dry-run

# Fail if env vars missing (no prompts)
mcp-helper add github --no-prompt
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--scope` | `local`, `user`, or `project` |
| `--env`, `-e` | Environment variable (KEY=VALUE, repeatable) |
| `--dry-run` | Show command without executing |
| `--no-prompt` | Fail if env vars missing instead of prompting |

### `info`

Show detailed information about a server.

```bash
mcp-helper info github
```

### `init`

Create sample configuration files.

```bash
mcp-helper init
```

### `aws discover`

Discover and add AWS MCP profiles automatically.

```bash
# Interactive selection
mcp-helper aws discover

# Add all discovered profiles
mcp-helper aws discover --all

# Preview without adding
mcp-helper aws discover --dry-run

# Output as JSON
mcp-helper aws discover --json

# Specific scope
mcp-helper aws discover --scope user

# Overwrite existing servers
mcp-helper aws discover --force
```

## Tool Discovery and Pre-Approval

Discover MCP tools from configured servers and pre-approve them to eliminate permission prompts during agent sessions.

### List Available Tools

```bash
# List all tools from configured MCP servers
mcp-helper tools list

# Output as JSON for scripting
mcp-helper tools list --json

# Filter by server or scope
mcp-helper tools list --server github
mcp-helper tools list --scope user

# Bypass cache for fresh results
mcp-helper tools list --no-cache
```

The table output shows server name, scope, tool count, and status:

```
SERVER          SCOPE     TOOLS  STATUS
github          user      5      ok
filesystem      project   3      ok
slack           user      -      timeout (10s)
```

### Pre-Approve Tools

```bash
# Interactive TUI to select and approve tools
mcp-helper tools approve

# Preview changes without applying
mcp-helper tools approve --dry-run

# Write to specific settings file
mcp-helper tools approve --target .claude/settings.local.json
```

The interactive TUI allows you to:
- Expand servers to view individual tools
- Select specific tools or use wildcards (`mcp__server__*` for all tools from a server)
- Choose the target settings file (user, project, or local)
- Preview changes in diff-style before applying
- Preserve existing permissions

Tools are added to the `permissions.allow` array in Claude Code's settings file with the format:
- Specific tool: `mcp__<server>__<tool>`
- Server wildcard: `mcp__<server>__*`

## Template Processing

Configuration values support Go templates with [slim-sprig](https://github.com/go-task/slim-sprig) functions.

### Environment Variables

```yaml
env:
  HOME_PATH:
    default: "{{ env \"HOME\" }}"
```

### 1Password Integration

Retrieve secrets from 1Password:

```yaml
env:
  API_TOKEN:
    default: '{{ onePasswordRead "op://vault/item/field" }}'
```

With specific account:

```yaml
env:
  API_TOKEN:
    default: '{{ onePasswordRead "op://vault/item/field" "my-account" }}'
```

Requires the [1Password CLI](https://developer.1password.com/docs/cli/) (`op`) to be installed and authenticated.

### Available Template Functions

All [slim-sprig](https://github.com/go-task/slim-sprig) functions are available, including:

- String: `trim`, `upper`, `lower`, `replace`, `contains`, etc.
- Math: `add`, `sub`, `mul`, `div`, etc.
- Encoding: `b64enc`, `b64dec`, etc.
- Environment: `env`, `expandenv`
- And many more (70+ functions)

## AWS Profile Discovery

The `aws discover` command finds AWS profiles configured with MCP suffixes:

- **`-mcpro`** - Read-only access
- **`-mcprw`** - Read-write access

### Example AWS Config

```ini
# ~/.aws/config

[profile dev-mcpro]
region = us-west-2
sso_start_url = https://my-sso.awsapps.com/start
sso_region = us-west-2
sso_account_id = 123456789012
sso_role_name = ReadOnlyAccess

[profile prod-mcprw]
region = us-east-1
sso_start_url = https://my-sso.awsapps.com/start
sso_region = us-east-1
sso_account_id = 987654321098
sso_role_name = PowerUserAccess
```

Run discovery:

```bash
$ mcp-helper aws discover --dry-run

Discovered AWS MCP Profiles

PROFILE              REGION          MODE         AUTH
────────────────────────────────────────────────────────────
dev-mcpro            us-west-2       read-only    SSO
prod-mcprw           us-east-1       read-write   SSO

Total: 2 profiles
```

### Prerequisites

AWS MCP servers require [uvx](https://github.com/astral-sh/uv) to be installed:

```bash
pip install uv
```

For SSO profiles, ensure you're logged in:

```bash
aws sso login --profile dev-mcpro
```

## Scopes

Servers can be installed to different scopes:

| Scope | Description | Config Location |
|-------|-------------|-----------------|
| `local` | Current directory only | `.claude/config.json` |
| `project` | Git project root | `<git-root>/.claude/config.json` |
| `user` | All projects for user | `~/.claude/config.json` |

## Secret Detection

Environment variables are automatically detected as secrets based on name patterns:

- `*TOKEN*`, `*SECRET*`, `*KEY*`, `*PASSWORD*`, `*CREDENTIAL*`

Secret values are hidden during interactive prompts.

## Development

### Requirements

- Go 1.25+

### Build

```bash
go build -o mcp-helper .
```

### Test

```bash
go test ./...
```

### Lint

```bash
golangci-lint run
```

## License

MIT License - see [LICENSE](LICENSE) for details.
