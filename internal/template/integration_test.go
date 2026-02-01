package template_test

import (
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/config"
	"github.com/stephenwilliams/mcp-helper/internal/env"
	"github.com/stephenwilliams/mcp-helper/internal/template"
)

// TestEnvPriorityWithTemplatedDefaults verifies that templated EnvVar.Default
// values are correctly resolved BEFORE env.CollectEnvVars uses them.
// This is the critical integration test that validates the architecture.
func TestEnvPriorityWithTemplatedDefaults(t *testing.T) {
	// Server with a templated default value
	server := &config.Server{
		Transport: "stdio",
		Command:   "test-server",
		Env: map[string]config.EnvVar{
			"API_KEY": {
				Required:    false,
				Description: "API key for service",
				Default:     "{{ .Env.FALLBACK_KEY }}",
			},
		},
	}

	// Template data with the fallback key
	tmplData := &template.TemplateData{
		Env: map[string]string{
			"FALLBACK_KEY": "resolved-secret-value",
		},
	}

	// Step 1: Process templates (as cmd/add.go does)
	processedServer, err := template.ProcessServer(server, tmplData)
	if err != nil {
		t.Fatalf("ProcessServer failed: %v", err)
	}

	// Verify the default was resolved
	if processedServer.Env["API_KEY"].Default != "resolved-secret-value" {
		t.Fatalf("Default not resolved: got %q, want %q",
			processedServer.Env["API_KEY"].Default, "resolved-secret-value")
	}

	// Step 2: Collect env vars (no provided values, non-interactive)
	// This should use the resolved default value
	collectedEnv, err := env.CollectEnvVars(processedServer, nil, false)
	if err != nil {
		t.Fatalf("CollectEnvVars failed: %v", err)
	}

	// Verify the collected value is the resolved template value
	if collectedEnv["API_KEY"] != "resolved-secret-value" {
		t.Errorf("CollectEnvVars used wrong default: got %q, want %q",
			collectedEnv["API_KEY"], "resolved-secret-value")
	}
}

// TestEnvPriorityProvidedOverridesTemplatedDefault verifies that
// explicitly provided values still override templated defaults.
func TestEnvPriorityProvidedOverridesTemplatedDefault(t *testing.T) {
	server := &config.Server{
		Transport: "stdio",
		Command:   "test-server",
		Env: map[string]config.EnvVar{
			"API_KEY": {
				Required: false,
				Default:  "{{ .Env.FALLBACK_KEY }}",
			},
		},
	}

	tmplData := &template.TemplateData{
		Env: map[string]string{"FALLBACK_KEY": "from-template"},
	}

	processedServer, err := template.ProcessServer(server, tmplData)
	if err != nil {
		t.Fatalf("ProcessServer failed: %v", err)
	}

	// Provide an explicit value
	providedEnv := map[string]string{"API_KEY": "explicitly-provided"}

	collectedEnv, err := env.CollectEnvVars(processedServer, providedEnv, false)
	if err != nil {
		t.Fatalf("CollectEnvVars failed: %v", err)
	}

	// Provided value should win over templated default
	if collectedEnv["API_KEY"] != "explicitly-provided" {
		t.Errorf("Provided value did not override: got %q, want %q",
			collectedEnv["API_KEY"], "explicitly-provided")
	}
}
