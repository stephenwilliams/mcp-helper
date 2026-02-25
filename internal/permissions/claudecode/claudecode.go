package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stephenwilliams/mcp-helper/internal/env"
	"github.com/stephenwilliams/mcp-helper/internal/mcp"
	"github.com/stephenwilliams/mcp-helper/internal/permissions"
)

// Adapter implements permissions.Adapter for Claude Code
type Adapter struct{}

// Verify interface compliance
var _ permissions.Adapter = (*Adapter)(nil)

func init() {
	// Use factory pattern for fresh instances (matches internal/adapter/registry.go)
	factory := func() permissions.Adapter { return &Adapter{} }
	permissions.Register("claudecode", factory)
	permissions.Register("claude-code", factory)
	permissions.Register("cc", factory)
}

func (a *Adapter) Name() string {
	return "claudecode"
}

// GetMCPServers reads MCP server configurations from all scopes:
// - User scope: $CLAUDE_CONFIG_DIR/.claude.json mcpServers (defaults to ~/.claude/.claude.json)
// - Local scope: $CLAUDE_CONFIG_DIR/.claude.json projects[cwd].mcpServers
// - Project scope: .mcp.json in current directory
func (a *Adapter) GetMCPServers() ([]mcp.ServerConfig, error) {
	var servers []mcp.ServerConfig

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Resolve symlinks to match paths in config (e.g., /var -> /private/var on macOS)
	resolvedCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		resolvedCwd = cwd // Fall back to original if symlink resolution fails
	}

	// Read .claude.json for user and local scope servers (respects CLAUDE_CONFIG_DIR)
	configDir, err := env.ClaudeConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	claudeConfigPath := filepath.Join(configDir, ".claude.json")
	data, err := os.ReadFile(claudeConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read %s: %w", claudeConfigPath, err)
	}

	if len(data) > 0 {
		var config ClaudeConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", claudeConfigPath, err)
		}

		// User scope: global mcpServers
		for name, cfg := range config.MCPServers {
			servers = append(servers, a.mcpConfigToServerConfig(name, "user", cfg))
		}

		// Local scope: project-specific mcpServers
		// Try both resolved path and original cwd to handle symlink variations
		for _, path := range []string{resolvedCwd, cwd} {
			if projectConfig, ok := config.Projects[path]; ok {
				for name, cfg := range projectConfig.MCPServers {
					servers = append(servers, a.mcpConfigToServerConfig(name, "local", cfg))
				}
				break // Found matching project, don't check again
			}
		}
	}

	// Project scope: .mcp.json in project root
	mcpJSONPath := filepath.Join(cwd, ".mcp.json")
	mcpData, err := os.ReadFile(mcpJSONPath)
	if err == nil {
		var mcpConfig MCPJSONConfig
		if err := json.Unmarshal(mcpData, &mcpConfig); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", mcpJSONPath, err)
		}

		for name, cfg := range mcpConfig.MCPServers {
			servers = append(servers, a.mcpConfigToServerConfig(name, "project", cfg))
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read %s: %w", mcpJSONPath, err)
	}

	return servers, nil
}

// mcpConfigToServerConfig converts an MCPServerConfig to mcp.ServerConfig
func (a *Adapter) mcpConfigToServerConfig(name, scope string, cfg MCPServerConfig) mcp.ServerConfig {
	server := mcp.ServerConfig{
		Name:  name,
		Scope: scope,
	}

	// Determine transport type based on config
	if cfg.Type == "http" || cfg.URL != "" {
		// HTTP transport
		server.Transport = "http"
		server.URL = cfg.URL
		server.Headers = cfg.Headers
	} else {
		// Default to stdio transport
		server.Transport = "stdio"
		server.Command = cfg.Command
		server.Args = cfg.Args
		server.Env = cfg.Env
	}

	return server
}

// GetSettingsPaths returns Claude Code settings file paths
func (a *Adapter) GetSettingsPaths() []permissions.SettingsPath {
	// Use ClaudeConfigDir to respect CLAUDE_CONFIG_DIR env var for user scope.
	configDir, _ := env.ClaudeConfigDir()

	paths := []permissions.SettingsPath{
		{
			Path:  filepath.Join(configDir, "settings.json"),
			Scope: "user",
		},
		{
			Path:  ".claude/settings.json",
			Scope: "project",
		},
		{
			Path:  ".claude/settings.local.json",
			Scope: "local",
		},
	}

	// Check which files exist and count MCP rules
	for i := range paths {
		if info, err := os.Stat(paths[i].Path); err == nil && !info.IsDir() {
			paths[i].Exists = true

			// Count existing MCP rules
			if rules, err := a.LoadPermissions(paths[i].Path); err == nil {
				for _, rule := range rules {
					if permissions.IsMCPRule(string(rule)) {
						paths[i].MCPRuleCount++
					}
				}
			}
		}
	}

	return paths
}

// LoadPermissions reads existing permissions from settings file
func (a *Adapter) LoadPermissions(path string) ([]permissions.PermissionRule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []permissions.PermissionRule{}, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var settings ClaudeCodeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if settings.Permissions == nil || settings.Permissions.Allow == nil {
		return []permissions.PermissionRule{}, nil
	}

	rules := make([]permissions.PermissionRule, len(settings.Permissions.Allow))
	for i, rule := range settings.Permissions.Allow {
		rules[i] = permissions.PermissionRule(rule)
	}

	return rules, nil
}

// SavePermissions writes permissions to settings file, preserving other fields
func (a *Adapter) SavePermissions(path string, rules []permissions.PermissionRule) error {
	// Read existing settings to preserve unknown fields
	var settings ClaudeCodeSettings
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing settings: %w", err)
	}

	if len(data) > 0 {
		// Parse into raw map first to preserve unknown fields
		if err := json.Unmarshal(data, &settings.raw); err != nil {
			return fmt.Errorf("failed to parse existing settings: %w", err)
		}
		// Then parse permissions specifically
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("failed to parse permissions from settings: %w", err)
		}
	} else {
		settings.raw = make(map[string]json.RawMessage)
	}

	// Create or update permissions block
	if settings.Permissions == nil {
		settings.Permissions = &PermissionsBlock{}
	}

	// Convert rules to strings
	allowRules := make([]string, len(rules))
	for i, rule := range rules {
		allowRules[i] = string(rule)
	}
	settings.Permissions.Allow = allowRules

	// Marshal the settings
	output, err := json.MarshalIndent(&settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write atomically using temp file + rename
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// FormatToolRule returns "mcp__server__tool" format
func (a *Adapter) FormatToolRule(serverName, toolName string) permissions.PermissionRule {
	return permissions.FormatMCPToolRule(serverName, toolName)
}

// FormatWildcardRule returns "mcp__server__*" format
func (a *Adapter) FormatWildcardRule(serverName string) permissions.PermissionRule {
	return permissions.FormatMCPWildcardRule(serverName)
}

// ClaudeCodeSettings represents settings.json with unknown field preservation
type ClaudeCodeSettings struct {
	Permissions *PermissionsBlock `json:"permissions,omitempty"`
	// Preserve unknown fields via custom marshal/unmarshal
	raw map[string]json.RawMessage
}

// MarshalJSON implements custom JSON marshaling to preserve unknown fields
func (s *ClaudeCodeSettings) MarshalJSON() ([]byte, error) {
	// Start with the raw map
	result := make(map[string]json.RawMessage)
	for k, v := range s.raw {
		result[k] = v
	}

	// Override with permissions if present
	if s.Permissions != nil {
		permData, err := json.Marshal(s.Permissions)
		if err != nil {
			return nil, err
		}
		result["permissions"] = permData
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements custom JSON unmarshaling to preserve unknown fields
func (s *ClaudeCodeSettings) UnmarshalJSON(data []byte) error {
	// Parse into raw map
	if err := json.Unmarshal(data, &s.raw); err != nil {
		return err
	}

	// Parse permissions specifically
	var temp struct {
		Permissions *PermissionsBlock `json:"permissions,omitempty"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	s.Permissions = temp.Permissions

	return nil
}

type PermissionsBlock struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// ClaudeConfig represents ~/.claude.json for MCP server discovery
type ClaudeConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
	// Projects contains per-project configurations keyed by project path
	Projects map[string]ProjectConfig `json:"projects"`
}

// ProjectConfig represents per-project configuration in ~/.claude.json
type ProjectConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPJSONConfig represents .mcp.json project-scope configuration
type MCPJSONConfig struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

type MCPServerConfig struct {
	// Transport type: "stdio" (default) or "http"
	Type string `json:"type,omitempty"`
	// Stdio transport fields
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	// HTTP transport fields
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}
