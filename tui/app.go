package tui

import (
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// Run starts the TUI application
// Deprecated: Use NewRunner and Runner.Run for more flexibility
func Run(cfg *config.Config, adptr adapter.Adapter) error {
	runner := NewRunner(cfg, adptr)
	_, err := runner.Run()
	return err
}

// RunFuzzySelect launches the multi-select TUI for bulk server installation
// Deprecated: Use NewRunner and Runner.Run with WithMultiSelect and WithScope options
func RunFuzzySelect(cfg *config.Config, adptr adapter.Adapter, scope adapter.Scope) ([]string, error) {
	runner := NewRunner(cfg, adptr)
	return runner.Run(WithMultiSelect(), WithScope(scope))
}
