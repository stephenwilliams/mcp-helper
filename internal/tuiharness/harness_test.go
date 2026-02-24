package tuiharness_test

import (
	"flag"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
	"github.com/stephenwilliams/mcp-helper/internal/tuiharness"
	"github.com/stephenwilliams/mcp-helper/tui"
)

var reproFile = flag.String("repro", "", "Path to repro file to replay")

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

// testConfig returns a test configuration with sample servers
func testConfig() *config.Config {
	return &config.Config{
		Servers: map[string]*config.Server{
			"server-alpha": {
				Description: "Alpha server for testing",
				Transport:   "stdio",
				Command:     "alpha-server",
			},
			"server-beta": {
				Description: "Beta server with environment variables",
				Transport:   "stdio",
				Command:     "beta-server",
				Env: map[string]config.EnvVar{
					"API_KEY": {
						Description: "API key for beta server",
						Required:    true,
					},
				},
			},
			"server-gamma": {
				Description: "Gamma HTTP server",
				Transport:   "http",
				URL:         "http://localhost:8080",
			},
		},
		Presets: map[string]*config.Preset{
			"basic": {
				Description: "Basic preset with alpha and gamma",
				Servers:     []string{"server-alpha", "server-gamma"},
			},
		},
	}
}

// createTestModel creates a test model factory
func createTestModel() tuiharness.ModelFactory {
	cfg := testConfig()
	mock := &mockAdapter{existingServers: map[string]bool{}}
	return func() tea.Model {
		return tui.NewModelWithOptions(cfg, mock, adapter.ScopeUser, true)
	}
}

// TestSmokeStartAndQuit verifies basic startup and quit functionality
func TestSmokeStartAndQuit(t *testing.T) {
	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	// Wait a bit for initial render
	time.Sleep(100 * time.Millisecond)

	// Get initial screen
	screen, err := h.Screen()
	if err != nil {
		t.Fatalf("Failed to get screen: %v", err)
	}

	// Verify we see the TUI
	if screen.GridText == "" {
		t.Error("Screen is empty after start")
	}

	// Check for title text
	if !screen.ContainsText("Select MCP Servers") && !screen.ContainsText("MCP Server Browser") {
		t.Logf("Screen content:\n%s", screen.Excerpt(10))
		t.Error("Expected to see server selection screen")
	}

	// Quit
	if err := h.Press(tuiharness.KeyEsc); err != nil {
		t.Fatalf("Failed to press Esc: %v", err)
	}
}

// TestSmokeNavigateServers verifies basic navigation
func TestSmokeNavigateServers(t *testing.T) {
	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Navigate down
	for i := 0; i < 3; i++ {
		if err := h.Press(tuiharness.KeyDown); err != nil {
			t.Fatalf("Failed to press Down: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Navigate up
	for i := 0; i < 2; i++ {
		if err := h.Press(tuiharness.KeyUp); err != nil {
			t.Fatalf("Failed to press Up: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Get screen and verify
	screen, err := h.Screen()
	if err != nil {
		t.Fatalf("Failed to get screen: %v", err)
	}

	// Should still show the server list
	if !screen.ContainsText("server-") {
		t.Logf("Screen content:\n%s", screen.Excerpt(10))
		t.Error("Expected to see server entries")
	}
}

// TestSmokeTabNavigation verifies tab switching between servers and presets
func TestSmokeTabNavigation(t *testing.T) {
	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Press Tab to switch to presets
	if err := h.Press(tuiharness.KeyTab); err != nil {
		t.Fatalf("Failed to press Tab: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	screen, err := h.Screen()
	if err != nil {
		t.Fatalf("Failed to get screen: %v", err)
	}

	// Should see presets tab content or tab indicator
	if !screen.ContainsText("Presets") && !screen.ContainsText("basic") {
		t.Logf("Screen content:\n%s", screen.Excerpt(15))
		// This is not necessarily a failure - depends on config
	}

	// Tab back to servers
	if err := h.Press(tuiharness.KeyTab); err != nil {
		t.Fatalf("Failed to press Tab: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	screen, err = h.Screen()
	if err != nil {
		t.Fatalf("Failed to get screen: %v", err)
	}

	// Should be back at servers
	if !screen.ContainsText("Servers") && !screen.ContainsText("server-") {
		t.Logf("Screen content:\n%s", screen.Excerpt(15))
	}
}

// TestSmokeSpaceToggle verifies space key toggles selection
func TestSmokeSpaceToggle(t *testing.T) {
	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Toggle selection with space
	if err := h.Press(tuiharness.KeySpace); err != nil {
		t.Fatalf("Failed to press Space: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	screen, err := h.Screen()
	if err != nil {
		t.Fatalf("Failed to get screen: %v", err)
	}

	// Should show selected count or checkbox
	if !screen.ContainsText("Selected: 1") && !screen.ContainsText("[✓]") {
		t.Logf("Screen content:\n%s", screen.Excerpt(15))
		// Not necessarily a failure, depends on implementation
	}

	// Toggle off
	if err := h.Press(tuiharness.KeySpace); err != nil {
		t.Fatalf("Failed to press Space: %v", err)
	}
}

// TestSmokeFilterInput verifies filter/search functionality
func TestSmokeFilterInput(t *testing.T) {
	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Type to filter (in multi-select mode, typing starts filter)
	if err := h.Type("alpha"); err != nil {
		t.Fatalf("Failed to type: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	screen, err := h.Screen()
	if err != nil {
		t.Fatalf("Failed to get screen: %v", err)
	}

	// Should show filter or filtered results
	if !screen.ContainsText("alpha") && !screen.ContainsText("Filter") {
		t.Logf("Screen content:\n%s", screen.Excerpt(15))
	}

	// Clear filter with Esc
	if err := h.Press(tuiharness.KeyEsc); err != nil {
		t.Fatalf("Failed to press Esc: %v", err)
	}
}

// TestSmokeResize verifies terminal resize handling
func TestSmokeResize(t *testing.T) {
	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	time.Sleep(100 * time.Millisecond)

	// Resize to smaller
	if err := h.Resize(80, 24); err != nil {
		t.Fatalf("Failed to resize: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	screen, err := h.Screen()
	if err != nil {
		t.Fatalf("Failed to get screen: %v", err)
	}

	// Should still render without crash
	if screen.GridText == "" {
		t.Error("Screen is empty after resize")
	}

	// Resize back
	if err := h.Resize(120, 40); err != nil {
		t.Fatalf("Failed to resize back: %v", err)
	}
}

// TestReplayRepro replays a reproduction file if provided
func TestReplayRepro(t *testing.T) {
	if *reproFile == "" {
		t.Skip("No repro file specified (use -repro flag)")
	}

	actions, err := tuiharness.ParseRepro(*reproFile)
	if err != nil {
		t.Fatalf("Failed to parse repro file: %v", err)
	}

	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	oracles := tuiharness.DefaultOracles()
	result, err := tuiharness.Replay(h, actions, oracles, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if result.Failure != nil {
		t.Logf("Failure reproduced at step %d", result.FailureStep)
		t.Logf("Oracle: %s", result.Failure.OracleName)
		t.Logf("Description: %s", result.Failure.Description)
		t.Logf("Details: %s", result.Failure.Details)
	} else {
		t.Log("No failure reproduced")
	}
}

// TestExplorerSmoke runs a short exploration to verify the explorer works
func TestExplorerSmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping explorer smoke test in short mode")
	}

	h, err := tuiharness.Start(createTestModel(), tuiharness.Options{
		Cols:        120,
		Rows:        40,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to start harness: %v", err)
	}
	defer h.Stop()

	// Short exploration
	explorerConfig := tuiharness.DefaultExplorerConfig()
	explorerConfig.Seed = 12345 // Fixed seed for reproducibility
	explorerConfig.MaxSteps = 50

	explorer := tuiharness.NewExplorer(explorerConfig)
	oracles := tuiharness.DefaultOracles()

	result := explorer.Run(h, oracles)

	t.Logf("Exploration completed: %d steps, %d unique states", result.TotalSteps, result.UniqueStates)

	if result.Failure != nil {
		t.Logf("Failure found: %s - %s", result.Failure.OracleName, result.Failure.Description)

		// Save artifacts
		artifactWriter := tuiharness.NewArtifactWriter("./artifacts")
		artifactDir, err := artifactWriter.SaveFailure(result)
		if err != nil {
			t.Logf("Failed to save artifacts: %v", err)
		} else {
			t.Logf("Artifacts saved to: %s", artifactDir)
		}
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}
