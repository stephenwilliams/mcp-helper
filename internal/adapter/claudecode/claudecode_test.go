package claudecode

import (
	"strings"
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

func TestNew(t *testing.T) {
	cc := New()
	if cc == nil {
		t.Fatal("New() returned nil")
	}
	if cc.claudePath != "claude" {
		t.Errorf("New() claudePath = %q, want %q", cc.claudePath, "claude")
	}
}

func TestNewWithPath(t *testing.T) {
	customPath := "/custom/path/to/claude"
	cc := NewWithPath(customPath)
	if cc == nil {
		t.Fatal("NewWithPath() returned nil")
	}
	if cc.claudePath != customPath {
		t.Errorf("NewWithPath() claudePath = %q, want %q", cc.claudePath, customPath)
	}
}

func TestName(t *testing.T) {
	cc := New()
	name := cc.Name()
	expected := "Claude Code"
	if name != expected {
		t.Errorf("Name() = %q, want %q", name, expected)
	}
}

func TestDryRun_StdioTransport(t *testing.T) {
	tests := []struct {
		name         string
		serverName   string
		server       *config.Server
		scope        adapter.Scope
		env          map[string]string
		wantContains []string
	}{
		{
			name:       "basic stdio server",
			serverName: "test-server",
			server: &config.Server{
				Transport: "stdio",
				Command:   "node",
				Args:      []string{"server.js"},
			},
			scope: adapter.ScopeUser,
			env:   nil,
			wantContains: []string{
				"claude",
				"mcp",
				"add",
				"--scope",
				"user",
				"test-server",
				"--",
				"node",
				"server.js",
			},
		},
		{
			name:       "stdio with environment variables",
			serverName: "env-server",
			server: &config.Server{
				Transport: "stdio",
				Command:   "/bin/test",
				Args:      []string{"--arg1", "value1"},
				Env: map[string]config.EnvVar{
					"API_TOKEN": {
						Required:    true,
						Description: "API token",
					},
					"DEBUG": {
						Required:    false,
						Description: "Debug mode",
						Default:     "false",
					},
				},
			},
			scope: adapter.ScopeLocal,
			env: map[string]string{
				"API_TOKEN": "secret123",
			},
			wantContains: []string{
				"mcp",
				"add",
				"--scope",
				"local",
				"-e",
				"API_TOKEN=secret123",
				"env-server",
				"--",
				"/bin/test",
				"--arg1",
				"value1",
			},
		},
		{
			name:       "stdio with default env value",
			serverName: "default-env",
			server: &config.Server{
				Transport: "stdio",
				Command:   "python",
				Args:      []string{"-m", "server"},
				Env: map[string]config.EnvVar{
					"PORT": {
						Required:    false,
						Description: "Server port",
						Default:     "8080",
					},
				},
			},
			scope: adapter.ScopeProject,
			env:   map[string]string{},
			wantContains: []string{
				"--scope",
				"project",
				"-e",
				"PORT=8080",
				"default-env",
			},
		},
		{
			name:       "stdio with arguments containing spaces",
			serverName: "spaces-server",
			server: &config.Server{
				Transport: "stdio",
				Command:   "/path/to/server",
				Args:      []string{"--config", "path with spaces/config.json"},
			},
			scope: adapter.ScopeUser,
			env:   nil,
			wantContains: []string{
				"mcp",
				"add",
				"spaces-server",
				"--",
				"/path/to/server",
				"--config",
				`"path with spaces/config.json"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := New()
			got := cc.DryRun(tt.serverName, tt.server, tt.scope, tt.env)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("DryRun() output missing %q\nGot: %s", want, got)
				}
			}
		})
	}
}

func TestDryRun_HttpTransport(t *testing.T) {
	tests := []struct {
		name         string
		serverName   string
		server       *config.Server
		scope        adapter.Scope
		env          map[string]string
		wantContains []string
		wantNotContain []string
	}{
		{
			name:       "basic http server",
			serverName: "http-server",
			server: &config.Server{
				Transport: "http",
				URL:       "http://localhost:8080",
			},
			scope: adapter.ScopeUser,
			env:   nil,
			wantContains: []string{
				"claude",
				"mcp",
				"add",
				"--transport",
				"http",
				"--scope",
				"user",
				"http-server",
				"http://localhost:8080",
			},
			wantNotContain: []string{" -- "},
		},
		{
			name:       "http with environment",
			serverName: "secure-http",
			server: &config.Server{
				Transport: "http",
				URL:       "https://api.example.com",
				Env: map[string]config.EnvVar{
					"API_KEY": {
						Required:    true,
						Description: "API key",
					},
				},
			},
			scope: adapter.ScopeLocal,
			env: map[string]string{
				"API_KEY": "key123",
			},
			wantContains: []string{
				"--transport",
				"http",
				"--scope",
				"local",
				"-e",
				"API_KEY=key123",
				"secure-http",
				"https://api.example.com",
			},
			wantNotContain: []string{" -- "},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := New()
			got := cc.DryRun(tt.serverName, tt.server, tt.scope, tt.env)

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("DryRun() output missing %q\nGot: %s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("DryRun() output should not contain %q\nGot: %s", notWant, got)
				}
			}
		})
	}
}

func TestDryRun_EnvMerging(t *testing.T) {
	server := &config.Server{
		Transport: "stdio",
		Command:   "test",
		Env: map[string]config.EnvVar{
			"FROM_CONFIG": {
				Required:    false,
				Description: "From config",
				Default:     "config_default",
			},
			"OVERRIDE_ME": {
				Required:    false,
				Description: "Will be overridden",
				Default:     "config_default",
			},
		},
	}

	env := map[string]string{
		"FROM_ENV":    "env_value",
		"OVERRIDE_ME": "env_override",
	}

	cc := New()
	got := cc.DryRun("test", server, adapter.ScopeUser, env)

	// Should contain values from env map
	if !strings.Contains(got, "FROM_ENV=env_value") {
		t.Errorf("DryRun() missing FROM_ENV=env_value\nGot: %s", got)
	}

	// Should contain overridden value
	if !strings.Contains(got, "OVERRIDE_ME=env_override") {
		t.Errorf("DryRun() missing OVERRIDE_ME=env_override\nGot: %s", got)
	}

	// Should contain default from config
	if !strings.Contains(got, "FROM_CONFIG=config_default") {
		t.Errorf("DryRun() missing FROM_CONFIG=config_default\nGot: %s", got)
	}
}

func TestBuildArgs_AllScopes(t *testing.T) {
	scopes := []struct {
		scope adapter.Scope
		want  string
	}{
		{adapter.ScopeLocal, "local"},
		{adapter.ScopeUser, "user"},
		{adapter.ScopeProject, "project"},
	}

	server := &config.Server{
		Transport: "stdio",
		Command:   "test",
	}

	cc := New()

	for _, tt := range scopes {
		t.Run(string(tt.scope), func(t *testing.T) {
			args := cc.buildArgs("test", server, tt.scope, nil)

			found := false
			for i, arg := range args {
				if arg == "--scope" && i+1 < len(args) {
					if args[i+1] == tt.want {
						found = true
						break
					}
				}
			}

			if !found {
				t.Errorf("buildArgs() scope not found or incorrect. Want --scope %s in %v", tt.want, args)
			}
		})
	}
}

func TestAddServer_Documentation(t *testing.T) {
	t.Skip("AddServer() requires claude CLI to be installed - skipping in CI")

	// This test documents the expected behavior of AddServer
	// It would fail in CI/CD environments where claude CLI is not installed
	//
	// Expected behavior:
	// 1. Executes: claude mcp add [args]
	// 2. Returns nil on success
	// 3. Returns error with helpful message if claude CLI not found
	// 4. Returns error with command output if command fails
}

func TestAddServer_NonExistentClaude(t *testing.T) {
	// Test error handling when claude command doesn't exist
	cc := NewWithPath("/nonexistent/claude")

	server := &config.Server{
		Transport: "stdio",
		Command:   "test",
	}

	err := cc.AddServer("test", server, adapter.ScopeUser, nil)
	if err == nil {
		t.Error("AddServer() with non-existent claude path should return error")
	}

	// Error should mention the path or that claude is not found
	if !strings.Contains(err.Error(), "nonexistent") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("AddServer() error should mention missing executable, got: %v", err)
	}
}
