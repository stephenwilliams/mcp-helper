package cmd

import (
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

func TestResolveScope(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		cfg       *config.Config
		want      string
	}{
		{
			name:      "flag value takes priority",
			flagValue: "project",
			cfg:       &config.Config{DefaultScope: "user"},
			want:      "project",
		},
		{
			name:      "config default used when flag empty",
			flagValue: "",
			cfg:       &config.Config{DefaultScope: "user"},
			want:      "user",
		},
		{
			name:      "fallback to local when both empty",
			flagValue: "",
			cfg:       &config.Config{},
			want:      "local",
		},
		{
			name:      "flag value used even when config has default",
			flagValue: "local",
			cfg:       &config.Config{DefaultScope: "user"},
			want:      "local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveScope(tt.flagValue, tt.cfg)
			if got != tt.want {
				t.Errorf("resolveScope(%q, cfg) = %q, want %q", tt.flagValue, got, tt.want)
			}
		})
	}
}
