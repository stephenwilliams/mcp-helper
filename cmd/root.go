package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/adapter/claudecode"
	"github.com/stephenwilliams/mcp-helper/internal/config"
	"github.com/stephenwilliams/mcp-helper/tui"
)

var (
	// Version information (set via ldflags)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"

	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:     "mcp-helper",
	Short:   "A helper tool for MCP server management",
	Long:    `mcp-helper is a CLI tool for managing and configuring MCP servers.`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $XDG_CONFIG_HOME/mcp-helper/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(browseCmd)

	// Set version template to include build info
	rootCmd.SetVersionTemplate(fmt.Sprintf("mcp-helper %s (commit: %s, built: %s)\n", Version, Commit, BuildDate))
}

func initConfig() error {
	var err error

	// Load configuration using config.Load() or config.LoadFromPath()
	if cfgFile != "" {
		cfg, err = config.LoadFromPath(cfgFile)
		if err != nil {
			return err
		}
	} else {
		cfg, err = config.Load()
		if err != nil {
			return err
		}
	}

	// cfg can be nil if no config file was found (this is not an error)
	return nil
}

// GetConfig returns the loaded configuration.
// May return nil if no configuration file was found.
func GetConfig() *config.Config {
	return cfg
}

var browseCmd = &cobra.Command{
	Use:     "browse",
	Aliases: []string{"interactive", "ui"},
	Short:   "Browse and install MCP servers interactively",
	Long:    `Launch an interactive TUI to browse available MCP servers, view their details, configure them, and install them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no configuration found: please run 'mcp-helper init' first")
		}

		// Create the adapter for installing servers
		adapter := claudecode.New()

		// Launch the TUI
		return tui.Run(cfg, adapter)
	},
}
