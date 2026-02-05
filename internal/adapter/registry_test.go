package adapter

import (
	"strings"
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// mockAdapter is a simple test adapter that implements the Adapter interface
type mockAdapter struct {
	name string
}

func (m *mockAdapter) Name() string { return m.name }

func (m *mockAdapter) AddServer(name string, server *config.Server, scope Scope, env map[string]string) error {
	return nil
}

func (m *mockAdapter) DryRun(name string, server *config.Server, scope Scope, env map[string]string) string {
	return ""
}

func (m *mockAdapter) GetConfigPath(scope Scope) string { return "" }

func (m *mockAdapter) ServerExists(name string, scope Scope) bool { return false }

func TestRegister(t *testing.T) {
	// Clear registry for test
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("test-adapter", func() Adapter { return &mockAdapter{name: "Test"} })

	if _, ok := registry["test-adapter"]; !ok {
		t.Error("expected adapter to be registered")
	}
}

func TestGet_Found(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("myagent", func() Adapter { return &mockAdapter{name: "MyAgent"} })

	adapter, err := Get("myagent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != "MyAgent" {
		t.Errorf("expected name 'MyAgent', got '%s'", adapter.Name())
	}
}

func TestGet_NotFound(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	_, err := Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent adapter")
	}
	if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("expected error to contain 'unknown agent', got: %v", err)
	}
}

func TestGet_CaseInsensitive(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("MyAgent", func() Adapter { return &mockAdapter{name: "MyAgent"} })

	// Should find regardless of case
	adapter, err := Get("MYAGENT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != "MyAgent" {
		t.Errorf("expected name 'MyAgent', got '%s'", adapter.Name())
	}

	adapter, err = Get("myagent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != "MyAgent" {
		t.Errorf("expected name 'MyAgent', got '%s'", adapter.Name())
	}
}

func TestList(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("agent1", func() Adapter { return &mockAdapter{} })
	Register("agent2", func() Adapter { return &mockAdapter{} })

	list := List()
	if len(list) != 2 {
		t.Errorf("expected 2 adapters, got %d", len(list))
	}

	// Verify both names are present
	found := make(map[string]bool)
	for _, name := range list {
		found[name] = true
	}
	if !found["agent1"] || !found["agent2"] {
		t.Error("expected both agent1 and agent2 in list")
	}
}

func TestGetWithDefault_FlagOverride(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("flag-agent", func() Adapter { return &mockAdapter{name: "FlagAgent"} })
	Register("config-agent", func() Adapter { return &mockAdapter{name: "ConfigAgent"} })

	// Flag should override config
	adapter, err := GetWithDefault("flag-agent", "config-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != "FlagAgent" {
		t.Errorf("expected 'FlagAgent', got '%s'", adapter.Name())
	}
}

func TestGetWithDefault_ConfigDefault(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("config-agent", func() Adapter { return &mockAdapter{name: "ConfigAgent"} })

	// Empty flag should use config default
	adapter, err := GetWithDefault("", "config-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != "ConfigAgent" {
		t.Errorf("expected 'ConfigAgent', got '%s'", adapter.Name())
	}
}

func TestGetWithDefault_Fallback(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("claudecode", func() Adapter { return &mockAdapter{name: "ClaudeCode"} })

	// Empty flag and config should use FallbackDefault ("claudecode")
	adapter, err := GetWithDefault("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != "ClaudeCode" {
		t.Errorf("expected 'ClaudeCode', got '%s'", adapter.Name())
	}
}

func TestGetWithDefault_AllEmpty_FallbackMissing(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	// Don't register the fallback default
	// This should return an error
	_, err := GetWithDefault("", "")
	if err == nil {
		t.Error("expected error when fallback default is not registered")
	}
	if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("expected error to contain 'unknown agent', got: %v", err)
	}
}

func TestRegister_Lowercase(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	// Register with mixed case
	Register("MixedCase", func() Adapter { return &mockAdapter{name: "MixedCase"} })

	// Should be stored in lowercase
	if _, ok := registry["mixedcase"]; !ok {
		t.Error("expected adapter to be stored in lowercase")
	}
}

func TestGet_ErrorMessage(t *testing.T) {
	originalRegistry := registry
	registry = make(map[string]func() Adapter)
	defer func() { registry = originalRegistry }()

	Register("agent1", func() Adapter { return &mockAdapter{} })
	Register("agent2", func() Adapter { return &mockAdapter{} })

	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	errMsg := err.Error()
	// Should mention the unknown agent name
	if !strings.Contains(errMsg, "nonexistent") {
		t.Errorf("error message should contain 'nonexistent', got: %s", errMsg)
	}
	// Should list available agents
	if !strings.Contains(errMsg, "available:") {
		t.Errorf("error message should list available agents, got: %s", errMsg)
	}
}

func TestFallbackDefault_Constant(t *testing.T) {
	// Verify the fallback default is set to expected value
	if FallbackDefault != "claudecode" {
		t.Errorf("expected FallbackDefault to be 'claudecode', got '%s'", FallbackDefault)
	}
}
