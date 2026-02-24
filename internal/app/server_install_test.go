package app

import (
	"context"
	"errors"
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/adapter/mock"
)

func TestServerInstaller_Install(t *testing.T) {
	tests := []struct {
		name           string
		serverName     string
		scope          adapter.Scope
		envValues      map[string]string
		mockErr        error
		wantSuccess    bool
		wantErrContain string
	}{
		{
			name:        "successful installation",
			serverName:  "server-alpha",
			scope:       adapter.ScopeUser,
			envValues:   nil,
			mockErr:     nil,
			wantSuccess: true,
		},
		{
			name:        "successful installation with env vars",
			serverName:  "server-beta",
			scope:       adapter.ScopeLocal,
			envValues:   map[string]string{"API_KEY": "test-key"},
			mockErr:     nil,
			wantSuccess: true,
		},
		{
			name:           "server not found",
			serverName:     "nonexistent-server",
			scope:          adapter.ScopeUser,
			envValues:      nil,
			mockErr:        nil,
			wantSuccess:    false,
			wantErrContain: "server not found",
		},
		{
			name:           "adapter error",
			serverName:     "server-alpha",
			scope:          adapter.ScopeUser,
			envValues:      nil,
			mockErr:        errors.New("adapter failure"),
			wantSuccess:    false,
			wantErrContain: "installation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdapter := mock.New()
			mockAdapter.AddServerErr = tt.mockErr
			cfg := mock.DefaultTestConfig()

			installer := NewServerInstaller(cfg, mockAdapter)

			req := ServerInstallRequest{
				ServerName: tt.serverName,
				Scope:      tt.scope,
				EnvValues:  tt.envValues,
			}

			resp := installer.Install(context.Background(), req)

			if resp.Success != tt.wantSuccess {
				t.Errorf("Install() success = %v, want %v", resp.Success, tt.wantSuccess)
			}

			if tt.wantSuccess {
				if resp.Error != nil {
					t.Errorf("Install() error = %v, want nil", resp.Error)
				}
				// Verify adapter was called
				if len(mockAdapter.AddServerCalls) != 1 {
					t.Errorf("expected 1 AddServer call, got %d", len(mockAdapter.AddServerCalls))
				}
			} else {
				if resp.Error == nil {
					t.Error("Install() error = nil, want error")
				} else if tt.wantErrContain != "" && !contains(resp.Error.Error(), tt.wantErrContain) {
					t.Errorf("Install() error = %v, want error containing %q", resp.Error, tt.wantErrContain)
				}
			}
		})
	}
}

func TestServerInstaller_DryRun(t *testing.T) {
	mockAdapter := mock.New()
	cfg := mock.DefaultTestConfig()
	installer := NewServerInstaller(cfg, mockAdapter)

	t.Run("dry run returns command", func(t *testing.T) {
		req := ServerInstallRequest{
			ServerName: "server-alpha",
			Scope:      adapter.ScopeUser,
			DryRun:     true,
		}

		resp := installer.Install(context.Background(), req)

		if !resp.Success {
			t.Errorf("DryRun() success = false, want true")
		}
		// Verify adapter was NOT called for AddServer
		if len(mockAdapter.AddServerCalls) != 0 {
			t.Errorf("expected 0 AddServer calls in dry run, got %d", len(mockAdapter.AddServerCalls))
		}
	})

	t.Run("dry run nonexistent server", func(t *testing.T) {
		output := installer.DryRun(ServerInstallRequest{
			ServerName: "nonexistent",
			Scope:      adapter.ScopeUser,
		})

		if !contains(output, "Error") {
			t.Errorf("DryRun() output = %q, want error message", output)
		}
	})
}

func TestServerInstaller_BulkInstall(t *testing.T) {
	t.Run("all successful", func(t *testing.T) {
		mockAdapter := mock.New()
		cfg := mock.DefaultTestConfig()
		installer := NewServerInstaller(cfg, mockAdapter)

		reqs := []ServerInstallRequest{
			{ServerName: "server-alpha", Scope: adapter.ScopeUser},
			{ServerName: "server-gamma", Scope: adapter.ScopeUser},
		}

		results := installer.BulkInstall(context.Background(), reqs)

		if len(results) != 2 {
			t.Fatalf("BulkInstall() returned %d results, want 2", len(results))
		}

		for i, resp := range results {
			if !resp.Success {
				t.Errorf("result[%d] success = false, want true (error: %v)", i, resp.Error)
			}
		}

		if len(mockAdapter.AddServerCalls) != 2 {
			t.Errorf("expected 2 AddServer calls, got %d", len(mockAdapter.AddServerCalls))
		}
	})

	t.Run("partial failure", func(t *testing.T) {
		mockAdapter := mock.New()
		cfg := mock.DefaultTestConfig()
		installer := NewServerInstaller(cfg, mockAdapter)

		reqs := []ServerInstallRequest{
			{ServerName: "server-alpha", Scope: adapter.ScopeUser},
			{ServerName: "nonexistent", Scope: adapter.ScopeUser},
			{ServerName: "server-gamma", Scope: adapter.ScopeUser},
		}

		results := installer.BulkInstall(context.Background(), reqs)

		if len(results) != 3 {
			t.Fatalf("BulkInstall() returned %d results, want 3", len(results))
		}

		if !results[0].Success {
			t.Error("result[0] should be successful")
		}
		if results[1].Success {
			t.Error("result[1] should fail (nonexistent)")
		}
		if !results[2].Success {
			t.Error("result[2] should be successful")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		mockAdapter := mock.New()
		cfg := mock.DefaultTestConfig()
		installer := NewServerInstaller(cfg, mockAdapter)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		reqs := []ServerInstallRequest{
			{ServerName: "server-alpha", Scope: adapter.ScopeUser},
			{ServerName: "server-gamma", Scope: adapter.ScopeUser},
		}

		results := installer.BulkInstall(ctx, reqs)

		// All should be marked as cancelled
		for i, resp := range results {
			if resp.Success {
				t.Errorf("result[%d] should not succeed when context is cancelled", i)
			}
		}
	})
}

func TestServerInstaller_InstallerWithAccess(t *testing.T) {
	mockAdapter := mock.New()
	cfg := mock.DefaultTestConfig()
	installer := NewServerInstaller(cfg, mockAdapter)

	// Type assertion to InstallerWithAccess
	iwa, ok := installer.(InstallerWithAccess)
	if !ok {
		t.Fatal("NewServerInstaller should return InstallerWithAccess")
	}

	if iwa.GetConfig() != cfg {
		t.Error("GetConfig() should return original config")
	}

	if iwa.GetAdapter() != mockAdapter {
		t.Error("GetAdapter() should return original adapter")
	}
}

func TestErrorSentinels(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrServerNotFound", ErrServerNotFound},
		{"ErrAlreadyInstalled", ErrAlreadyInstalled},
		{"ErrInvalidScope", ErrInvalidScope},
		{"ErrMissingRequiredEnv", ErrMissingRequiredEnv},
		{"ErrInstallFailed", ErrInstallFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s should have an error message", tt.name)
			}
		})
	}
}

func TestServerInstallResponse_ServerName(t *testing.T) {
	mockAdapter := mock.New()
	cfg := mock.DefaultTestConfig()
	installer := NewServerInstaller(cfg, mockAdapter)

	req := ServerInstallRequest{
		ServerName: "server-alpha",
		Scope:      adapter.ScopeUser,
	}

	resp := installer.Install(context.Background(), req)

	if resp.ServerName != "server-alpha" {
		t.Errorf("Response.ServerName = %q, want %q", resp.ServerName, "server-alpha")
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
