package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// Run starts the TUI application
func Run(cfg *config.Config, adptr adapter.Adapter) error {
	p := tea.NewProgram(NewModel(cfg, adptr), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
