package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPluginRoot(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		root     string
		expected string
	}{
		{"basic expansion", "${CLAUDE_PLUGIN_ROOT}/bin/server", "/path/to/plugin", "/path/to/plugin/bin/server"},
		{"no expansion needed", "node", "/path/to/plugin", "node"},
		{"multiple occurrences", "${CLAUDE_PLUGIN_ROOT}${CLAUDE_PLUGIN_ROOT}", "/p", "/p/p"},
		{"empty string", "", "/path", ""},
		{"partial match", "${CLAUDE_PLUGIN", "/path", "${CLAUDE_PLUGIN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPluginRoot(tt.input, tt.root)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExpandPluginRootSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		root     string
		expected []string
	}{
		{"nil input", nil, "/path", nil},
		{"empty slice", []string{}, "/path", []string{}},
		{"single element", []string{"${CLAUDE_PLUGIN_ROOT}/file"}, "/p", []string{"/p/file"}},
		{"multiple elements", []string{"${CLAUDE_PLUGIN_ROOT}/a", "b", "${CLAUDE_PLUGIN_ROOT}/c"}, "/x", []string{"/x/a", "b", "/x/c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPluginRootSlice(tt.input, tt.root)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("index %d: got %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestExpandPluginRootMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		root     string
		expected map[string]string
	}{
		{"nil input", nil, "/path", nil},
		{"empty map", map[string]string{}, "/path", map[string]string{}},
		{"single element", map[string]string{"PATH": "${CLAUDE_PLUGIN_ROOT}/bin"}, "/p", map[string]string{"PATH": "/p/bin"}},
		{"mixed values", map[string]string{"A": "${CLAUDE_PLUGIN_ROOT}/a", "B": "literal"}, "/x", map[string]string{"A": "/x/a", "B": "literal"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPluginRootMap(tt.input, tt.root)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("length mismatch: got %d, want %d", len(result), len(tt.expected))
				return
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("key %q: got %q, want %q", k, result[k], v)
				}
			}
		})
	}
}

func TestExtractPluginName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"oh-my-claudecode@omc", "oh-my-claudecode"},
		{"simple-plugin@marketplace", "simple-plugin"},
		{"no-marketplace", "no-marketplace"},
		{"@starts-with-at", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractPluginName(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSelectLatestInstallation(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := selectLatestInstallation([]InstalledPlugin{})
		if result != nil {
			t.Errorf("expected nil for empty slice, got %v", result)
		}
	})

	t.Run("single installation", func(t *testing.T) {
		installations := []InstalledPlugin{
			{InstallPath: "/only", InstalledAt: "2026-01-01T00:00:00Z"},
		}
		result := selectLatestInstallation(installations)
		if result == nil || result.InstallPath != "/only" {
			t.Errorf("expected /only, got %v", result)
		}
	})

	t.Run("multiple installations", func(t *testing.T) {
		installations := []InstalledPlugin{
			{InstallPath: "/old", InstalledAt: "2026-01-01T00:00:00Z"},
			{InstallPath: "/latest", InstalledAt: "2026-02-25T00:00:00Z"},
			{InstallPath: "/middle", InstalledAt: "2026-02-01T00:00:00Z"},
		}
		result := selectLatestInstallation(installations)
		if result == nil || result.InstallPath != "/latest" {
			t.Errorf("expected /latest, got %v", result)
		}
	})

	t.Run("invalid timestamp format", func(t *testing.T) {
		installations := []InstalledPlugin{
			{InstallPath: "/invalid", InstalledAt: "not-a-timestamp"},
			{InstallPath: "/valid", InstalledAt: "2026-01-01T00:00:00Z"},
		}
		// Should not panic and should return something
		result := selectLatestInstallation(installations)
		if result == nil {
			t.Error("expected non-nil result even with invalid timestamps")
		}
	})
}

func TestLoadBlocklist(t *testing.T) {
	t.Run("missing file returns empty map", func(t *testing.T) {
		tmpDir := t.TempDir()
		blocklist := loadBlocklist(tmpDir)
		if len(blocklist) != 0 {
			t.Errorf("expected empty map, got %v", blocklist)
		}
	})

	t.Run("valid blocklist", func(t *testing.T) {
		tmpDir := t.TempDir()
		bl := PluginBlocklist{Plugins: []string{"blocked@omc", "also-blocked@npm"}}
		data, _ := json.Marshal(bl)
		if err := os.WriteFile(filepath.Join(tmpDir, "blocklist.json"), data, 0644); err != nil {
			t.Fatal(err)
		}

		blocklist := loadBlocklist(tmpDir)
		if !blocklist["blocked@omc"] {
			t.Error("expected blocked@omc to be blocked")
		}
		if !blocklist["also-blocked@npm"] {
			t.Error("expected also-blocked@npm to be blocked")
		}
		if blocklist["not-blocked@omc"] {
			t.Error("expected not-blocked@omc to not be blocked")
		}
	})

	t.Run("malformed JSON returns empty map", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmpDir, "blocklist.json"), []byte("not json"), 0644); err != nil {
			t.Fatal(err)
		}

		blocklist := loadBlocklist(tmpDir)
		if len(blocklist) != 0 {
			t.Errorf("expected empty map for malformed JSON, got %v", blocklist)
		}
	})
}

func TestGetPluginMCPServers(t *testing.T) {
	t.Run("full discovery", func(t *testing.T) {
		// Create temp directory structure
		tmpDir := t.TempDir()

		// Setup: $CLAUDE_CONFIG_DIR/plugins/
		pluginsDir := filepath.Join(tmpDir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Setup: plugin install path
		installPath := filepath.Join(pluginsDir, "cache", "omc", "test-plugin", "1.0.0")
		if err := os.MkdirAll(filepath.Join(installPath, ".claude-plugin"), 0755); err != nil {
			t.Fatal(err)
		}

		// Write installed_plugins.json
		installed := InstalledPluginsFile{
			Version: 2,
			Plugins: map[string][]InstalledPlugin{
				"test-plugin@omc": {{
					Scope:       "user",
					InstallPath: installPath,
					Version:     "1.0.0",
					InstalledAt: "2026-02-25T00:00:00Z",
				}},
			},
		}
		installedJSON, _ := json.Marshal(installed)
		if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedJSON, 0644); err != nil {
			t.Fatal(err)
		}

		// Write plugin manifest
		manifest := PluginManifest{
			Name:       "test-plugin",
			Version:    "1.0.0",
			MCPServers: "./.mcp.json",
		}
		manifestJSON, _ := json.Marshal(manifest)
		if err := os.WriteFile(filepath.Join(installPath, ".claude-plugin", "plugin.json"), manifestJSON, 0644); err != nil {
			t.Fatal(err)
		}

		// Write MCP config
		mcpConfig := MCPJSONConfig{
			MCPServers: map[string]MCPServerConfig{
				"test-server": {
					Command: "node",
					Args:    []string{"${CLAUDE_PLUGIN_ROOT}/server.js"},
					Env:     map[string]string{"PLUGIN_PATH": "${CLAUDE_PLUGIN_ROOT}"},
				},
			},
		}
		mcpJSON, _ := json.Marshal(mcpConfig)
		if err := os.WriteFile(filepath.Join(installPath, ".mcp.json"), mcpJSON, 0644); err != nil {
			t.Fatal(err)
		}

		// Set CLAUDE_CONFIG_DIR for test
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		// Run discovery
		adapter := &Adapter{}
		servers, err := adapter.getPluginMCPServers()

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(servers) != 1 {
			t.Fatalf("expected 1 server, got %d", len(servers))
		}

		server := servers[0]
		if server.Name != "plugin:test-plugin:test-server" {
			t.Errorf("unexpected name: %s", server.Name)
		}
		if server.Scope != "plugin" {
			t.Errorf("unexpected scope: %s", server.Scope)
		}
		if server.Transport != "stdio" {
			t.Errorf("unexpected transport: %s", server.Transport)
		}
		if server.Command != "node" {
			t.Errorf("unexpected command: %s", server.Command)
		}
		expectedArg := installPath + "/server.js"
		if len(server.Args) != 1 || server.Args[0] != expectedArg {
			t.Errorf("${CLAUDE_PLUGIN_ROOT} not expanded in args: %v", server.Args)
		}
		if server.Env["PLUGIN_PATH"] != installPath {
			t.Errorf("${CLAUDE_PLUGIN_ROOT} not expanded in env: %v", server.Env)
		}
	})

	t.Run("multiple servers in plugin", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		installPath := filepath.Join(pluginsDir, "cache", "omc", "multi", "1.0.0")
		if err := os.MkdirAll(filepath.Join(installPath, ".claude-plugin"), 0755); err != nil {
			t.Fatal(err)
		}

		installed := InstalledPluginsFile{
			Version: 2,
			Plugins: map[string][]InstalledPlugin{
				"multi@omc": {{InstallPath: installPath, InstalledAt: "2026-01-01T00:00:00Z"}},
			},
		}
		installedJSON, _ := json.Marshal(installed)
		os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedJSON, 0644)

		manifest := PluginManifest{MCPServers: ".mcp.json"}
		manifestJSON, _ := json.Marshal(manifest)
		os.WriteFile(filepath.Join(installPath, ".claude-plugin", "plugin.json"), manifestJSON, 0644)

		mcpConfig := MCPJSONConfig{
			MCPServers: map[string]MCPServerConfig{
				"server-a": {Command: "node", Args: []string{"a.js"}},
				"server-b": {Command: "python", Args: []string{"b.py"}},
			},
		}
		mcpJSON, _ := json.Marshal(mcpConfig)
		os.WriteFile(filepath.Join(installPath, ".mcp.json"), mcpJSON, 0644)

		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		adapter := &Adapter{}
		servers, err := adapter.getPluginMCPServers()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(servers) != 2 {
			t.Errorf("expected 2 servers, got %d", len(servers))
		}
	})
}

func TestGetPluginMCPServers_BlockedPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write installed_plugins.json with a plugin
	installed := InstalledPluginsFile{
		Version: 2,
		Plugins: map[string][]InstalledPlugin{
			"blocked-plugin@omc": {{InstallPath: "/fake", InstalledAt: "2026-01-01T00:00:00Z"}},
		},
	}
	installedJSON, _ := json.Marshal(installed)
	if err := os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedJSON, 0644); err != nil {
		t.Fatal(err)
	}

	// Write blocklist that blocks the plugin
	blocklist := PluginBlocklist{Plugins: []string{"blocked-plugin@omc"}}
	blocklistJSON, _ := json.Marshal(blocklist)
	if err := os.WriteFile(filepath.Join(pluginsDir, "blocklist.json"), blocklistJSON, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

	adapter := &Adapter{}
	servers, err := adapter.getPluginMCPServers()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 0 {
		t.Errorf("expected 0 servers (plugin blocked), got %d", len(servers))
	}
}

func TestGetPluginMCPServers_MissingFiles(t *testing.T) {
	t.Run("no plugins directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		// No plugins directory exists
		adapter := &Adapter{}
		servers, err := adapter.getPluginMCPServers()

		if err != nil {
			t.Fatalf("missing plugins dir should not error: %v", err)
		}
		if len(servers) != 0 {
			t.Errorf("expected 0 servers, got %d", len(servers))
		}
	})

	t.Run("empty installed_plugins.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		os.MkdirAll(pluginsDir, 0755)

		installed := InstalledPluginsFile{Version: 2, Plugins: map[string][]InstalledPlugin{}}
		installedJSON, _ := json.Marshal(installed)
		os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedJSON, 0644)

		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		adapter := &Adapter{}
		servers, err := adapter.getPluginMCPServers()

		if err != nil {
			t.Fatalf("empty plugins should not error: %v", err)
		}
		if len(servers) != 0 {
			t.Errorf("expected 0 servers, got %d", len(servers))
		}
	})

	t.Run("plugin with no mcpServers field", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		installPath := filepath.Join(pluginsDir, "cache", "omc", "no-mcp", "1.0.0")
		os.MkdirAll(filepath.Join(installPath, ".claude-plugin"), 0755)

		installed := InstalledPluginsFile{
			Version: 2,
			Plugins: map[string][]InstalledPlugin{
				"no-mcp@omc": {{InstallPath: installPath, InstalledAt: "2026-01-01T00:00:00Z"}},
			},
		}
		installedJSON, _ := json.Marshal(installed)
		os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedJSON, 0644)

		// Manifest with no MCPServers
		manifest := PluginManifest{Name: "no-mcp", Version: "1.0.0"}
		manifestJSON, _ := json.Marshal(manifest)
		os.WriteFile(filepath.Join(installPath, ".claude-plugin", "plugin.json"), manifestJSON, 0644)

		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		adapter := &Adapter{}
		servers, err := adapter.getPluginMCPServers()

		if err != nil {
			t.Fatalf("plugin with no MCP servers should not error: %v", err)
		}
		if len(servers) != 0 {
			t.Errorf("expected 0 servers, got %d", len(servers))
		}
	})

	t.Run("malformed installed_plugins.json", func(t *testing.T) {
		tmpDir := t.TempDir()
		pluginsDir := filepath.Join(tmpDir, "plugins")
		os.MkdirAll(pluginsDir, 0755)

		os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), []byte("not json"), 0644)

		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		adapter := &Adapter{}
		_, err := adapter.getPluginMCPServers()

		if err == nil {
			t.Error("expected error for malformed JSON")
		}
	})
}

func TestGetPluginMCPServers_LatestInstallation(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")

	// Create two install paths
	oldPath := filepath.Join(pluginsDir, "cache", "omc", "plugin", "1.0.0")
	newPath := filepath.Join(pluginsDir, "cache", "omc", "plugin", "2.0.0")
	os.MkdirAll(filepath.Join(oldPath, ".claude-plugin"), 0755)
	os.MkdirAll(filepath.Join(newPath, ".claude-plugin"), 0755)

	// Multiple installations - new one should be selected
	installed := InstalledPluginsFile{
		Version: 2,
		Plugins: map[string][]InstalledPlugin{
			"plugin@omc": {
				{InstallPath: oldPath, InstalledAt: "2026-01-01T00:00:00Z"},
				{InstallPath: newPath, InstalledAt: "2026-02-25T00:00:00Z"},
			},
		},
	}
	installedJSON, _ := json.Marshal(installed)
	os.WriteFile(filepath.Join(pluginsDir, "installed_plugins.json"), installedJSON, 0644)

	// Only new path has valid manifest
	manifest := PluginManifest{MCPServers: ".mcp.json"}
	manifestJSON, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(newPath, ".claude-plugin", "plugin.json"), manifestJSON, 0644)

	mcpConfig := MCPJSONConfig{
		MCPServers: map[string]MCPServerConfig{
			"server": {Command: "node"},
		},
	}
	mcpJSON, _ := json.Marshal(mcpConfig)
	os.WriteFile(filepath.Join(newPath, ".mcp.json"), mcpJSON, 0644)

	t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

	adapter := &Adapter{}
	servers, err := adapter.getPluginMCPServers()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	// The server was discovered, meaning the latest installation was used
}
