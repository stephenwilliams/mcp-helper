package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/env"
	"github.com/stephenwilliams/mcp-helper/internal/template"
	"github.com/stephenwilliams/mcp-helper/tui"
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
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: ServerNameCompletion,
	RunE:              runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVar(&addScope, "scope", "", "Configuration scope: local, user, or project (default from config or 'local')")
	addCmd.Flags().StringSliceVarP(&addEnvVars, "env", "e", nil, "Environment variable (KEY=VALUE, repeatable)")
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "Show command without executing")
	addCmd.Flags().BoolVar(&addNoPrompt, "no-prompt", false, "Fail if env vars missing instead of prompting")

	// Register completions
	addCmd.RegisterFlagCompletionFunc("scope", ScopeCompletion)
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Get configuration first (needed for both paths)
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration file found")
	}

	// NO ARGS: Launch fuzzy select TUI
	if len(args) == 0 {
		// Determine scope
		scope := addScope
		if scope == "" {
			if cfg.DefaultScope != "" {
				scope = cfg.DefaultScope
			} else {
				scope = "local"
			}
		}
		parsedScope, err := adapter.ParseScope(scope)
		if err != nil {
			return err
		}

		// Get adapter
		adptr, err := GetAdapter()
		if err != nil {
			return err
		}

		// Launch TUI (handles everything including installation)
		installed, err := tui.RunFuzzySelect(cfg, adptr, parsedScope)
		if err != nil {
			return err
		}
		if len(installed) == 0 {
			fmt.Println("No servers installed.")
		}
		return nil
	}

	// SINGLE ARG: Check if it's a preset (p: prefix)
	serverName := args[0]

	// Handle preset expansion
	if strings.HasPrefix(serverName, "p:") {
		presetName := strings.TrimPrefix(serverName, "p:")
		presetServers, err := cfg.ExpandPreset(presetName)
		if err != nil {
			return fmt.Errorf("preset '%s' not found in configuration", presetName)
		}

		// Determine scope
		scope := addScope
		if scope == "" {
			if cfg.DefaultScope != "" {
				scope = cfg.DefaultScope
			} else {
				scope = "local"
			}
		}
		parsedScope, err := adapter.ParseScope(scope)
		if err != nil {
			return err
		}

		// Get adapter
		adptr, err := GetAdapter()
		if err != nil {
			return err
		}

		// Filter out already-installed servers
		var availableServers []string
		for _, s := range presetServers {
			if !adptr.ServerExists(s, parsedScope) {
				availableServers = append(availableServers, s)
			}
		}

		if len(availableServers) == 0 {
			fmt.Println("All servers in preset are already installed.")
			return nil
		}

		// Handle --dry-run: print dry-run output for each server (no TUI)
		if addDryRun {
			for _, srvName := range availableServers {
				server := cfg.Servers[srvName]
				// Parse --env flags
				providedEnv, err := parseEnvFlags(addEnvVars)
				if err != nil {
					return err
				}
				// Process templates
				tmplData := template.NewTemplateData()
				processedServer, err := template.ProcessServer(server, tmplData)
				if err != nil {
					return fmt.Errorf("failed to process templates for %s: %w", srvName, err)
				}
				// Collect env vars (non-interactive for dry-run)
				collectedEnv, err := env.CollectEnvVars(processedServer, providedEnv, false)
				if err != nil {
					return fmt.Errorf("failed to collect env vars for %s: %w", srvName, err)
				}
				fmt.Printf("--- %s ---\n", srvName)
				fmt.Println(adptr.DryRun(srvName, processedServer, parsedScope, collectedEnv))
			}
			return nil
		}

		// Handle --no-prompt: install all non-interactively
		if addNoPrompt {
			providedEnv, err := parseEnvFlags(addEnvVars)
			if err != nil {
				return err
			}
			for _, srvName := range availableServers {
				server := cfg.Servers[srvName]
				tmplData := template.NewTemplateData()
				processedServer, err := template.ProcessServer(server, tmplData)
				if err != nil {
					return fmt.Errorf("failed to process templates for %s: %w", srvName, err)
				}
				collectedEnv, err := env.CollectEnvVars(processedServer, providedEnv, false)
				if err != nil {
					return fmt.Errorf("server %s: %w", srvName, err)
				}
				if err := adptr.AddServer(srvName, processedServer, parsedScope, collectedEnv); err != nil {
					return fmt.Errorf("failed to install %s: %w", srvName, err)
				}
				fmt.Printf("Successfully added server '%s' to %s (scope: %s)\n", srvName, adptr.Name(), parsedScope)
			}
			return nil
		}

		// Single server in preset: use normal single-server flow
		if len(availableServers) == 1 {
			serverName = availableServers[0]
			// Fall through to existing single-server logic below
		} else {
			// Multiple servers: launch BulkConfigureModel TUI
			bulkModel := tui.NewBulkConfigureModel(availableServers, cfg, adptr, parsedScope)
			p := tea.NewProgram(bulkModel, tea.WithAltScreen())
			_, err = p.Run()
			return err
		}
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

	// Get adapter based on --agent flag or config default
	adptr, err := GetAdapter()
	if err != nil {
		return err
	}

	// Dry run or execute (server already processed, no template work needed in adapter)
	if addDryRun {
		dryRunOutput := adptr.DryRun(serverName, processedServer, parsedScope, collectedEnv)
		fmt.Println("Command that would be executed:")
		fmt.Println(dryRunOutput)
		return nil
	}

	// Execute
	if err := adptr.AddServer(serverName, processedServer, parsedScope, collectedEnv); err != nil {
		return err
	}

	fmt.Printf("Successfully added server '%s' to %s (scope: %s)\n", serverName, adptr.Name(), parsedScope)
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
