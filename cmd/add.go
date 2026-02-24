package cmd

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/app"
	"github.com/stephenwilliams/mcp-helper/internal/config"
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
	_ = addCmd.RegisterFlagCompletionFunc("scope", ScopeCompletion)
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Get configuration first (needed for both paths)
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration file found")
	}

	// NO ARGS: Launch fuzzy select TUI
	if len(args) == 0 {
		return runAddInteractive(cfg)
	}

	// SINGLE ARG: Check if it's a preset (p: prefix)
	serverName := args[0]
	if strings.HasPrefix(serverName, "p:") {
		presetName := strings.TrimPrefix(serverName, "p:")
		return runAddPreset(cfg, presetName)
	}

	// Single server
	return runAddSingle(cfg, serverName)
}

// runAddInteractive launches the fuzzy select TUI for server selection
func runAddInteractive(cfg *config.Config) error {
	// Determine scope
	parsedScope, err := adapter.ParseScope(resolveScope(addScope, cfg))
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

// runAddPreset handles preset expansion and installation
func runAddPreset(cfg *config.Config, presetName string) error {
	// Expand preset
	presetServers, err := expandPreset(cfg, presetName)
	if err != nil {
		return err
	}

	// Determine scope
	parsedScope, err := adapter.ParseScope(resolveScope(addScope, cfg))
	if err != nil {
		return err
	}

	// Get adapter
	adptr, err := GetAdapter()
	if err != nil {
		return err
	}

	// Filter out already-installed servers
	availableServers := filterAvailableServers(presetServers, adptr, parsedScope)
	if len(availableServers) == 0 {
		fmt.Println("All servers in preset are already installed.")
		return nil
	}

	// Handle --dry-run
	if addDryRun {
		return presetDryRun(cfg, availableServers, adptr, parsedScope)
	}

	// Handle --no-prompt
	if addNoPrompt {
		return presetNoPrompt(cfg, availableServers, adptr, parsedScope)
	}

	// Single server: use normal single-server flow
	if len(availableServers) == 1 {
		return runAddSingle(cfg, availableServers[0])
	}

	// Multiple servers: launch interactive TUI
	return presetInteractive(cfg, availableServers, adptr, parsedScope)
}

// expandPreset expands a preset name into a list of server names
func expandPreset(cfg *config.Config, presetName string) ([]string, error) {
	presetServers, err := cfg.ExpandPreset(presetName)
	if err != nil {
		return nil, fmt.Errorf("preset '%s' not found in configuration", presetName)
	}
	return presetServers, nil
}

// filterAvailableServers filters out already-installed servers
func filterAvailableServers(servers []string, adptr adapter.Adapter, scope adapter.Scope) []string {
	var available []string
	for _, s := range servers {
		if !adptr.ServerExists(s, scope) {
			available = append(available, s)
		}
	}
	return available
}

// presetDryRun prints dry-run output for all servers in a preset
func presetDryRun(cfg *config.Config, servers []string, adptr adapter.Adapter, scope adapter.Scope) error {
	for _, srvName := range servers {
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
		fmt.Println(adptr.DryRun(srvName, processedServer, scope, collectedEnv))
	}
	return nil
}

// presetNoPrompt installs all servers in a preset non-interactively
func presetNoPrompt(cfg *config.Config, servers []string, adptr adapter.Adapter, scope adapter.Scope) error {
	providedEnv, err := parseEnvFlags(addEnvVars)
	if err != nil {
		return err
	}
	installer := app.NewServerInstaller(cfg, adptr)
	for _, srvName := range servers {
		if err := installServerNonInteractive(cfg, installer, srvName, scope, providedEnv, adptr); err != nil {
			return err
		}
	}
	return nil
}

// installServerNonInteractive installs a single server without prompting
func installServerNonInteractive(cfg *config.Config, installer app.ServerInstaller, srvName string, scope adapter.Scope, providedEnv map[string]string, adptr adapter.Adapter) error {
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
	req := app.ServerInstallRequest{
		ServerName: srvName,
		Scope:      scope,
		EnvValues:  collectedEnv,
	}
	resp := installer.Install(context.Background(), req)
	if resp.Error != nil {
		return fmt.Errorf("failed to install %s: %w", srvName, resp.Error)
	}
	fmt.Printf("Successfully added server '%s' to %s (scope: %s)\n", srvName, adptr.Name(), scope)
	return nil
}

// presetInteractive launches the BulkConfigureModel TUI for multiple servers
func presetInteractive(cfg *config.Config, servers []string, adptr adapter.Adapter, scope adapter.Scope) error {
	bulkModel := tui.NewBulkConfigureModel(servers, cfg, adptr, scope)
	p := tea.NewProgram(bulkModel, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// runAddSingle handles adding a single server
func runAddSingle(cfg *config.Config, serverName string) error {
	// Find server in config
	server, exists := cfg.Servers[serverName]
	if !exists {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	// Process templates BEFORE collecting env vars
	tmplData := template.NewTemplateData()
	processedServer, err := template.ProcessServer(server, tmplData)
	if err != nil {
		return fmt.Errorf("failed to process templates: %w", err)
	}

	// Determine scope
	parsedScope, err := adapter.ParseScope(resolveScope(addScope, cfg))
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

	// Create installer service
	installer := app.NewServerInstaller(cfg, adptr)

	// Dry run or execute
	if addDryRun {
		req := app.ServerInstallRequest{
			ServerName: serverName,
			Scope:      parsedScope,
			EnvValues:  collectedEnv,
			DryRun:     true,
		}
		dryRunOutput := installer.DryRun(req)
		fmt.Println("Command that would be executed:")
		fmt.Println(dryRunOutput)
		return nil
	}

	// Execute
	req := app.ServerInstallRequest{
		ServerName: serverName,
		Scope:      parsedScope,
		EnvValues:  collectedEnv,
	}
	resp := installer.Install(context.Background(), req)
	if resp.Error != nil {
		return resp.Error
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
