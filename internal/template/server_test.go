package template

import (
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

func TestProcessServer(t *testing.T) {
	tests := []struct {
		name    string
		server  *config.Server
		data    *TemplateData
		check   func(*testing.T, *config.Server)
		wantErr bool
	}{
		{
			name:   "nil server",
			server: nil,
			data:   NewTemplateData(),
			check: func(t *testing.T, s *config.Server) {
				if s != nil {
					t.Error("expected nil server")
				}
			},
		},
		{
			name: "no templates - passthrough",
			server: &config.Server{
				Transport: "stdio",
				Command:   "/usr/bin/test",
				Args:      []string{"--flag", "value"},
			},
			data: NewTemplateData(),
			check: func(t *testing.T, s *config.Server) {
				if s.Command != "/usr/bin/test" {
					t.Errorf("Command = %q, want %q", s.Command, "/usr/bin/test")
				}
			},
		},
		{
			name: "template in command",
			server: &config.Server{
				Transport: "stdio",
				Command:   "{{ .Env.HOME }}/bin/server",
			},
			data: &TemplateData{Env: map[string]string{"HOME": "/home/user"}},
			check: func(t *testing.T, s *config.Server) {
				if s.Command != "/home/user/bin/server" {
					t.Errorf("Command = %q, want %q", s.Command, "/home/user/bin/server")
				}
			},
		},
		{
			name: "template in args",
			server: &config.Server{
				Transport: "stdio",
				Command:   "server",
				Args:      []string{"--config", "{{ .Env.CONFIG_PATH }}"},
			},
			data: &TemplateData{Env: map[string]string{"CONFIG_PATH": "/etc/config.yaml"}},
			check: func(t *testing.T, s *config.Server) {
				if len(s.Args) != 2 || s.Args[1] != "/etc/config.yaml" {
					t.Errorf("Args = %v, want [--config /etc/config.yaml]", s.Args)
				}
			},
		},
		{
			name: "template in URL",
			server: &config.Server{
				Transport: "http",
				URL:       "http://{{ .Env.API_HOST }}:8080",
			},
			data: &TemplateData{Env: map[string]string{"API_HOST": "localhost"}},
			check: func(t *testing.T, s *config.Server) {
				if s.URL != "http://localhost:8080" {
					t.Errorf("URL = %q, want %q", s.URL, "http://localhost:8080")
				}
			},
		},
		{
			name: "template in env default - CRITICAL TEST",
			server: &config.Server{
				Transport: "stdio",
				Command:   "server",
				Env: map[string]config.EnvVar{
					"API_KEY": {
						Required:    true,
						Description: "API key",
						Default:     "{{ .Env.DEFAULT_API_KEY }}",
					},
				},
			},
			data: &TemplateData{Env: map[string]string{"DEFAULT_API_KEY": "secret123"}},
			check: func(t *testing.T, s *config.Server) {
				if s.Env["API_KEY"].Default != "secret123" {
					t.Errorf("Env[API_KEY].Default = %q, want %q", s.Env["API_KEY"].Default, "secret123")
				}
			},
		},
		{
			name: "description fields NOT templated",
			server: &config.Server{
				Transport:   "stdio",
				Command:     "server",
				Description: "{{ .Env.SHOULD_NOT_EXPAND }}",
				Env: map[string]config.EnvVar{
					"VAR": {
						Description: "{{ .Env.SHOULD_NOT_EXPAND }}",
						Default:     "literal",
					},
				},
			},
			data: &TemplateData{Env: map[string]string{"SHOULD_NOT_EXPAND": "expanded"}},
			check: func(t *testing.T, s *config.Server) {
				// Description fields should be unchanged (not templated)
				if s.Description != "{{ .Env.SHOULD_NOT_EXPAND }}" {
					t.Errorf("Description was templated, got %q", s.Description)
				}
				if s.Env["VAR"].Description != "{{ .Env.SHOULD_NOT_EXPAND }}" {
					t.Errorf("EnvVar.Description was templated, got %q", s.Env["VAR"].Description)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessServer(tt.server, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
