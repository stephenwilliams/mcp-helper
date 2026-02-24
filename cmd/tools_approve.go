package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/permissions"
	"github.com/stephenwilliams/mcp-helper/tui"
)

var (
	toolsApproveDryRun bool
	toolsApproveTarget string
)

var toolsApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Interactively pre-approve MCP tools",
	Long: `Launch an interactive TUI to select MCP tools for pre-approval.

Selected tools will be added to the permissions.allow array in settings.json,
eliminating permission prompts during agent sessions.

Permission format: mcp__<server>__<tool> or mcp__<server>__* (wildcard)

Examples:
  # Launch interactive approval TUI
  mcp-helper tools approve

  # Preview changes without applying
  mcp-helper tools approve --dry-run

  # Write to specific settings file
  mcp-helper tools approve --target .claude/settings.local.json`,
	RunE: runToolsApprove,
}

func init() {
	toolsCmd.AddCommand(toolsApproveCmd)
	toolsApproveCmd.Flags().BoolVar(&toolsApproveDryRun, "dry-run", false, "Preview changes without applying")
	toolsApproveCmd.Flags().StringVar(&toolsApproveTarget, "target", "", "Target settings file path")
}

func runToolsApprove(cmd *cobra.Command, args []string) error {
	// Get the permissions adapter (default to claudecode)
	adapter, err := permissions.GetWithDefault("", "claudecode")
	if err != nil {
		return fmt.Errorf("failed to get permissions adapter: %w", err)
	}

	// Create the TUI model
	model := tui.NewToolsApproveModel(adapter, toolsApproveDryRun, toolsApproveTarget)

	// Run the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if there was an error in the final model
	if m, ok := finalModel.(tui.ToolsApproveModel); ok {
		if m.Err != nil {
			return m.Err
		}
	}

	return nil
}
