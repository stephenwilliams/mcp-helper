package permissions

import (
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/mcp"
)

func TestRegister(t *testing.T) {
	// Register a test adapter
	testName := "test-adapter-unique"
	Register(testName, func() Adapter {
		return &mockAdapter{name: testName}
	})

	// Verify it can be retrieved
	adapter, err := Get(testName)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if adapter.Name() != testName {
		t.Errorf("Get() returned adapter with name %v, want %v", adapter.Name(), testName)
	}
}

func TestGet_CaseInsensitive(t *testing.T) {
	testName := "TestAdapter"
	Register(testName, func() Adapter {
		return &mockAdapter{name: testName}
	})

	tests := []string{
		"testadapter",
		"TESTADAPTER",
		"TestAdapter",
		"tEsTaDaPtEr",
	}

	for _, name := range tests {
		adapter, err := Get(name)
		if err != nil {
			t.Errorf("Get(%q) failed: %v", name, err)
			continue
		}
		if adapter == nil {
			t.Errorf("Get(%q) returned nil adapter", name)
		}
	}
}

func TestGet_NotFound(t *testing.T) {
	_, err := Get("nonexistent-adapter-xyz")
	if err == nil {
		t.Error("Get() with nonexistent adapter should return error")
	}
}

func TestGetWithDefault(t *testing.T) {
	defaultName := "default-test-adapter"
	Register(defaultName, func() Adapter {
		return &mockAdapter{name: defaultName}
	})

	tests := []struct {
		name        string
		flagName    string
		defaultName string
		wantName    string
		wantErr     bool
	}{
		{
			name:        "use flag name",
			flagName:    defaultName,
			defaultName: "",
			wantName:    defaultName,
			wantErr:     false,
		},
		{
			name:        "use default when flag empty",
			flagName:    "",
			defaultName: defaultName,
			wantName:    defaultName,
			wantErr:     false,
		},
		{
			name:        "both empty - error",
			flagName:    "",
			defaultName: "",
			wantName:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := GetWithDefault(tt.flagName, tt.defaultName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetWithDefault() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && adapter.Name() != tt.wantName {
				t.Errorf("GetWithDefault() name = %v, want %v", adapter.Name(), tt.wantName)
			}
		})
	}
}

func TestList(t *testing.T) {
	// Register some test adapters
	names := []string{"test-list-1", "test-list-2", "test-list-3"}
	for _, name := range names {
		n := name // capture loop variable
		Register(n, func() Adapter {
			return &mockAdapter{name: n}
		})
	}

	list := List()

	// Verify all registered adapters are in the list
	found := make(map[string]bool)
	for _, name := range list {
		found[name] = true
	}

	for _, name := range names {
		if !found[name] {
			t.Errorf("List() missing adapter: %v", name)
		}
	}
}

// Note: TestClaudeCodeAdapterRegistered is in claudecode_test.go to avoid import cycles

// mockAdapter is a simple test adapter
type mockAdapter struct {
	name string
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) GetMCPServers() ([]mcp.ServerConfig, error) {
	return nil, nil
}
func (m *mockAdapter) GetSettingsPaths() []SettingsPath                          { return nil }
func (m *mockAdapter) LoadPermissions(path string) ([]PermissionRule, error)     { return nil, nil }
func (m *mockAdapter) SavePermissions(path string, rules []PermissionRule) error { return nil }
func (m *mockAdapter) FormatToolRule(serverName, toolName string) PermissionRule {
	return FormatMCPToolRule(serverName, toolName)
}
func (m *mockAdapter) FormatWildcardRule(serverName string) PermissionRule {
	return FormatMCPWildcardRule(serverName)
}
