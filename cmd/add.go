package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/adapter/claudecode"
	"github.com/stephenwilliams/mcp-helper/internal/env"
	"github.com/stephenwilliams/mcp-helper/internal/template"
)

var (
	addScope    string
	addEnvVars  []string
	addDryRun   bool
	addNoPrompt bool
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add an MCP server to Claude Code",
	Long: `Add an MCP server to Claude Code configuration.

The server must be defined in the mcp-helper configuration file.
You can provide environment variables via --env flags or be prompted interactively.

Examples:
  # Add a server with interactive prompts for env vars
  mcp-helper add github

  # Add a server with a specific scope
  mcp-helper add github --scope project

  # Add a server with environment variables
  mcp-helper add github --env GITHUB_TOKEN=ghp_xxx

  # Preview the command without executing it
  mcp-helper add github --dry-run

  # Fail if env vars are missing (no prompts)
  mcp-helper add github --no-prompt`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVar(&addScope, "scope", "", "Configuration scope: local, user, or project (default from config or 'local')")
	addCmd.Flags().StringSliceVarP(&addEnvVars, "env", "e", nil, "Environment variable (KEY=VALUE, repeatable)")
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "Show command without executing")
	addCmd.Flags().BoolVar(&addNoPrompt, "no-prompt", false, "Fail if env vars missing instead of prompting")
}

func runAdd(cmd *cobra.Command, args []string) error {
	serverName := args[0]

	// Get configuration
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration file found")
	}

	// Find server in config
	server, exists := cfg.Servers[serverName]
	if !exists {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	// Process templates BEFORE collecting env vars
	// This ensures templated EnvVar.Default values are resolved
	// before the priority system in CollectEnvVars uses them
	tmplData := template.NewTemplateData()
	processedServer, err := template.ProcessServer(server, tmplData)
	if err != nil {
		return fmt.Errorf("failed to process templates: %w", err)
	}

	// Determine scope
	scope := addScope
	if scope == "" {
		// Use default from config, or fallback to "local"
		if cfg.DefaultScope != "" {
			scope = cfg.DefaultScope
		} else {
			scope = "local"
		}
	}

	// Parse scope
	parsedScope, err := adapter.ParseScope(scope)
	if err != nil {
		return err
	}

	// Parse --env flags into provided map
	providedEnv, err := parseEnvFlags(addEnvVars)
	if err != nil {
		return err
	}

	// Collect environment variables (uses processedServer with resolved defaults)
	interactive := !addNoPrompt && !addDryRun
	collectedEnv, err := env.CollectEnvVars(processedServer, providedEnv, interactive)
	if err != nil {
		return err
	}

	// Create adapter
	ccAdapter := claudecode.New()

	// Dry run or execute (server already processed, no template work needed in adapter)
	if addDryRun {
		dryRunOutput := ccAdapter.DryRun(serverName, processedServer, parsedScope, collectedEnv)
		fmt.Println("Command that would be executed:")
		fmt.Println(dryRunOutput)
		return nil
	}

	// Execute
	if err := ccAdapter.AddServer(serverName, processedServer, parsedScope, collectedEnv); err != nil {
		return err
	}

	fmt.Printf("Successfully added server '%s' to %s (scope: %s)\n", serverName, ccAdapter.Name(), parsedScope)
	return nil
}

// parseEnvFlags parses --env flag values (KEY=VALUE) into a map.
func parseEnvFlags(envFlags []string) (map[string]string, error) {
	result := make(map[string]string)

	for _, flag := range envFlags {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --env flag format: '%s' (expected KEY=VALUE)", flag)
		}

		key := strings.TrimSpace(parts[0])
		value := parts[1] // Don't trim value, preserve spaces

		if key == "" {
			return nil, fmt.Errorf("invalid --env flag: key cannot be empty")
		}

		result[key] = value
	}

	return result, nil
}
