// Package tui provides terminal user interface components for mcp-helper.
// This file defines the interfaces that the TUI layer depends on,
// following Go's "consumer defines interface" pattern.
package tui

import (
	"context"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/app"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// ServerInstaller defines the installation capabilities needed by the TUI.
// This mirrors app.ServerInstaller but is defined here to follow
// Go's idiom of defining interfaces where they are used.
type ServerInstaller interface {
	// Install installs a single server with the provided configuration.
	Install(ctx context.Context, req app.ServerInstallRequest) app.ServerInstallResponse

	// BulkInstall installs multiple servers, returning results for each.
	BulkInstall(ctx context.Context, reqs []app.ServerInstallRequest) []app.ServerInstallResponse

	// DryRun returns the command that would be executed without actually installing.
	DryRun(req app.ServerInstallRequest) string
}

// ServerChecker provides server existence checking capabilities.
// This is a minimal interface for checking if a server is already installed.
type ServerChecker interface {
	// ServerExists checks if a server is already installed in the given scope.
	ServerExists(name string, scope adapter.Scope) bool
}

// ConfigProvider provides access to server configuration.
// This is used when the TUI needs to display server details.
type ConfigProvider interface {
	// GetConfig returns the server configuration.
	GetConfig() *config.Config
}

// AdapterProvider provides access to the underlying adapter.
// This is used for adapter-specific operations like ServerExists and Name().
type AdapterProvider interface {
	// GetAdapter returns the adapter being used.
	GetAdapter() adapter.Adapter
}

// FullInstaller combines all capabilities needed by the TUI.
// This is the primary interface used by TUI models.
type FullInstaller interface {
	ServerInstaller
	ConfigProvider
	AdapterProvider
}
