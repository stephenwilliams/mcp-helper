package cmd

import (
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// resolveScope determines the scope to use based on flag value and config defaults.
// Priority: flagValue > cfg.DefaultScope > "local"
func resolveScope(flagValue string, cfg *config.Config) string {
	if flagValue != "" {
		return flagValue
	}
	if cfg.DefaultScope != "" {
		return cfg.DefaultScope
	}
	return "local"
}
