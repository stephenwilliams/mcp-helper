// Package claudecode provides an adapter for configuring MCP servers in Claude Code.
// It uses the `claude mcp add` command to add servers to Claude Code's configuration.
package claudecode

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// ClaudeCode implements the Adapter interface for Claude Code.
// It uses the Claude CLI to manage MCP server configurations.
type ClaudeCode struct {
	claudePath string
}

// New creates a new ClaudeCode adapter with the default claude command path.
// The default path is "claude", which assumes the claude binary is in the system PATH.
//
// Returns:
//   - A new ClaudeCode adapter instance
func New() *ClaudeCode {
	return &ClaudeCode{
		claudePath: "claude",
	}
}

// NewWithPath creates a new ClaudeCode adapter with a custom claude command path.
// This is useful when the claude binary is not in the system PATH or when using
// a specific version of the claude CLI.
//
// Parameters:
//   - path: The path to the claude binary
//
// Returns:
//   - A new ClaudeCode adapter instance
func NewWithPath(path string) *ClaudeCode {
	return &ClaudeCode{
		claudePath: path,
	}
}

// Name returns the human-readable name of this adapter.
//
// Returns:
//   - The adapter name "Claude Code"
func (c *ClaudeCode) Name() string {
	return "Claude Code"
}

// AddServer adds an MCP server to Claude Code's configuration using the claude CLI.
// It supports both stdio and HTTP transports and handles environment variable configuration.
//
// For stdio transport, the command format is:
//   claude mcp add --scope <scope> [-e KEY=val]... <name> -- <command> [args...]
//
// For HTTP transport, the command format is:
//   claude mcp add --transport http --scope <scope> <name> <url>
//
// Parameters:
//   - name: The unique identifier for the server
//   - server: The server configuration including command, arguments, and environment
//   - scope: The configuration scope (local, user, or project)
//   - env: Additional environment variables to merge with the server's environment
//
// Returns:
//   - An error if the server could not be added or if the claude CLI is not found
func (c *ClaudeCode) AddServer(name string, server *config.Server, scope adapter.Scope, env map[string]string) error {
	args := c.buildArgs(name, server, scope, env)

	cmd := exec.Command(c.claudePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is due to claude not being found
		if strings.Contains(err.Error(), "executable file not found") {
			return fmt.Errorf("claude CLI not found at '%s': please install Claude Code CLI or specify the correct path", c.claudePath)
		}
		return fmt.Errorf("failed to add server: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// DryRun returns the command string that would be executed to add the server,
// without actually executing it. This is useful for previewing changes or
// generating documentation.
//
// Parameters:
//   - name: The unique identifier for the server
//   - server: The server configuration including command, arguments, and environment
//   - scope: The configuration scope (local, user, or project)
//   - env: Additional environment variables to merge with the server's environment
//
// Returns:
//   - A string representation of the command that would be executed
func (c *ClaudeCode) DryRun(name string, server *config.Server, scope adapter.Scope, env map[string]string) string {
	args := c.buildArgs(name, server, scope, env)

	// Build the command string with proper quoting
	parts := []string{c.claudePath}
	for _, arg := range args {
		// Quote arguments that contain spaces
		if strings.Contains(arg, " ") {
			parts = append(parts, fmt.Sprintf(`"%s"`, arg))
		} else {
			parts = append(parts, arg)
		}
	}

	return strings.Join(parts, " ")
}

// buildArgs constructs the command-line arguments for the claude mcp add command.
// It handles both stdio and HTTP transports and merges environment variables.
//
// Parameters:
//   - name: The unique identifier for the server
//   - server: The server configuration including command, arguments, and environment
//   - scope: The configuration scope (local, user, or project)
//   - env: Additional environment variables to merge with the server's environment
//
// Returns:
//   - A slice of command-line arguments for the claude CLI
func (c *ClaudeCode) buildArgs(name string, server *config.Server, scope adapter.Scope, env map[string]string) []string {
	args := []string{"mcp", "add"}

	// Add transport flag for HTTP
	if server.Transport == "http" {
		args = append(args, "--transport", "http")
	}

	// Add scope
	args = append(args, "--scope", string(scope))

	// Merge environment variables
	mergedEnv := make(map[string]string)
	// First, add server's environment variables
	for key, envVar := range server.Env {
		if val, exists := env[key]; exists {
			mergedEnv[key] = val
		} else if envVar.Default != "" {
			mergedEnv[key] = envVar.Default
		}
	}
	// Then, add any additional environment variables
	for key, val := range env {
		if val != "" {
			mergedEnv[key] = val
		}
	}

	// Add environment variables
	for key, val := range mergedEnv {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}

	// Add name
	args = append(args, name)

	// Add command and args for stdio, or URL for http
	if server.Transport == "http" {
		args = append(args, server.URL)
	} else {
		// For stdio, add separator then command and args
		args = append(args, "--")
		args = append(args, server.Command)
		args = append(args, server.Args...)
	}

	return args
}
