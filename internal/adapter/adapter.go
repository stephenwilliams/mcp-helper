// Package adapter provides an abstraction layer for managing MCP (Model Context Protocol)
// servers across different environments and scopes. It defines a common interface for
// adding servers and generating configuration in a dry-run mode.
package adapter

import (
	"fmt"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// Scope represents the configuration scope where an MCP server should be added.
// Different scopes determine the visibility and lifecycle of the server configuration.
type Scope string

const (
	// ScopeLocal indicates the server configuration is specific to the current working directory.
	// Typically stored in a local configuration file like .mcp.json or project-specific config.
	ScopeLocal Scope = "local"

	// ScopeUser indicates the server configuration is specific to the current user.
	// Stored in the user's home directory and applies across all projects for that user.
	ScopeUser Scope = "user"

	// ScopeProject indicates the server configuration is specific to a project.
	// Similar to local scope but may have different semantic meaning depending on the adapter.
	ScopeProject Scope = "project"
)

// ParseScope converts a string representation of a scope into a Scope type.
// It validates that the provided scope string matches one of the defined scope constants.
//
// Parameters:
//   - s: The scope string to parse (e.g., "local", "user", "project")
//
// Returns:
//   - The parsed Scope value
//   - An error if the scope string is invalid
func ParseScope(s string) (Scope, error) {
	scope := Scope(s)
	switch scope {
	case ScopeLocal, ScopeUser, ScopeProject:
		return scope, nil
	default:
		return "", fmt.Errorf("invalid scope: %s (must be one of: local, user, project)", s)
	}
}

// Adapter defines the interface for managing MCP server configurations across different
// environments (e.g., Claude Desktop, VS Code, Zed editor). Each adapter implementation
// handles the specifics of how servers are configured for its target environment.
type Adapter interface {
	// Name returns the human-readable name of the adapter.
	// This is used for display purposes and logging.
	//
	// Returns:
	//   - The adapter name (e.g., "Claude Desktop", "VS Code")
	Name() string

	// AddServer adds an MCP server configuration to the target environment.
	// It handles reading the existing configuration, merging the new server,
	// and persisting the updated configuration.
	//
	// Parameters:
	//   - name: The unique identifier for the server
	//   - server: The server configuration including command, arguments, and environment
	//   - scope: The configuration scope (local, user, or project)
	//   - env: Additional environment variables to merge with the server's environment
	//
	// Returns:
	//   - An error if the server could not be added
	AddServer(name string, server *config.Server, scope Scope, env map[string]string) error

	// DryRun returns a string representation of the command that would be executed
	// to add the server, without actually modifying any configuration files.
	// This is useful for previewing changes or generating documentation.
	//
	// Parameters:
	//   - name: The unique identifier for the server
	//   - server: The server configuration including command, arguments, and environment
	//   - scope: The configuration scope (local, user, or project)
	//   - env: Additional environment variables to merge with the server's environment
	//
	// Returns:
	//   - A string representation of the command or configuration changes
	DryRun(name string, server *config.Server, scope Scope, env map[string]string) string
}
