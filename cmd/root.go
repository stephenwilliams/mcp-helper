package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

var (
	// Version information (set via ldflags)
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"

	cfgFile string
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

	// Set version template to include build info
	rootCmd.SetVersionTemplate(fmt.Sprintf("mcp-helper %s (commit: %s, built: %s)\n", Version, Commit, BuildDate))
}

func initConfig() error {
	if cfgFile != "" {
		// Use config file from flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Use XDG-compliant config paths
		configPath, err := config.GetConfigDir()
		if err != nil {
			return fmt.Errorf("failed to determine config directory: %w", err)
		}

		// Search in current directory first, then XDG config
		viper.AddConfigPath(".")
		viper.AddConfigPath(configPath)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Environment variable support
	viper.SetEnvPrefix("MCP")
	viper.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	return nil
}
