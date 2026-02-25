package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/mcp"
	"github.com/stephenwilliams/mcp-helper/internal/permissions"
)

var (
	toolsListJSON    bool
	toolsListScope   string
	toolsListServer  string
	toolsListNoCache bool
)

var toolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tools from configured MCP servers",
	Long: `Discover and list tools from all configured MCP servers.

Tools are read from the agent's MCP configuration.
For Claude Code: $CLAUDE_CONFIG_DIR/.claude.json (defaults to ~/.claude/.claude.json).
Environment variables are sourced from that same config.

By default, results are cached for 1 hour. Use --no-cache to bypass the cache.

Examples:
  # List all tools in table format
  mcp-helper tools list

  # Output as JSON for scripting
  mcp-helper tools list --json

  # List tools from a specific server
  mcp-helper tools list --server github

  # Filter by scope
  mcp-helper tools list --scope user

  # Bypass cache and fetch fresh
  mcp-helper tools list --no-cache`,
	RunE: runToolsList,
}

func init() {
	toolsCmd.AddCommand(toolsListCmd)
	toolsListCmd.Flags().BoolVar(&toolsListJSON, "json", false, "Output as JSON")
	toolsListCmd.Flags().StringVar(&toolsListScope, "scope", "", "Filter by scope (local, user, project)")
	toolsListCmd.Flags().StringVar(&toolsListServer, "server", "", "List tools from specific server only")
	toolsListCmd.Flags().BoolVar(&toolsListNoCache, "no-cache", false, "Bypass cache and fetch fresh")
}

func runToolsList(cmd *cobra.Command, args []string) error {
	// Get permissions adapter (default to claudecode)
	adapter, err := permissions.GetWithDefault(agentName, "claudecode")
	if err != nil {
		return fmt.Errorf("failed to get permissions adapter: %w", err)
	}

	// Get MCP server configurations
	servers, err := adapter.GetMCPServers()
	if err != nil {
		return fmt.Errorf("failed to get MCP servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Println("No MCP servers configured.")
		if !toolsListJSON {
			fmt.Printf("\nConfigure MCP servers in your agent's configuration file.\n")
			fmt.Printf("For Claude Code: $CLAUDE_CONFIG_DIR/.claude.json (defaults to ~/.claude/.claude.json)\n")
		}
		return nil
	}

	// Apply filters
	var filteredServers []mcp.ServerConfig
	for _, srv := range servers {
		// Filter by scope
		if toolsListScope != "" && srv.Scope != toolsListScope {
			continue
		}
		// Filter by server name
		if toolsListServer != "" && srv.Name != toolsListServer {
			continue
		}
		filteredServers = append(filteredServers, srv)
	}

	if len(filteredServers) == 0 {
		if toolsListJSON {
			// Output empty JSON
			output := map[string]interface{}{
				"servers": []interface{}{},
				"cached":  false,
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(output)
		}
		fmt.Println("No servers match the specified filters.")
		return nil
	}

	// Create MCP client with cache
	cache := mcp.NewCache(mcp.DefaultCacheTTL)
	client := mcp.NewClient(10*time.Second, cache)

	// Discover tools from all servers
	ctx := context.Background()
	useCache := !toolsListNoCache
	results := client.DiscoverTools(ctx, filteredServers, useCache)

	// Output results
	if toolsListJSON {
		return outputToolsListJSON(results, useCache)
	}
	return outputToolsTable(results)
}

// outputToolsTable prints results in table format
func outputToolsTable(results []mcp.ServerInfo) error {
	// Print summary table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "SERVER\tSCOPE\tTOOLS\tSTATUS")

	for _, result := range results {
		status := "ok"
		toolCount := fmt.Sprintf("%d", len(result.Tools))
		if result.Error != nil {
			status = fmt.Sprintf("error: %s", result.Error.Error())
			toolCount = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", result.Name, result.Scope, toolCount, status)
	}
	_ = w.Flush()

	// Print tool details
	fmt.Println("\nTools:")
	hasTools := false
	for _, result := range results {
		if len(result.Tools) > 0 {
			hasTools = true
			for _, tool := range result.Tools {
				desc := tool.Description
				if desc == "" {
					desc = "(no description)"
				}
				// Truncate long descriptions
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				fmt.Printf("  %s.%s - %s\n", result.Name, tool.Name, desc)
			}
		}
	}

	if !hasTools {
		fmt.Println("  (no tools discovered)")
	}

	return nil
}

// outputToolsListJSON prints results as JSON (renamed to avoid conflict with aws.go)
func outputToolsListJSON(results []mcp.ServerInfo, cached bool) error {
	// Convert to JSON-friendly format
	type jsonTool struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description,omitempty"`
		InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
	}

	type jsonServer struct {
		Name      string     `json:"name"`
		Scope     string     `json:"scope"`
		Transport string     `json:"transport"`
		Tools     []jsonTool `json:"tools,omitempty"`
		Error     string     `json:"error,omitempty"`
	}

	servers := make([]jsonServer, len(results))
	for i, result := range results {
		srv := jsonServer{
			Name:      result.Name,
			Scope:     result.Scope,
			Transport: result.Transport,
		}

		if result.Error != nil {
			srv.Error = result.Error.Error()
		} else {
			srv.Tools = make([]jsonTool, len(result.Tools))
			for j, tool := range result.Tools {
				srv.Tools[j] = jsonTool{
					Name:        tool.Name,
					Description: tool.Description,
					InputSchema: tool.InputSchema,
				}
			}
		}

		servers[i] = srv
	}

	output := map[string]interface{}{
		"servers": servers,
		"cached":  cached,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}
