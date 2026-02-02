package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/adapter/claudecode"
	"github.com/stephenwilliams/mcp-helper/internal/aws"
	"github.com/stephenwilliams/mcp-helper/internal/env"
	"github.com/stephenwilliams/mcp-helper/internal/template"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	awsDiscoverAll    bool
	awsDiscoverScope  string
	awsDiscoverJSON   bool
	awsDiscoverDryRun bool
	awsDiscoverForce  bool
)

var awsCmd = &cobra.Command{
	Use:   "aws",
	Short: "AWS MCP server management",
	Long:  "Commands for managing AWS MCP servers via profile auto-discovery.",
}

var awsDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover and add AWS MCP profiles",
	Long: `Discover AWS profiles with -mcpro or -mcprw suffixes and add them to Claude Code.

Profiles are auto-detected from ~/.aws/config and ~/.aws/credentials files.
Use interactive TUI to select which profiles to add, or use --all to add all.

Examples:
  # Interactive selection
  mcp-helper aws discover

  # Add all discovered profiles
  mcp-helper aws discover --all

  # Preview without adding
  mcp-helper aws discover --dry-run

  # Output discovered profiles as JSON
  mcp-helper aws discover --json

  # Add to specific scope
  mcp-helper aws discover --scope user

  # Overwrite existing servers
  mcp-helper aws discover --force`,
	RunE: runAwsDiscover,
}

func init() {
	rootCmd.AddCommand(awsCmd)
	awsCmd.AddCommand(awsDiscoverCmd)

	awsDiscoverCmd.Flags().BoolVar(&awsDiscoverAll, "all", false, "Skip interactive selection and add all discovered profiles")
	awsDiscoverCmd.Flags().StringVar(&awsDiscoverScope, "scope", "", "Configuration scope: local, user, or project (default from config or 'user')")
	awsDiscoverCmd.Flags().BoolVar(&awsDiscoverJSON, "json", false, "Output discovered profiles in JSON format (non-interactive)")
	awsDiscoverCmd.Flags().BoolVar(&awsDiscoverDryRun, "dry-run", false, "Show what would be added without actually adding")
	awsDiscoverCmd.Flags().BoolVar(&awsDiscoverForce, "force", false, "Overwrite existing servers (default: skip with warning)")

	// Register completions
	awsDiscoverCmd.RegisterFlagCompletionFunc("scope", ScopeCompletion)
}

func runAwsDiscover(cmd *cobra.Command, args []string) error {
	// Create ProfileManager and discover MCP profiles
	pm := aws.NewProfileManager()
	profiles, err := pm.ListMCPProfiles()
	if err != nil {
		return fmt.Errorf("failed to discover AWS profiles: %w", err)
	}

	if len(profiles) == 0 {
		fmt.Println("No AWS MCP profiles found.")
		fmt.Println("\nAWS MCP profiles must have -mcpro (read-only) or -mcprw (read-write) suffix.")
		fmt.Println("Example profile names: dev-mcpro, prod-mcprw")
		return nil
	}

	// Pre-flight check: warn if uvx not in PATH
	if _, err := exec.LookPath("uvx"); err != nil {
		fmt.Println("⚠ Warning: 'uvx' command not found in PATH.")
		fmt.Println("  AWS MCP servers require uvx to be installed.")
		fmt.Println("  Install with: pip install uv")
		fmt.Println()
	}

	// Handle --json flag: output JSON and return
	if awsDiscoverJSON {
		return outputJSON(profiles)
	}

	// Handle --dry-run flag: show table and return
	if awsDiscoverDryRun {
		return outputDryRun(profiles)
	}

	// Determine scope
	scope := awsDiscoverScope
	if scope == "" {
		cfg := GetConfig()
		if cfg != nil && cfg.DefaultScope != "" {
			scope = cfg.DefaultScope
		} else {
			scope = "user"
		}
	}

	// Parse scope
	parsedScope, err := adapter.ParseScope(scope)
	if err != nil {
		return err
	}

	// Determine which profiles to add
	var profilesToAdd []aws.MCPProfile

	if awsDiscoverAll {
		// Add all profiles without TUI
		profilesToAdd = profiles
		fmt.Printf("Adding all %d discovered profiles...\n", len(profiles))
	} else {
		// Run interactive TUI
		model := aws.NewMultiSelectModel(profiles)
		p := tea.NewProgram(model)

		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		m := finalModel.(aws.MultiSelectModel)

		if m.WasCancelled() {
			fmt.Println("Cancelled.")
			return nil
		}

		profilesToAdd = m.SelectedProfiles()

		if len(profilesToAdd) == 0 {
			fmt.Println("No profiles selected.")
			return nil
		}
	}

	// Add selected profiles
	return addProfiles(profilesToAdd, parsedScope)
}

// outputJSON outputs profiles in JSON format
func outputJSON(profiles []aws.MCPProfile) error {
	type jsonProfile struct {
		Profile    string `json:"profile"`
		ServerName string `json:"serverName"`
		Region     string `json:"region"`
		Mode       string `json:"mode"`
		IsSSO      bool   `json:"isSSO"`
	}

	jsonProfiles := make([]jsonProfile, len(profiles))
	for i, profile := range profiles {
		jsonProfiles[i] = jsonProfile{
			Profile:    profile.Name,
			ServerName: aws.GenerateServerName(profile),
			Region:     profile.Region,
			Mode:       profile.Mode,
			IsSSO:      profile.IsSSO,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonProfiles)
}

// outputDryRun shows a table of what would be added
func outputDryRun(profiles []aws.MCPProfile) error {
	// Styles for table
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	roStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	rwStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	ssoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	fmt.Println(headerStyle.Render("Discovered AWS MCP Profiles"))
	fmt.Println()

	// Table header
	fmt.Printf("%-20s %-15s %-12s %s\n",
		headerStyle.Render("PROFILE"),
		headerStyle.Render("REGION"),
		headerStyle.Render("MODE"),
		headerStyle.Render("AUTH"),
	)
	fmt.Println(strings.Repeat("─", 60))

	// Table rows
	for _, profile := range profiles {
		var modeDisplay string
		if profile.Mode == "ro" {
			modeDisplay = roStyle.Render("read-only")
		} else {
			modeDisplay = rwStyle.Render("read-write")
		}

		authDisplay := "IAM"
		if profile.IsSSO {
			authDisplay = ssoStyle.Render("SSO")
		}

		fmt.Printf("%-20s %-15s %-12s %s\n",
			profile.Name,
			profile.Region,
			modeDisplay,
			authDisplay,
		)
	}

	fmt.Printf("\nTotal: %d profiles\n", len(profiles))

	// Count SSO profiles
	ssoCount := 0
	for _, profile := range profiles {
		if profile.IsSSO {
			ssoCount++
		}
	}

	if ssoCount > 0 {
		fmt.Printf("\nℹ %d SSO profiles detected. Ensure you've run: aws sso login --profile <profile>\n", ssoCount)
	}

	return nil
}

// addProfiles adds the selected profiles to Claude Code
func addProfiles(profiles []aws.MCPProfile, scope adapter.Scope) error {
	ccAdapter := claudecode.New()

	addedCount := 0
	skippedCount := 0
	errorCount := 0
	ssoProfiles := []string{}

	for _, profile := range profiles {
		serverName := aws.GenerateServerName(profile)
		server := aws.GenerateServer(profile)

		// Process templates
		tmplData := template.NewTemplateData()
		processedServer, err := template.ProcessServer(server, tmplData)
		if err != nil {
			fmt.Printf("⚠ Failed to process server '%s': %v\n", serverName, err)
			errorCount++
			continue
		}

		// Collect environment variables (AWS_PROFILE has default, so no prompting needed)
		collectedEnv, err := env.CollectEnvVars(processedServer, map[string]string{}, false)
		if err != nil {
			fmt.Printf("⚠ Failed to collect env vars for '%s': %v\n", serverName, err)
			errorCount++
			continue
		}

		// Check if server already exists
		if serverExists(ccAdapter, serverName, scope) && !awsDiscoverForce {
			fmt.Printf("⚠ Skipping '%s': server already exists (use --force to overwrite)\n", serverName)
			skippedCount++
			continue
		}

		// Add server
		if err := ccAdapter.AddServer(serverName, processedServer, scope, collectedEnv); err != nil {
			fmt.Printf("✗ Failed to add '%s': %v\n", serverName, err)
			errorCount++
			continue
		}

		fmt.Printf("✓ Added '%s' (%s, %s)\n", serverName, profile.Region, profile.Mode)
		addedCount++

		if profile.IsSSO {
			ssoProfiles = append(ssoProfiles, profile.Name)
		}
	}

	// Summary
	fmt.Println()
	fmt.Printf("Summary: %d added, %d skipped, %d errors\n", addedCount, skippedCount, errorCount)

	// SSO reminder
	if len(ssoProfiles) > 0 {
		fmt.Println()
		fmt.Println("ℹ SSO profiles detected. Ensure you've run:")
		for _, ssoProfile := range ssoProfiles {
			fmt.Printf("  aws sso login --profile %s\n", ssoProfile)
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("encountered %d errors while adding profiles", errorCount)
	}

	return nil
}

// serverExists checks if a server already exists in the configuration
func serverExists(ccAdapter *claudecode.ClaudeCode, serverName string, scope adapter.Scope) bool {
	// Read existing configuration
	configPath := ccAdapter.GetConfigPath(scope)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}

	// Parse existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var existingConfig claudecode.ClaudeConfig
	if err := json.Unmarshal(data, &existingConfig); err != nil {
		return false
	}

	// Check if server exists
	_, exists := existingConfig.MCPServers[serverName]
	return exists
}
