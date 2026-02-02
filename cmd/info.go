package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/config"
	"github.com/stephenwilliams/mcp-helper/internal/template"
)

var infoJSONFlag bool

var infoCmd = &cobra.Command{
	Use:               "info <name>",
	Short:             "Display detailed information about an MCP server",
	Long:              `Display full configuration details for a specific MCP server including transport, command, environment variables, and metadata.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: ServerNameCompletion,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]

		cfg := GetConfig()
		if cfg == nil {
			return fmt.Errorf("no configuration found")
		}

		server, err := cfg.GetServer(serverName)
		if err != nil {
			return err
		}

		// Process templates so users see resolved values
		tmplData := template.NewTemplateData()
		processedServer, err := template.ProcessServer(server, tmplData)
		if err != nil {
			return fmt.Errorf("failed to process templates: %w", err)
		}

		if infoJSONFlag {
			return printJSON(serverName, processedServer)
		}

		return printHuman(serverName, processedServer)
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().BoolVar(&infoJSONFlag, "json", false, "Output in JSON format")
}

func printJSON(name string, server *config.Server) error {
	output := map[string]interface{}{
		"name":        name,
		"description": server.Description,
		"transport":   server.Transport,
	}

	if server.Transport == "stdio" {
		output["command"] = server.Command
		if len(server.Args) > 0 {
			output["args"] = server.Args
		}
	} else if server.Transport == "http" {
		output["url"] = server.URL
	}

	if len(server.Env) > 0 {
		output["env"] = server.Env
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func printHuman(name string, server *config.Server) error {
	fmt.Printf("Server: %s\n", name)

	if server.Description != "" {
		fmt.Printf("Description: %s\n", server.Description)
	}

	fmt.Printf("Transport: %s\n", server.Transport)

	if server.Transport == "stdio" {
		cmdLine := server.Command
		if len(server.Args) > 0 {
			cmdLine += " " + strings.Join(server.Args, " ")
		}
		fmt.Printf("Command: %s\n", cmdLine)
	} else if server.Transport == "http" {
		fmt.Printf("URL: %s\n", server.URL)
	}

	if len(server.Env) > 0 {
		fmt.Println("\nEnvironment Variables:")
		for key, envVar := range server.Env {
			required := ""
			if envVar.Required {
				required = " (required)"
			}
			fmt.Printf("  %s%s\n", key, required)
			if envVar.Description != "" {
				fmt.Printf("    %s\n", envVar.Description)
			}
			if envVar.Default != "" {
				fmt.Printf("    Default: %s\n", envVar.Default)
			}
		}
	}

	return nil
}
