<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# testdata

## Purpose

Test fixtures and sample configuration files used by unit tests across the project.

## Key Files

| File | Description |
|------|-------------|
| `config_test.yaml` | Valid sample configuration for testing config loading |
| `invalid_config.yaml` | Intentionally invalid config for testing error handling |

## For AI Agents

### Working In This Directory

- Files here are read by tests, not executed
- Keep test fixtures minimal and focused
- Name files descriptively to indicate their test purpose
- YAML files should match the schema in `internal/config/types.go`

### Testing Requirements

- Update fixtures when config schema changes
- Add new fixtures for edge cases being tested
- Invalid fixtures should trigger specific validation errors

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
