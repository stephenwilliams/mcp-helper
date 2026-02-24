// Package permissions provides an abstraction layer for managing tool permissions
// across different AI agent environments. It defines a common interface for reading
// and writing permission rules in agent-specific formats.
package permissions

import "github.com/stephenwilliams/mcp-helper/internal/mcp"

// Adapter defines the interface for agent-specific permission handling.
// Each supported agent (Claude Code, OpenCode, Cursor, etc.) implements this interface.
type Adapter interface {
	// Name returns the adapter name (e.g., "claudecode", "opencode")
	Name() string

	// GetMCPServers reads MCP server configurations from the agent's config
	// Returns servers grouped by scope
	GetMCPServers() ([]mcp.ServerConfig, error)

	// GetSettingsPaths returns available settings file paths for this agent
	GetSettingsPaths() []SettingsPath

	// LoadPermissions reads existing permissions from a settings file
	LoadPermissions(path string) ([]PermissionRule, error)

	// SavePermissions writes permissions to a settings file
	// Merges with existing permissions, preserves other settings
	SavePermissions(path string, rules []PermissionRule) error

	// FormatToolRule formats a server+tool as a permission rule
	// e.g., Claude Code: "mcp__server__tool"
	FormatToolRule(serverName, toolName string) PermissionRule

	// FormatWildcardRule formats a server wildcard rule
	// e.g., Claude Code: "mcp__server__*"
	FormatWildcardRule(serverName string) PermissionRule
}

// SettingsPath represents a potential settings file location
type SettingsPath struct {
	Path         string
	Scope        string // "user", "project", "local"
	Exists       bool
	MCPRuleCount int // Number of existing MCP permission rules
}
