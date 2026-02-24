package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// RunOption configures the TUI runner
type RunOption func(*runConfig)

// runConfig holds configuration for a TUI run
type runConfig struct {
	scope       adapter.Scope
	multiSelect bool
	altScreen   bool
}

// WithScope sets the installation scope
func WithScope(scope adapter.Scope) RunOption {
	return func(cfg *runConfig) {
		cfg.scope = scope
	}
}

// WithMultiSelect enables multi-select mode
func WithMultiSelect() RunOption {
	return func(cfg *runConfig) {
		cfg.multiSelect = true
	}
}

// WithAltScreen enables or disables alternate screen mode
func WithAltScreen(enabled bool) RunOption {
	return func(cfg *runConfig) {
		cfg.altScreen = enabled
	}
}

// Runner handles TUI execution
type Runner struct {
	config  *config.Config
	adapter adapter.Adapter
}

// NewRunner creates a new TUI runner
func NewRunner(cfg *config.Config, adptr adapter.Adapter) *Runner {
	return &Runner{
		config:  cfg,
		adapter: adptr,
	}
}

// Run starts the TUI with the given options
func (r *Runner) Run(opts ...RunOption) ([]string, error) {
	// Apply default configuration
	cfg := &runConfig{
		scope:       adapter.ScopeUser,
		multiSelect: false,
		altScreen:   true,
	}

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Create model with options
	m := NewModelWithOptions(r.config, r.adapter, cfg.scope, cfg.multiSelect)
	if m.allInstalled {
		fmt.Println("All servers are already installed.")
		return nil, nil
	}

	// Create program with or without alt screen
	var p *tea.Program
	if cfg.altScreen {
		p = tea.NewProgram(m, tea.WithAltScreen())
	} else {
		p = tea.NewProgram(m)
	}

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	// In single-select mode, return nil (no selection tracking)
	if !cfg.multiSelect {
		return nil, nil
	}

	// In multi-select mode, extract installed servers
	// Handle different model types - the model transitions from Model
	// to BulkConfigureModel when user presses Enter with selections
	switch result := finalModel.(type) {
	case Model:
		// User quit without selecting (Esc/q)
		return nil, nil
	case BulkConfigureModel:
		// Extract successfully installed servers from results
		var installed []string
		for _, r := range result.results {
			if r.success {
				installed = append(installed, r.serverName)
			}
		}
		return installed, nil
	default:
		return nil, nil
	}
}
