package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// Run starts the TUI application
func Run(cfg *config.Config, adptr adapter.Adapter) error {
	m := NewModel(cfg, adptr)
	if m.allInstalled {
		fmt.Println("All servers are already installed.")
		return nil
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunFuzzySelect launches the multi-select TUI for bulk server installation
func RunFuzzySelect(cfg *config.Config, adptr adapter.Adapter, scope adapter.Scope) ([]string, error) {
	// Use the enhanced Model with multi-select mode enabled
	m := NewModelWithOptions(cfg, adptr, scope, true)
	if m.allInstalled {
		fmt.Println("All servers are already installed.")
		return nil, nil
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

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
