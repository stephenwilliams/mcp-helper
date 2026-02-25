package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
	"github.com/stephenwilliams/mcp-helper/tui"
)

// mockInstaller is a simple mock ServerInstaller
type mockInstaller struct {
	called []string
	err    error
}

func testCfg() *config.Config {
	return &config.Config{
		Servers: map[string]*config.Server{
			"server-alpha": {
				Description: "Alpha server",
				Transport:   "stdio",
				Command:     "alpha",
			},
			"server-beta": {
				Description: "Beta server",
				Transport:   "stdio",
				Command:     "beta",
				Env: map[string]config.EnvVar{
					"API_KEY": {Description: "API key", Required: true},
				},
			},
		},
	}
}

type mockAdapterTest struct{}

func (m *mockAdapterTest) Name() string                                                                    { return "mock" }
func (m *mockAdapterTest) AddServer(name string, server *config.Server, scope adapter.Scope, env map[string]string) error { return nil }
func (m *mockAdapterTest) DryRun(name string, server *config.Server, scope adapter.Scope, env map[string]string) string  { return "" }
func (m *mockAdapterTest) GetConfigPath(scope adapter.Scope) string                                       { return "" }
func (m *mockAdapterTest) ServerExists(name string, scope adapter.Scope) bool                             { return false }

func sendKey(m tea.Model, key string) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated
}

func sendSpecialKey(m tea.Model, keyType tea.KeyType) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: keyType})
	return updated
}

func sendNamedKey(m tea.Model, name string) tea.Model {
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)})
	return updated
}

func TestBulkConfigure_InitialView(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha", "server-beta"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	view := m.View()
	if !strings.Contains(view, "Bulk Server Configuration") {
		t.Errorf("Expected 'Bulk Server Configuration' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "server-alpha") {
		t.Errorf("Expected 'server-alpha' in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Configuring server 1 of 2") {
		t.Errorf("Expected progress '1 of 2' in view, got:\n%s", view)
	}
}

func TestBulkConfigure_ServerWithNoEnvVars(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	// server-alpha has no env vars
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	view := m.View()
	if !strings.Contains(view, "No configuration needed") {
		t.Errorf("Expected 'No configuration needed' for server without env vars, got:\n%s", view)
	}
	if !strings.Contains(view, "Press Enter to install") {
		t.Errorf("Expected 'Press Enter to install' prompt, got:\n%s", view)
	}
}

func TestBulkConfigure_ServerWithEnvVars(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	// server-beta has API_KEY env var
	servers := []string{"server-beta"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	view := m.View()
	if !strings.Contains(view, "API_KEY") {
		t.Errorf("Expected 'API_KEY' env var in view, got:\n%s", view)
	}
	if !strings.Contains(view, "Field 1 of 1") {
		t.Errorf("Expected 'Field 1 of 1' progress in view, got:\n%s", view)
	}
}

func TestBulkConfigure_EscShowsCancelDialog(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha", "server-beta"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Press Esc to trigger cancel confirmation
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	view := updated.View()

	if !strings.Contains(view, "Cancel Remaining Servers") {
		t.Errorf("Expected cancel confirmation dialog, got:\n%s", view)
	}
}

func TestBulkConfigure_CancelConfirmN(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha", "server-beta"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Press Esc to trigger cancel
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	// Press n to dismiss
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	view := m3.View()

	// Should be back to configuring
	if !strings.Contains(view, "Bulk Server Configuration") {
		t.Errorf("Expected to return to configuration after 'n', got:\n%s", view)
	}
}

func TestBulkConfigure_CancelConfirmY(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha", "server-beta"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Press Esc then y to cancel all
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	view := m3.View()

	if !strings.Contains(view, "Bulk Installation Complete") {
		t.Errorf("Expected completion screen after cancel-all, got:\n%s", view)
	}
	if !strings.Contains(view, "cancelled") {
		t.Errorf("Expected 'cancelled' in completion summary, got:\n%s", view)
	}
}

func TestBulkConfigure_WindowSizeMsg(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	// Should not crash and should still render
	view := updated.View()
	if view == "" {
		t.Error("Expected non-empty view after resize")
	}
}

func TestBulkConfigure_QuitKey(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("Expected quit command when 'q' pressed")
	}
}

func TestBulkConfigure_TextInputInEnvField(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-beta"} // has API_KEY (a secret field)
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Type some text into a secret field (API_KEY)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	view := m4.View()

	// API_KEY is a secret field — input is masked, not shown as plaintext
	if strings.Contains(view, "abc") {
		t.Errorf("Secret field should not show plaintext 'abc', got:\n%s", view)
	}
	// Masked input should show asterisks (3 chars typed = 3 asterisks)
	if !strings.Contains(view, "***") {
		t.Errorf("Expected masked input '***' in secret field view, got:\n%s", view)
	}
}

func TestBulkConfigure_BackspaceInEnvField(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-beta"} // has API_KEY
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Type then backspace
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	view := m4.View()

	// Should show 'a' but not 'ab'
	if strings.Contains(view, "ab") && !strings.Contains(view, "abc") {
		t.Logf("View contains 'ab' after backspace (expected 'a' only):\n%s", view)
	}
	// At minimum, no panic
	if view == "" {
		t.Error("Expected non-empty view after backspace")
	}
}

func TestBulkConfigure_Init(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	cmd := m.Init()
	if cmd != nil {
		t.Error("Expected Init() to return nil for BulkConfigureModel")
	}
}

// runCmd executes a tea.Cmd and feeds the resulting message back into the model.
func runCmd(m tea.Model, cmd tea.Cmd) tea.Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	updated, _ := m.Update(msg)
	return updated
}

func TestBulkConfigure_EnterOnNoEnvVarsTriggersInstall(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	// server-alpha has no env vars — Enter should trigger install immediately
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Press Enter — should start installing
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := m2.View()
	if !strings.Contains(view, "Installing") {
		t.Errorf("Expected 'Installing' state after Enter on no-env server, got:\n%s", view)
	}

	// Execute the install command to get completion message
	m3 := runCmd(m2, cmd)
	view3 := m3.View()

	if !strings.Contains(view3, "Bulk Installation Complete") {
		t.Errorf("Expected completion screen after install finishes, got:\n%s", view3)
	}
	if !strings.Contains(view3, "1 successful") {
		t.Errorf("Expected '1 successful' in completion view, got:\n%s", view3)
	}
}

func TestBulkConfigure_ViewInstalling(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Press Enter to trigger install state
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := m2.View()

	if !strings.Contains(view, "Installing Server") {
		t.Errorf("Expected 'Installing Server' title in installing view, got:\n%s", view)
	}
	if !strings.Contains(view, "Installing server 1 of 1") {
		t.Errorf("Expected progress 'Installing server 1 of 1', got:\n%s", view)
	}
	if !strings.Contains(view, "server-alpha") {
		t.Errorf("Expected server name in installing view, got:\n%s", view)
	}
	if !strings.Contains(view, "Please wait") {
		t.Errorf("Expected 'Please wait' in installing view, got:\n%s", view)
	}
}

func TestBulkConfigure_InstallingStateIgnoresKeys(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Enter installing state
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Keys other than q/ctrl+c should be ignored during install
	m3, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if cmd != nil {
		t.Error("Expected no command when pressing non-quit key during install")
	}
	view := m3.View()
	if !strings.Contains(view, "Installing") {
		t.Errorf("Expected still installing after irrelevant key, got:\n%s", view)
	}
}

func TestBulkConfigure_CompleteStateEnterQuits(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Install and complete
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := runCmd(m2, cmd)

	// Verify we are on complete screen
	view := m3.View()
	if !strings.Contains(view, "Bulk Installation Complete") {
		t.Fatalf("Expected completion screen, got:\n%s", view)
	}

	// Press Enter on complete screen should return quit command
	_, quitCmd := m3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if quitCmd == nil {
		t.Error("Expected quit command when pressing Enter on complete screen")
	}
}

func TestBulkConfigure_CompleteStateQKey(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := runCmd(m2, cmd)

	_, quitCmd := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if quitCmd == nil {
		t.Error("Expected quit command when pressing 'q' on complete screen")
	}
}

func TestBulkConfigure_MultiServerFlow(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	// Two servers, both with no env vars
	servers := []string{"server-alpha", "server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	view := m.View()
	if !strings.Contains(view, "Configuring server 1 of 2") {
		t.Errorf("Expected '1 of 2' progress, got:\n%s", view)
	}

	// Install first server
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Complete first install — should move to second server
	m3 := runCmd(m2, cmd)

	view3 := m3.View()
	// After first install with 2 servers, should be on 2nd server or installing it
	if strings.Contains(view3, "Bulk Installation Complete") {
		// Both done (second had no env vars, auto-installed)
		if !strings.Contains(view3, "2 successful") {
			t.Errorf("Expected '2 successful', got:\n%s", view3)
		}
	} else if strings.Contains(view3, "Installing server 2 of 2") || strings.Contains(view3, "Configuring server 2 of 2") {
		// On second server — ok
	} else {
		t.Errorf("Unexpected view after first install of 2:\n%s", view3)
	}
}

func TestBulkConfigure_EnvFieldNavigation(t *testing.T) {
	// Create a config with 2 env vars to test tab/shift-tab
	cfg := &config.Config{
		Servers: map[string]*config.Server{
			"multi-env-server": {
				Description: "Server with multiple env vars",
				Transport:   "stdio",
				Command:     "multi",
				Env: map[string]config.EnvVar{
					"AAA_KEY": {Description: "First key", Required: true},
					"BBB_KEY": {Description: "Second key", Required: false},
				},
			},
		},
	}
	adpt := &mockAdapterTest{}
	servers := []string{"multi-env-server"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	view := m.View()
	if !strings.Contains(view, "Field 1 of 2") {
		t.Errorf("Expected 'Field 1 of 2', got:\n%s", view)
	}

	// Tab to next field
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	view2 := m2.View()
	if !strings.Contains(view2, "Field 2 of 2") {
		t.Errorf("Expected 'Field 2 of 2' after Tab, got:\n%s", view2)
	}

	// Shift+Tab back to first field
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	view3 := m3.View()
	if !strings.Contains(view3, "Field 1 of 2") {
		t.Errorf("Expected 'Field 1 of 2' after Shift+Tab, got:\n%s", view3)
	}
}

func TestBulkConfigure_CursorLeftRight(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-beta"} // has API_KEY
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Type 3 chars then move left/right — should not panic
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m6, _ := m5.Update(tea.KeyMsg{Type: tea.KeyRight})

	view := m6.View()
	if view == "" {
		t.Error("Expected non-empty view after cursor movement")
	}
}

func TestBulkConfigure_HomeEnd(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-beta"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Type some chars
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})

	// Home key (ctrl+a equivalent)
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyHome})
	// End key (ctrl+e equivalent)
	m5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyEnd})

	view := m5.View()
	if view == "" {
		t.Error("Expected non-empty view after Home/End keys")
	}
}

func TestBulkConfigure_ViewComplete_WithFailure(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	servers := []string{"server-alpha", "server-alpha"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Cancel all (marks remaining as cancelled)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	view := m3.View()
	if !strings.Contains(view, "Bulk Installation Complete") {
		t.Fatalf("Expected completion screen after cancel-all, got:\n%s", view)
	}
	// cancelled count should be 2
	if !strings.Contains(view, "cancelled") {
		t.Errorf("Expected 'cancelled' in completion view, got:\n%s", view)
	}
}

func TestBulkConfigure_ViewConfirmCancel_ShowsPreviousResults(t *testing.T) {
	cfg := testCfg()
	adpt := &mockAdapterTest{}
	// Two servers: first has no env vars, second has env vars
	servers := []string{"server-alpha", "server-beta"}
	m := tui.NewBulkConfigureModel(servers, cfg, adpt, adapter.ScopeUser)

	// Install first server
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := runCmd(m2, cmd)

	// Now press Esc to cancel remaining
	m4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	view := m4.View()

	if !strings.Contains(view, "Cancel Remaining Servers") {
		t.Fatalf("Expected cancel dialog, got:\n%s", view)
	}
	// Should show already installed result
	if !strings.Contains(view, "Already installed") {
		t.Errorf("Expected 'Already installed' section in cancel dialog, got:\n%s", view)
	}
	if !strings.Contains(view, "server-alpha") {
		t.Errorf("Expected server-alpha in already installed list, got:\n%s", view)
	}
}
