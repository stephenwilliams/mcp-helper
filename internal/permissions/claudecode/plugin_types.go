package claudecode

// InstalledPluginsFile represents $CLAUDE_CONFIG_DIR/plugins/installed_plugins.json
type InstalledPluginsFile struct {
	Version int                           `json:"version"`
	Plugins map[string][]InstalledPlugin `json:"plugins"`
}

// InstalledPlugin represents a single plugin installation.
// Note: Plugin key format is "pluginName@marketplace" (e.g., "oh-my-claudecode@omc")
type InstalledPlugin struct {
	Scope        string `json:"scope"`
	InstallPath  string `json:"installPath"`
	Version      string `json:"version"`
	InstalledAt  string `json:"installedAt"`  // ISO 8601 timestamp for sorting
	LastUpdated  string `json:"lastUpdated"`
	GitCommitSha string `json:"gitCommitSha"`
}

// PluginManifest represents {installPath}/.claude-plugin/plugin.json
type PluginManifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	// MCPServers is the path to .mcp.json, RELATIVE TO installPath (not .claude-plugin/)
	MCPServers string `json:"mcpServers"`
}

// PluginBlocklist represents $CLAUDE_CONFIG_DIR/plugins/blocklist.json
type PluginBlocklist struct {
	Plugins []string `json:"plugins"` // List of blocked plugin keys
}
