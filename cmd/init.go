package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

var (
	initForce bool
	initPath  string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize mcp-helper configuration",
	Long: `Initialize mcp-helper by creating a sample configuration file.

The configuration file will be created at ~/.config/mcp-helper/config.yaml
by default, or at a custom location if --path is specified.

Use --force to overwrite an existing configuration file.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing config file")
	initCmd.Flags().StringVar(&initPath, "path", "", "custom config file path")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine target config path
	var configPath string
	var err error

	if initPath != "" {
		configPath = initPath
	} else {
		configPath, err = config.GetConfigFilePath()
		if err != nil {
			return fmt.Errorf("failed to determine config path: %w", err)
		}
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil && !initForce {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", configPath)
	}

	// Ensure directory exists
	if err := config.EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Sample config content
	sampleConfig := `# MCP Helper Configuration
# Documentation: https://github.com/stephenwilliams/mcp-helper

# Default scope for 'add' command (local, user, project)
default_scope: local

# Define your MCP servers here
servers:
  # Example: GitHub server with required token
  # github:
  #   description: "GitHub API integration"
  #   transport: stdio
  #   command: npx
  #   args: ["-y", "@modelcontextprotocol/server-github"]
  #   env:
  #     GITHUB_PERSONAL_ACCESS_TOKEN:
  #       required: true
  #       description: "GitHub PAT with repo access"
`

	// Write config file
	if err := os.WriteFile(configPath, []byte(sampleConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration file created at: %s\n", configPath)
	return nil
}
