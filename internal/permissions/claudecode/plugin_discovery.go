package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/stephenwilliams/mcp-helper/internal/env"
	"github.com/stephenwilliams/mcp-helper/internal/mcp"
)

// getPluginMCPServers discovers MCP servers from installed Claude Code plugins
func (a *Adapter) getPluginMCPServers() ([]mcp.ServerConfig, error) {
	var servers []mcp.ServerConfig

	// 1. Get plugins directory (respects CLAUDE_CONFIG_DIR)
	configDir, err := env.ClaudeConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}
	pluginsDir := filepath.Join(configDir, "plugins")

	// 2. Read installed_plugins.json
	installedPath := filepath.Join(pluginsDir, "installed_plugins.json")
	installedData, err := os.ReadFile(installedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return servers, nil // No plugins installed
		}
		return nil, fmt.Errorf("failed to read installed_plugins.json: %w", err)
	}

	var installed InstalledPluginsFile
	if err := json.Unmarshal(installedData, &installed); err != nil {
		return nil, fmt.Errorf("failed to parse installed_plugins.json: %w", err)
	}

	// 3. Load blocklist (optional, missing file = no blocked plugins)
	blocklist := loadBlocklist(pluginsDir)

	// 4. Process each plugin
	for pluginKey, installations := range installed.Plugins {
		// Skip blocked plugins
		if blocklist[pluginKey] {
			continue
		}

		// Handle multiple installations: use latest by installedAt
		installation := selectLatestInstallation(installations)
		if installation == nil {
			continue
		}

		// Extract plugin name from key (format: "pluginName@marketplace")
		pluginName := extractPluginName(pluginKey)

		// Discover servers from this plugin
		pluginServers, err := a.discoverPluginServers(pluginName, installation.InstallPath)
		if err != nil {
			// Log warning but continue with other plugins
			continue
		}
		servers = append(servers, pluginServers...)
	}

	return servers, nil
}

// loadBlocklist reads the plugin blocklist, returning a set of blocked plugin keys
func loadBlocklist(pluginsDir string) map[string]bool {
	blocklist := make(map[string]bool)

	blocklistPath := filepath.Join(pluginsDir, "blocklist.json")
	data, err := os.ReadFile(blocklistPath)
	if err != nil {
		return blocklist // Missing or unreadable = no blocked plugins
	}

	var bl PluginBlocklist
	if err := json.Unmarshal(data, &bl); err != nil {
		return blocklist // Malformed = treat as empty
	}

	for _, plugin := range bl.Plugins {
		blocklist[plugin] = true
	}
	return blocklist
}

// selectLatestInstallation returns the installation with the latest installedAt timestamp
func selectLatestInstallation(installations []InstalledPlugin) *InstalledPlugin {
	if len(installations) == 0 {
		return nil
	}
	if len(installations) == 1 {
		return &installations[0]
	}

	// Sort by installedAt descending
	sorted := make([]InstalledPlugin, len(installations))
	copy(sorted, installations)
	sort.Slice(sorted, func(i, j int) bool {
		ti, _ := time.Parse(time.RFC3339, sorted[i].InstalledAt)
		tj, _ := time.Parse(time.RFC3339, sorted[j].InstalledAt)
		return ti.After(tj)
	})
	return &sorted[0]
}

// extractPluginName extracts the plugin name from a plugin key.
// Format: "pluginName@marketplace" -> "pluginName"
func extractPluginName(pluginKey string) string {
	if idx := strings.Index(pluginKey, "@"); idx != -1 {
		return pluginKey[:idx]
	}
	return pluginKey
}

// discoverPluginServers reads a plugin's manifest and MCP config
func (a *Adapter) discoverPluginServers(pluginName, installPath string) ([]mcp.ServerConfig, error) {
	var servers []mcp.ServerConfig

	// Read plugin manifest
	manifestPath := filepath.Join(installPath, ".claude-plugin", "plugin.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin manifest: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse plugin manifest: %w", err)
	}

	// Check if plugin has MCP servers
	if manifest.MCPServers == "" {
		return servers, nil // Plugin has no MCP servers
	}

	// Resolve MCP config path (relative to installPath, NOT .claude-plugin/)
	mcpConfigPath := filepath.Join(installPath, manifest.MCPServers)
	mcpData, err := os.ReadFile(mcpConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config: %w", err)
	}

	var mcpConfig MCPJSONConfig
	if err := json.Unmarshal(mcpData, &mcpConfig); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config: %w", err)
	}

	// Convert each server
	for serverName, cfg := range mcpConfig.MCPServers {
		server := mcp.ServerConfig{
			Name:      fmt.Sprintf("plugin:%s:%s", pluginName, serverName),
			Scope:     "plugin",
			Transport: "stdio",
			Command:   expandPluginRoot(cfg.Command, installPath),
			Args:      expandPluginRootSlice(cfg.Args, installPath),
			Env:       expandPluginRootMap(cfg.Env, installPath),
		}
		servers = append(servers, server)
	}

	return servers, nil
}

// expandPluginRoot expands ${CLAUDE_PLUGIN_ROOT} in a string.
// Security: Only expands this specific variable, not arbitrary env vars.
func expandPluginRoot(value, pluginRoot string) string {
	return strings.ReplaceAll(value, "${CLAUDE_PLUGIN_ROOT}", pluginRoot)
}

// expandPluginRootSlice expands ${CLAUDE_PLUGIN_ROOT} in a slice of strings
func expandPluginRootSlice(values []string, pluginRoot string) []string {
	if values == nil {
		return nil
	}
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = expandPluginRoot(v, pluginRoot)
	}
	return result
}

// expandPluginRootMap expands ${CLAUDE_PLUGIN_ROOT} in map values
func expandPluginRootMap(values map[string]string, pluginRoot string) map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[string]string, len(values))
	for k, v := range values {
		result[k] = expandPluginRoot(v, pluginRoot)
	}
	return result
}
