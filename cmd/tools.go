package cmd

import (
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Discover and manage MCP server tools",
	Long: `Commands for discovering tools from configured MCP servers and managing tool permissions.

MCP servers must be configured in the agent's configuration file (e.g., ~/.claude.json for Claude Code).
Tools are discovered by connecting to servers and querying their tool lists.`,
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
