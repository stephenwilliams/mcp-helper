package app

import (
	"context"
	"fmt"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// ServerInstallRequest contains parameters for installing a server.
type ServerInstallRequest struct {
	ServerName string
	Scope      adapter.Scope
	EnvValues  map[string]string
	DryRun     bool
}

// ServerInstallResponse contains the result of an installation attempt.
type ServerInstallResponse struct {
	ServerName string
	Success    bool
	DryRunCmd  string // populated if DryRun=true
	Error      error
}

// ServerInstaller defines the interface for server installation operations.
// This interface is defined in the application layer and implemented here,
// allowing the TUI and CLI to depend on the abstraction rather than concrete types.
type ServerInstaller interface {
	// Install installs a single server with the provided configuration.
	Install(ctx context.Context, req ServerInstallRequest) ServerInstallResponse

	// BulkInstall installs multiple servers, returning results for each.
	BulkInstall(ctx context.Context, reqs []ServerInstallRequest) []ServerInstallResponse

	// DryRun returns the command that would be executed without actually installing.
	DryRun(req ServerInstallRequest) string
}

// ServerChecker defines the interface for checking server state.
// This is a minimal interface for checking if a server exists.
type ServerChecker interface {
	// ServerExists checks if a server is already installed in the given scope.
	ServerExists(name string, scope adapter.Scope) bool
}

// serverInstaller implements the ServerInstaller interface.
type serverInstaller struct {
	config  *config.Config
	adapter adapter.Adapter
}

// NewServerInstaller creates a new ServerInstaller with the given configuration and adapter.
func NewServerInstaller(cfg *config.Config, adptr adapter.Adapter) ServerInstaller {
	return &serverInstaller{
		config:  cfg,
		adapter: adptr,
	}
}

// Install implements ServerInstaller.Install.
func (s *serverInstaller) Install(ctx context.Context, req ServerInstallRequest) ServerInstallResponse {
	server, exists := s.config.Servers[req.ServerName]
	if !exists {
		return ServerInstallResponse{
			ServerName: req.ServerName,
			Success:    false,
			Error:      fmt.Errorf("%w: %s", ErrServerNotFound, req.ServerName),
		}
	}

	if req.DryRun {
		cmd := s.adapter.DryRun(req.ServerName, server, req.Scope, req.EnvValues)
		return ServerInstallResponse{
			ServerName: req.ServerName,
			Success:    true,
			DryRunCmd:  cmd,
		}
	}

	err := s.adapter.AddServer(req.ServerName, server, req.Scope, req.EnvValues)
	if err != nil {
		return ServerInstallResponse{
			ServerName: req.ServerName,
			Success:    false,
			Error:      fmt.Errorf("%w: %v", ErrInstallFailed, err),
		}
	}

	return ServerInstallResponse{
		ServerName: req.ServerName,
		Success:    true,
	}
}

// BulkInstall implements ServerInstaller.BulkInstall.
func (s *serverInstaller) BulkInstall(ctx context.Context, reqs []ServerInstallRequest) []ServerInstallResponse {
	results := make([]ServerInstallResponse, 0, len(reqs))

	for _, req := range reqs {
		// Check for context cancellation between installs
		select {
		case <-ctx.Done():
			// Mark remaining as cancelled
			for i := len(results); i < len(reqs); i++ {
				results = append(results, ServerInstallResponse{
					ServerName: reqs[i].ServerName,
					Success:    false,
					Error:      ctx.Err(),
				})
			}
			return results
		default:
			results = append(results, s.Install(ctx, req))
		}
	}

	return results
}

// DryRun implements ServerInstaller.DryRun.
func (s *serverInstaller) DryRun(req ServerInstallRequest) string {
	server, exists := s.config.Servers[req.ServerName]
	if !exists {
		return fmt.Sprintf("# Error: server not found: %s", req.ServerName)
	}

	return s.adapter.DryRun(req.ServerName, server, req.Scope, req.EnvValues)
}

// GetConfig returns the underlying configuration.
// This is useful for accessing server metadata without going through the adapter.
func (s *serverInstaller) GetConfig() *config.Config {
	return s.config
}

// GetAdapter returns the underlying adapter.
// This provides access to adapter-specific functionality like ServerExists.
func (s *serverInstaller) GetAdapter() adapter.Adapter {
	return s.adapter
}

// InstallerWithAccess provides access to the underlying config and adapter.
// This interface is used when callers need more than just installation capabilities.
type InstallerWithAccess interface {
	ServerInstaller
	GetConfig() *config.Config
	GetAdapter() adapter.Adapter
}

// Ensure serverInstaller implements InstallerWithAccess
var _ InstallerWithAccess = (*serverInstaller)(nil)
