package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// ServerNameCompletion provides completion for server names from config.
// Note: This always uses default config discovery. The --config flag
// is not available during completion (shell limitation).
func ServerNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete first argument
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Use GetConfig which respects cfgFile and caching
	completionCfg := GetConfig()

	// If cfg is nil and cfgFile is set, try to load from that path
	if completionCfg == nil && cfgFile != "" {
		completionCfg, _ = config.LoadFromPath(cfgFile)
	}
	// Note: We don't fallback to config.Load() here because:
	// 1. In tests with cfg=nil and cfgFile="", we want to return empty results
	// 2. In normal operation, cfg should already be loaded via PersistentPreRunE

	if completionCfg == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get server names
	servers := completionCfg.ListServers()

	// Add preset names with p: prefix
	presets := completionCfg.ListPresets()
	for _, preset := range presets {
		servers = append(servers, "p:"+preset)
	}

	return servers, cobra.ShellCompDirectiveNoFileComp
}

// ScopeCompletion provides completion for --scope flag values
func ScopeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"local", "user", "project"}, cobra.ShellCompDirectiveNoFileComp
}
