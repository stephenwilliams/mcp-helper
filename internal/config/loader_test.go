package config

import (
	"os"
	"testing"
)

func TestLoad_NoConfigFile(t *testing.T) {
	// Save and restore original env and working directory
	oldEnv := os.Getenv("MCP_HELPER_CONFIG")
	oldDir, _ := os.Getwd()
	defer func() {
		os.Setenv("MCP_HELPER_CONFIG", oldEnv)
		os.Chdir(oldDir)
	}()

	// Create a temp directory with no config file and change to it
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	os.Unsetenv("MCP_HELPER_CONFIG")

	cfg, err := Load()
	if err != nil {
		t.Errorf("Load() returned error for missing config: %v", err)
	}
	if cfg != nil {
		t.Errorf("Load() expected nil config for missing file, got %v", cfg)
	}
}

func TestLoadFromPath_ValidConfig(t *testing.T) {
	testConfig := "../../testdata/config_test.yaml"

	cfg, err := LoadFromPath(testConfig)
	if err != nil {
		t.Fatalf("LoadFromPath() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadFromPath() returned nil config")
	}

	if cfg.DefaultScope != "user" {
		t.Errorf("DefaultScope = %q, want %q", cfg.DefaultScope, "user")
	}

	if len(cfg.Servers) != 3 {
		t.Errorf("len(Servers) = %d, want 3", len(cfg.Servers))
	}

	// Test stdio server
	stdioServer, err := cfg.GetServer("test-stdio")
	if err != nil {
		t.Errorf("GetServer(test-stdio) error = %v", err)
	}
	if stdioServer.Transport != "stdio" {
		t.Errorf("test-stdio transport = %q, want %q", stdioServer.Transport, "stdio")
	}
	if stdioServer.Command != "/usr/bin/test-server" {
		t.Errorf("test-stdio command = %q, want %q", stdioServer.Command, "/usr/bin/test-server")
	}
	if len(stdioServer.Args) != 2 {
		t.Errorf("test-stdio args count = %d, want 2", len(stdioServer.Args))
	}

	// Test http server
	httpServer, err := cfg.GetServer("test-http")
	if err != nil {
		t.Errorf("GetServer(test-http) error = %v", err)
	}
	if httpServer.Transport != "http" {
		t.Errorf("test-http transport = %q, want %q", httpServer.Transport, "http")
	}
	if httpServer.URL != "http://localhost:8080" {
		t.Errorf("test-http URL = %q, want %q", httpServer.URL, "http://localhost:8080")
	}
}

func TestLoadFromPath_InvalidPath(t *testing.T) {
	_, err := LoadFromPath("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("LoadFromPath() expected error for invalid path, got nil")
	}
}

func TestLoadFromPath_InvalidConfig(t *testing.T) {
	testConfig := "../../testdata/invalid_config.yaml"

	_, err := LoadFromPath(testConfig)
	if err == nil {
		t.Error("LoadFromPath() expected error for invalid config, got nil")
	}
}

func TestGetServer(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		serverName string
		wantErr    bool
		wantNil    bool
	}{
		{
			name: "server found",
			config: &Config{
				Servers: map[string]*Server{
					"test": {
						Transport: "stdio",
						Command:   "test",
					},
				},
			},
			serverName: "test",
			wantErr:    false,
			wantNil:    false,
		},
		{
			name: "server not found",
			config: &Config{
				Servers: map[string]*Server{
					"test": {
						Transport: "stdio",
						Command:   "test",
					},
				},
			},
			serverName: "nonexistent",
			wantErr:    true,
			wantNil:    true,
		},
		{
			name: "nil servers map",
			config: &Config{
				Servers: nil,
			},
			serverName: "test",
			wantErr:    true,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := tt.config.GetServer(tt.serverName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (server == nil) != tt.wantNil {
				t.Errorf("GetServer() server = %v, wantNil %v", server, tt.wantNil)
			}
		})
	}
}

func TestListServers(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   []string
	}{
		{
			name: "multiple servers sorted",
			config: &Config{
				Servers: map[string]*Server{
					"zebra":  {Transport: "stdio", Command: "zebra"},
					"alpha":  {Transport: "stdio", Command: "alpha"},
					"middle": {Transport: "stdio", Command: "middle"},
				},
			},
			want: []string{"alpha", "middle", "zebra"},
		},
		{
			name: "single server",
			config: &Config{
				Servers: map[string]*Server{
					"only": {Transport: "stdio", Command: "only"},
				},
			},
			want: []string{"only"},
		},
		{
			name: "nil servers map",
			config: &Config{
				Servers: nil,
			},
			want: []string{},
		},
		{
			name: "empty servers map",
			config: &Config{
				Servers: map[string]*Server{},
			},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ListServers()
			if len(got) != len(tt.want) {
				t.Errorf("ListServers() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ListServers()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid stdio config",
			config: &Config{
				Servers: map[string]*Server{
					"test": {
						Transport: "stdio",
						Command:   "/bin/test",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid http config",
			config: &Config{
				Servers: map[string]*Server{
					"test": {
						Transport: "http",
						URL:       "http://localhost:8080",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty config",
			config: &Config{
				Servers: nil,
			},
			wantErr: false,
		},
		{
			name: "invalid transport",
			config: &Config{
				Servers: map[string]*Server{
					"test": {
						Transport: "invalid",
						Command:   "/bin/test",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid transport",
		},
		{
			name: "stdio missing command",
			config: &Config{
				Servers: map[string]*Server{
					"test": {
						Transport: "stdio",
						Command:   "",
					},
				},
			},
			wantErr: true,
			errMsg:  "has no command specified",
		},
		{
			name: "http missing URL",
			config: &Config{
				Servers: map[string]*Server{
					"test": {
						Transport: "http",
						URL:       "",
					},
				},
			},
			wantErr: true,
			errMsg:  "has no URL specified",
		},
		{
			name: "nil server",
			config: &Config{
				Servers: map[string]*Server{
					"test": nil,
				},
			},
			wantErr: true,
			errMsg:  "is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
