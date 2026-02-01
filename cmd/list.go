package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured MCP servers",
	Long:  `Display all MCP servers configured in the configuration file.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output in JSON format")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg := GetConfig()

	// Handle no servers configured
	if cfg == nil || len(cfg.Servers) == 0 {
		if listJSON {
			output := map[string]interface{}{
				"servers":     []interface{}{},
				"count":       0,
				"config_path": getConfigPath(),
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(output)
		}

		fmt.Println("No servers configured.")
		fmt.Printf("\nConfiguration file: %s\n", getConfigPath())
		return nil
	}

	// JSON output
	if listJSON {
		type ServerJSON struct {
			Name        string `json:"name"`
			Transport   string `json:"transport"`
			Description string `json:"description"`
		}

		servers := make([]ServerJSON, 0, len(cfg.Servers))
		for name, server := range cfg.Servers {
			servers = append(servers, ServerJSON{
				Name:        name,
				Transport:   server.Transport,
				Description: server.Description,
			})
		}

		// Sort by name for consistent output
		sort.Slice(servers, func(i, j int) bool {
			return servers[i].Name < servers[j].Name
		})

		output := map[string]interface{}{
			"servers":     servers,
			"count":       len(servers),
			"config_path": getConfigPath(),
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTRANSPORT\tDESCRIPTION")

	// Sort server names for consistent output
	names := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		server := cfg.Servers[name]
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, server.Transport, server.Description)
	}

	w.Flush()

	// Footer with count and path
	count := len(cfg.Servers)
	plural := "server"
	if count != 1 {
		plural = "servers"
	}
	fmt.Printf("\n%d %s configured in %s\n", count, plural, getConfigPath())

	return nil
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	// Use the same default path logic as config package
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = home + "/.config"
	}
	return configDir + "/mcp-helper/config.yaml"
}
