package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

func TestNew(t *testing.T) {
	o := New()
	if o == nil {
		t.Fatal("New() returned nil")
	}
}

func TestName(t *testing.T) {
	o := New()
	if o.Name() != "OpenCode" {
		t.Errorf("expected 'OpenCode', got '%s'", o.Name())
	}
}

func TestGetConfigPath_User(t *testing.T) {
	o := New()
	path := o.GetConfigPath(adapter.ScopeUser)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "opencode", "opencode.json")

	if path != expected {
		t.Errorf("expected '%s', got '%s'", expected, path)
	}
}

func TestGetConfigPath_Local(t *testing.T) {
	o := New()
	path := o.GetConfigPath(adapter.ScopeLocal)

	if path != "opencode.json" {
		t.Errorf("expected 'opencode.json', got '%s'", path)
	}
}

func TestGetConfigPath_Project(t *testing.T) {
	o := New()
	path := o.GetConfigPath(adapter.ScopeProject)

	if path != "opencode.json" {
		t.Errorf("expected 'opencode.json', got '%s'", path)
	}
}

func TestServerExists_NotFound(t *testing.T) {
	o := New()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// No config file exists
	if o.ServerExists("test-server", adapter.ScopeLocal) {
		t.Error("expected ServerExists to return false when no config exists")
	}
}

func TestServerExists_Found(t *testing.T) {
	o := New()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create config with a server
	cfg := OpenCodeConfig{
		MCP: map[string]OpenCodeMCPServer{
			"existing-server": {
				Type:    "local",
				Command: []string{"node", "server.js"},
				Enabled: true,
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile("opencode.json", data, 0644)

	if !o.ServerExists("existing-server", adapter.ScopeLocal) {
		t.Error("expected ServerExists to return true for existing server")
	}

	if o.ServerExists("nonexistent", adapter.ScopeLocal) {
		t.Error("expected ServerExists to return false for nonexistent server")
	}
}

func TestServerExists_InvalidJSON(t *testing.T) {
	o := New()

	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Write invalid JSON
	os.WriteFile("opencode.json", []byte("not valid json"), 0644)

	if o.ServerExists("any-server", adapter.ScopeLocal) {
		t.Error("expected ServerExists to return false for invalid JSON")
	}
}

func TestAddServer_StdioTransport(t *testing.T) {
	o := New()

	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	server := &config.Server{
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "@modelcontextprotocol/server-github"},
	}

	err = o.AddServer("github", server, adapter.ScopeLocal, map[string]string{"GITHUB_TOKEN": "test123"})
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	// Read and verify config
	data, _ := os.ReadFile("opencode.json")
	var cfg OpenCodeConfig
	json.Unmarshal(data, &cfg)

	srv, exists := cfg.MCP["github"]
	if !exists {
		t.Fatal("server 'github' not found in config")
	}

	if srv.Type != "local" {
		t.Errorf("expected type 'local', got '%s'", srv.Type)
	}

	expectedCmd := []string{"npx", "-y", "@modelcontextprotocol/server-github"}
	if len(srv.Command) != len(expectedCmd) {
		t.Fatalf("expected command length %d, got %d", len(expectedCmd), len(srv.Command))
	}
	for i := range expectedCmd {
		if srv.Command[i] != expectedCmd[i] {
			t.Errorf("expected command[%d]='%s', got '%s'", i, expectedCmd[i], srv.Command[i])
		}
	}

	if srv.Environment["GITHUB_TOKEN"] != "test123" {
		t.Errorf("expected GITHUB_TOKEN='test123', got '%s'", srv.Environment["GITHUB_TOKEN"])
	}

	if !srv.Enabled {
		t.Error("expected Enabled to be true")
	}
}

func TestAddServer_HttpTransport(t *testing.T) {
	o := New()

	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	server := &config.Server{
		Transport: "http",
		URL:       "https://example.com/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	}

	err = o.AddServer("remote-server", server, adapter.ScopeLocal, nil)
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	data, _ := os.ReadFile("opencode.json")
	var cfg OpenCodeConfig
	json.Unmarshal(data, &cfg)

	srv := cfg.MCP["remote-server"]

	if srv.Type != "remote" {
		t.Errorf("expected type 'remote', got '%s'", srv.Type)
	}

	if srv.URL != "https://example.com/mcp" {
		t.Errorf("expected URL 'https://example.com/mcp', got '%s'", srv.URL)
	}

	if srv.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("expected Authorization header, got '%s'", srv.Headers["Authorization"])
	}
}

func TestAddServer_MergesExistingConfig(t *testing.T) {
	o := New()

	tmpDir, err := os.MkdirTemp("", "opencode-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create initial config
	initialCfg := OpenCodeConfig{
		Schema: "https://opencode.ai/config.json",
		MCP: map[string]OpenCodeMCPServer{
			"existing": {
				Type:    "local",
				Command: []string{"node", "existing.js"},
				Enabled: true,
			},
		},
	}
	data, _ := json.MarshalIndent(initialCfg, "", "  ")
	os.WriteFile("opencode.json", data, 0644)

	// Add new server
	server := &config.Server{
		Transport: "stdio",
		Command:   "node",
		Args:      []string{"new.js"},
	}

	err = o.AddServer("new-server", server, adapter.ScopeLocal, nil)
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	// Verify both servers exist
	data, _ = os.ReadFile("opencode.json")
	var cfg OpenCodeConfig
	json.Unmarshal(data, &cfg)

	if _, exists := cfg.MCP["existing"]; !exists {
		t.Error("existing server was removed")
	}

	if _, exists := cfg.MCP["new-server"]; !exists {
		t.Error("new server was not added")
	}
}

func TestDryRun_StdioTransport(t *testing.T) {
	o := New()

	server := &config.Server{
		Transport: "stdio",
		Command:   "npx",
		Args:      []string{"-y", "package"},
	}

	output := o.DryRun("test-server", server, adapter.ScopeLocal, map[string]string{"KEY": "value"})

	if output == "" {
		t.Error("DryRun returned empty string")
	}

	// Verify it contains expected elements
	if !strings.Contains(output, "opencode.json") {
		t.Error("expected output to contain config path")
	}
	if !strings.Contains(output, "test-server") {
		t.Error("expected output to contain server name")
	}
	if !strings.Contains(output, "local") {
		t.Error("expected output to contain type 'local'")
	}
}

func TestDryRun_HttpTransport(t *testing.T) {
	o := New()

	server := &config.Server{
		Transport: "http",
		URL:       "https://example.com/mcp",
	}

	output := o.DryRun("remote", server, adapter.ScopeLocal, nil)

	if !strings.Contains(output, "remote") {
		t.Error("expected output to contain type 'remote'")
	}
	if !strings.Contains(output, "https://example.com/mcp") {
		t.Error("expected output to contain URL")
	}
}
