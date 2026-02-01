<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# internal

## Purpose

Container directory for internal Go packages. These packages are not importable by external projects, following Go's `internal/` convention.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `adapter/` | Adapter interface and implementations for target environments (see `adapter/AGENTS.md`) |
| `config/` | Configuration types, loading, and XDG path utilities (see `config/AGENTS.md`) |
| `env/` | Environment variable collection and interactive prompting (see `env/AGENTS.md`) |
| `template/` | Go template processing with slim-sprig and 1Password (see `template/AGENTS.md`) |

## For AI Agents

### Working In This Directory

- All packages here are internal to mcp-helper
- Each subdirectory is a separate Go package
- Follow Go package naming conventions (lowercase, short)
- Packages should have minimal dependencies on each other

### Package Dependency Graph

```
cmd/ uses all internal packages
tui/ uses: config, adapter, env

internal/adapter/claudecode → internal/adapter, internal/config
internal/env → internal/config
internal/template → internal/config
```

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
