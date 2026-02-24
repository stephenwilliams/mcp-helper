// Package main provides a CLI tool for autonomous TUI exploration testing.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
	"github.com/stephenwilliams/mcp-helper/internal/tuiharness"
	"github.com/stephenwilliams/mcp-helper/tui"
)

var (
	// Exploration options
	seeds       = flag.Int("seeds", 1, "Number of exploration episodes to run")
	steps       = flag.Int("steps", 500, "Maximum steps per episode")
	timeout     = flag.Duration("timeout", 5*time.Minute, "Overall timeout for exploration")
	seed        = flag.Int64("seed", 0, "Starting seed (0 = random)")

	// Terminal size
	cols        = flag.Int("cols", 120, "Terminal columns")
	rows        = flag.Int("rows", 40, "Terminal rows")

	// Output options
	artifactDir = flag.String("artifacts", "./artifacts", "Directory for failure artifacts")
	verbose     = flag.Bool("verbose", false, "Verbose output")
	minimize    = flag.Bool("minimize", true, "Attempt to minimize failure repros")

	// Replay mode
	replayFile  = flag.String("replay", "", "Replay a repro file instead of exploring")

	// Config file
	configFile  = flag.String("config", "", "Path to MCP helper config file")
)

func main() {
	flag.Parse()

	if *replayFile != "" {
		if err := runReplay(); err != nil {
			fmt.Fprintf(os.Stderr, "Replay failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := runExploration(); err != nil {
		fmt.Fprintf(os.Stderr, "Exploration failed: %v\n", err)
		os.Exit(1)
	}
}

func runReplay() error {
	fmt.Printf("Replaying: %s\n", *replayFile)

	actions, err := tuiharness.ParseRepro(*replayFile)
	if err != nil {
		return fmt.Errorf("failed to parse repro file: %w", err)
	}

	fmt.Printf("Loaded %d actions\n", len(actions))

	h, err := tuiharness.Start(createModelFactory(), tuiharness.Options{
		Cols:        *cols,
		Rows:        *rows,
		WaitTimeout: 2 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to start harness: %w", err)
	}
	defer h.Stop()

	oracles := tuiharness.DefaultOracles()
	result, err := tuiharness.Replay(h, actions, oracles, 50*time.Millisecond)
	if err != nil {
		return fmt.Errorf("replay error: %w", err)
	}

	if result.Failure != nil {
		fmt.Println()
		fmt.Println("FAILURE REPRODUCED")
		fmt.Printf("  Step:        %d\n", result.FailureStep)
		fmt.Printf("  Oracle:      %s\n", result.Failure.OracleName)
		fmt.Printf("  Description: %s\n", result.Failure.Description)
		if result.Failure.Details != "" {
			fmt.Printf("  Details:     %s\n", result.Failure.Details)
		}
		return fmt.Errorf("failure reproduced")
	}

	fmt.Println()
	fmt.Println("No failure reproduced")
	return nil
}

func runExploration() error {
	startTime := time.Now()
	overallTimeout := time.After(*timeout)

	report := tuiharness.NewReport()

	baseSeed := *seed
	if baseSeed == 0 {
		baseSeed = time.Now().UnixNano()
	}

	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("                    TUI EXPLORER")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Episodes:    %d\n", *seeds)
	fmt.Printf("  Steps/ep:    %d\n", *steps)
	fmt.Printf("  Base seed:   %d\n", baseSeed)
	fmt.Printf("  Terminal:    %dx%d\n", *cols, *rows)
	fmt.Printf("  Artifacts:   %s\n", *artifactDir)
	fmt.Printf("  Minimize:    %v\n", *minimize)
	fmt.Println()

	failureFound := false

	for episode := 0; episode < *seeds; episode++ {
		select {
		case <-overallTimeout:
			fmt.Println("\nOverall timeout reached")
			break
		default:
		}

		currentSeed := baseSeed + int64(episode)
		fmt.Printf("Episode %d/%d (seed: %d)... ", episode+1, *seeds, currentSeed)

		h, err := tuiharness.Start(createModelFactory(), tuiharness.Options{
			Cols:        *cols,
			Rows:        *rows,
			WaitTimeout: 2 * time.Second,
		})
		if err != nil {
			fmt.Printf("FAILED TO START: %v\n", err)
			continue
		}

		explorerConfig := tuiharness.DefaultExplorerConfig()
		explorerConfig.Seed = currentSeed
		explorerConfig.MaxSteps = *steps
		explorerConfig.EnableMinimize = *minimize

		explorer := tuiharness.NewExplorer(explorerConfig)
		oracles := tuiharness.DefaultOracles()

		result := explorer.Run(h, oracles)
		h.Stop()

		if result.Failure != nil {
			fmt.Printf("FAILURE at step %d (%s)\n", result.FailureStep, result.Failure.OracleName)
			failureFound = true

			// Save artifacts
			artifactWriter := tuiharness.NewArtifactWriter(*artifactDir)
			artifactPath, err := artifactWriter.SaveFailure(result)
			if err != nil {
				fmt.Printf("  Warning: failed to save artifacts: %v\n", err)
			} else {
				fmt.Printf("  Artifacts: %s\n", artifactPath)
				report.AddResult(result, artifactPath)

				// Attempt minimization
				if *minimize && len(result.Steps) > 1 {
					fmt.Printf("  Minimizing...")
					minimizer := tuiharness.NewMinimizer(createModelFactory(), oracles, tuiharness.Options{
						Cols: *cols,
						Rows: *rows,
					})
					if err := minimizer.MinimizeAndSave(artifactPath, result.Steps[:result.FailureStep+1]); err != nil {
						fmt.Printf(" failed: %v\n", err)
					} else {
						fmt.Printf(" done\n")
					}
				}
			}
		} else {
			fmt.Printf("OK (%d steps, %d states)\n", result.TotalSteps, result.UniqueStates)
			report.AddResult(result, "")
		}

		if *verbose {
			fmt.Printf("  Duration: %v\n", result.Duration)
			fmt.Printf("  Stuck events: %d\n", result.StuckEvents)
		}
	}

	// Print report
	report.Write(os.Stdout)

	elapsed := time.Since(startTime)
	fmt.Printf("\nTotal time: %v\n", elapsed)

	if failureFound {
		fmt.Println("\nExiting with code 1 (failures found)")
		os.Exit(1)
	}

	fmt.Println("\nExiting with code 0 (no failures)")
	return nil
}

func createModelFactory() tuiharness.ModelFactory {
	cfg := loadConfig()
	mock := &mockAdapter{existingServers: map[string]bool{}}
	return func() tea.Model {
		return tui.NewModelWithOptions(cfg, mock, adapter.ScopeUser, true)
	}
}

func loadConfig() *config.Config {
	if *configFile != "" {
		cfg, err := config.LoadFromPath(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load config %s: %v\n", *configFile, err)
		} else {
			return cfg
		}
	}

	// Default test config
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
			"server-delta": {
				Description: "Delta server",
				Transport:   "stdio",
				Command:     "delta-server",
			},
			"server-epsilon": {
				Description: "Epsilon server",
				Transport:   "stdio",
				Command:     "epsilon-server",
			},
		},
		Presets: map[string]*config.Preset{
			"basic": {
				Description: "Basic preset with alpha and gamma",
				Servers:     []string{"server-alpha", "server-gamma"},
			},
			"full": {
				Description: "All servers",
				Servers:     []string{"server-alpha", "server-beta", "server-gamma", "server-delta", "server-epsilon"},
			},
		},
	}
}

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
