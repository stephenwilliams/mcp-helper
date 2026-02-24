package tui

import (
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// mockAdapter implements adapter.Adapter for testing
type mockAdapter struct {
	existingServers map[string]bool
}

func (m *mockAdapter) Name() string {
	return "mock"
}

func (m *mockAdapter) AddServer(name string, server *config.Server, scope adapter.Scope, env map[string]string) error {
	return nil
}

func (m *mockAdapter) DryRun(name string, server *config.Server, scope adapter.Scope, env map[string]string) string {
	return ""
}

func (m *mockAdapter) GetConfigPath(scope adapter.Scope) string {
	return ""
}

func (m *mockAdapter) ServerExists(name string, scope adapter.Scope) bool {
	return m.existingServers[name]
}

// TestNewModelWithOptions_NoServersInstalled verifies that when no servers are installed,
// all servers from the config are included in the model.
func TestNewModelWithOptions_NoServersInstalled(t *testing.T) {
	// Setup: Config with 3 servers, none installed
	cfg := &config.Config{
		Servers: map[string]*config.Server{
			"server-a": {Description: "Server A", Transport: "stdio"},
			"server-b": {Description: "Server B", Transport: "stdio"},
			"server-c": {Description: "Server C", Transport: "stdio"},
		},
	}

	mock := &mockAdapter{
		existingServers: map[string]bool{}, // None installed
	}

	// Execute
	m := NewModelWithOptions(cfg, mock, adapter.ScopeUser, false)

	// Assert: All 3 servers should be in the list
	if len(m.servers) != 3 {
		t.Errorf("Expected 3 servers, got %d", len(m.servers))
	}

	// Assert: allInstalled should be false
	if m.allInstalled {
		t.Error("Expected allInstalled to be false when no servers are installed")
	}

	// Verify servers are sorted and present
	expectedServers := []string{"server-a", "server-b", "server-c"}
	for i, expected := range expectedServers {
		if m.servers[i] != expected {
			t.Errorf("Expected server at index %d to be %s, got %s", i, expected, m.servers[i])
		}
	}
}

// TestNewModelWithOptions_SomeServersInstalled verifies that installed servers
// are filtered out from the available server list.
func TestNewModelWithOptions_SomeServersInstalled(t *testing.T) {
	// Setup: Config with 3 servers, one installed
	cfg := &config.Config{
		Servers: map[string]*config.Server{
			"server-a": {Description: "Server A", Transport: "stdio"},
			"server-b": {Description: "Server B", Transport: "stdio"},
			"server-c": {Description: "Server C", Transport: "stdio"},
		},
	}

	mock := &mockAdapter{
		existingServers: map[string]bool{
			"server-b": true, // Only server-b is installed
		},
	}

	// Execute
	m := NewModelWithOptions(cfg, mock, adapter.ScopeUser, false)

	// Assert: Only 2 servers should be in the list (a and c)
	if len(m.servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(m.servers))
	}

	// Assert: server-b should NOT be in the list
	for _, name := range m.servers {
		if name == "server-b" {
			t.Error("Expected server-b to be filtered out (already installed)")
		}
	}

	// Assert: server-a and server-c should be present
	expectedServers := map[string]bool{
		"server-a": false,
		"server-c": false,
	}
	for _, name := range m.servers {
		if _, ok := expectedServers[name]; !ok {
			t.Errorf("Unexpected server in list: %s", name)
		}
		expectedServers[name] = true
	}
	for name, found := range expectedServers {
		if !found {
			t.Errorf("Expected server %s to be in list but was not found", name)
		}
	}

	// Assert: allInstalled should be false
	if m.allInstalled {
		t.Error("Expected allInstalled to be false when some servers are not installed")
	}
}

// TestNewModelWithOptions_AllServersInstalled verifies that when all servers
// are already installed, the server list is empty and allInstalled is true.
func TestNewModelWithOptions_AllServersInstalled(t *testing.T) {
	// Setup: Config with 3 servers, all installed
	cfg := &config.Config{
		Servers: map[string]*config.Server{
			"server-a": {Description: "Server A", Transport: "stdio"},
			"server-b": {Description: "Server B", Transport: "stdio"},
			"server-c": {Description: "Server C", Transport: "stdio"},
		},
	}

	mock := &mockAdapter{
		existingServers: map[string]bool{
			"server-a": true,
			"server-b": true,
			"server-c": true, // All installed
		},
	}

	// Execute
	m := NewModelWithOptions(cfg, mock, adapter.ScopeUser, false)

	// Assert: Server list should be empty
	if len(m.servers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(m.servers))
	}

	// Assert: allInstalled should be true
	if !m.allInstalled {
		t.Error("Expected allInstalled to be true when all servers are installed")
	}
}

// TestNewModelWithOptions_EmptyConfig verifies behavior when the config has no servers.
func TestNewModelWithOptions_EmptyConfig(t *testing.T) {
	// Setup: Config with no servers
	cfg := &config.Config{
		Servers: map[string]*config.Server{}, // Empty
	}

	mock := &mockAdapter{
		existingServers: map[string]bool{},
	}

	// Execute
	m := NewModelWithOptions(cfg, mock, adapter.ScopeUser, false)

	// Assert: Server list should be empty
	if len(m.servers) != 0 {
		t.Errorf("Expected 0 servers, got %d", len(m.servers))
	}

	// Assert: allInstalled should be false (registry is empty, not "all installed")
	if m.allInstalled {
		t.Error("Expected allInstalled to be false when registry is empty")
	}
}
