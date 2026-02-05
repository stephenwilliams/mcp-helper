// Package opencode provides an adapter for configuring MCP servers in OpenCode.
// It uses direct configuration file manipulation to add servers.
package opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

func init() {
	adapter.Register("opencode", func() adapter.Adapter { return New() })
	adapter.Register("open-code", func() adapter.Adapter { return New() })
	adapter.Register("oc", func() adapter.Adapter { return New() })
}

// OpenCodeConfig represents the OpenCode configuration file structure
type OpenCodeConfig struct {
	Schema string                       `json:"$schema,omitempty"`
	MCP    map[string]OpenCodeMCPServer `json:"mcp"`
}

// OpenCodeMCPServer represents a single MCP server in OpenCode config
type OpenCodeMCPServer struct {
	Type        string            `json:"type"`                  // "local" or "remote"
	Command     []string          `json:"command,omitempty"`     // For local servers
	URL         string            `json:"url,omitempty"`         // For remote servers
	Headers     map[string]string `json:"headers,omitempty"`     // For remote servers
	Environment map[string]string `json:"environment,omitempty"` // Env vars (NOT "env")
	Enabled     bool              `json:"enabled"`
}

// OpenCode implements the Adapter interface for OpenCode.
type OpenCode struct{}

// New creates a new OpenCode adapter.
func New() *OpenCode {
	return &OpenCode{}
}

// Name returns the human-readable name of this adapter.
func (o *OpenCode) Name() string {
	return "OpenCode"
}

// GetConfigPath returns the path to the OpenCode configuration file for the given scope.
func (o *OpenCode) GetConfigPath(scope adapter.Scope) string {
	switch scope {
	case adapter.ScopeUser:
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, ".config", "opencode", "opencode.json")
	case adapter.ScopeLocal, adapter.ScopeProject:
		return "opencode.json"
	default:
		return "opencode.json"
	}
}

// ServerExists checks if a server with the given name exists in the OpenCode configuration.
func (o *OpenCode) ServerExists(name string, scope adapter.Scope) bool {
	configPath := o.GetConfigPath(scope)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}

	var existingConfig OpenCodeConfig
	if err := json.Unmarshal(data, &existingConfig); err != nil {
		return false
	}

	_, exists := existingConfig.MCP[name]
	return exists
}

// AddServer adds an MCP server to OpenCode's configuration by directly manipulating the config file.
func (o *OpenCode) AddServer(name string, server *config.Server, scope adapter.Scope, env map[string]string) error {
	configPath := o.GetConfigPath(scope)

	// Read existing config or create new
	var existingConfig OpenCodeConfig
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &existingConfig)
	}

	if existingConfig.MCP == nil {
		existingConfig.MCP = make(map[string]OpenCodeMCPServer)
	}

	// Build server config
	mcpServer := o.buildMCPServer(server, env)
	existingConfig.MCP[name] = mcpServer

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Write config
	data, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// DryRun returns a string representation of the configuration that would be added.
func (o *OpenCode) DryRun(name string, server *config.Server, scope adapter.Scope, env map[string]string) string {
	mcpServer := o.buildMCPServer(server, env)
	configPath := o.GetConfigPath(scope)

	serverConfig := map[string]OpenCodeMCPServer{name: mcpServer}
	data, _ := json.MarshalIndent(serverConfig, "", "  ")

	return fmt.Sprintf("Would add to %s:\n%s", configPath, string(data))
}

// buildMCPServer creates an OpenCodeMCPServer from a config.Server
func (o *OpenCode) buildMCPServer(server *config.Server, env map[string]string) OpenCodeMCPServer {
	mcpServer := OpenCodeMCPServer{
		Enabled: true,
	}

	if server.Transport == "http" || server.Transport == "sse" {
		mcpServer.Type = "remote"
		mcpServer.URL = server.URL
		if server.Headers != nil {
			mcpServer.Headers = make(map[string]string)
			for k, v := range server.Headers {
				mcpServer.Headers[k] = v
			}
		}
	} else {
		mcpServer.Type = "local"
		mcpServer.Command = append([]string{server.Command}, server.Args...)
	}

	// Set environment variables
	if len(env) > 0 {
		mcpServer.Environment = make(map[string]string)
		for k, v := range env {
			mcpServer.Environment[k] = v
		}
	}

	return mcpServer
}
