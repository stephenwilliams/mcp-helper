package env

import (
	"os"
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// TestCollectEnvVars_ComprehensiveIntegration tests the full integration
// of all CollectEnvVars code paths in non-interactive mode
func TestCollectEnvVars_ComprehensiveIntegration(t *testing.T) {
	// Save and restore environment
	testVars := map[string]string{
		"INTEGRATION_ENV_VAR":     os.Getenv("INTEGRATION_ENV_VAR"),
		"INTEGRATION_OVERRIDE":    os.Getenv("INTEGRATION_OVERRIDE"),
		"INTEGRATION_UNSET":       os.Getenv("INTEGRATION_UNSET"),
		"INTEGRATION_EMPTY_CHECK": os.Getenv("INTEGRATION_EMPTY_CHECK"),
	}
	defer func() {
		for key, val := range testVars {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Setup test environment
	os.Setenv("INTEGRATION_ENV_VAR", "from_environment")
	os.Setenv("INTEGRATION_OVERRIDE", "original_env")
	os.Unsetenv("INTEGRATION_UNSET")
	os.Setenv("INTEGRATION_EMPTY_CHECK", "")

	server := &config.Server{
		Env: map[string]config.EnvVar{
			// This will come from provided
			"INTEGRATION_PROVIDED": {
				Required:    true,
				Description: "Provided value takes priority",
			},
			// This will come from os.Getenv
			"INTEGRATION_ENV_VAR": {
				Required:    false,
				Description: "From OS environment",
			},
			// This will be overridden by provided
			"INTEGRATION_OVERRIDE": {
				Required:    false,
				Description: "Should be overridden",
			},
			// This will use default
			"INTEGRATION_DEFAULT": {
				Required:    false,
				Description: "Uses default value",
				Default:     "default_value",
			},
			// This will be missing (optional)
			"INTEGRATION_OPTIONAL_MISSING": {
				Required:    false,
				Description: "Optional and not provided",
			},
			// This has empty string in env, should use default
			"INTEGRATION_EMPTY_CHECK": {
				Required:    false,
				Description: "Empty in env, should use default",
				Default:     "fallback_default",
			},
		},
	}

	provided := map[string]string{
		"INTEGRATION_PROVIDED": "user_provided",
		"INTEGRATION_OVERRIDE": "overridden_value",
	}

	result, err := CollectEnvVars(server, provided, false)
	if err != nil {
		t.Fatalf("CollectEnvVars() error = %v", err)
	}

	// Verify all expected values
	tests := []struct {
		key      string
		want     string
		mustHave bool
	}{
		{"INTEGRATION_PROVIDED", "user_provided", true},
		{"INTEGRATION_ENV_VAR", "from_environment", true},
		{"INTEGRATION_OVERRIDE", "overridden_value", true},
		{"INTEGRATION_DEFAULT", "default_value", true},
		{"INTEGRATION_EMPTY_CHECK", "fallback_default", true},
		{"INTEGRATION_OPTIONAL_MISSING", "", false},
	}

	for _, tt := range tests {
		got, exists := result[tt.key]
		if tt.mustHave {
			if !exists {
				t.Errorf("Result missing required key %q", tt.key)
				continue
			}
			if got != tt.want {
				t.Errorf("Result[%q] = %q, want %q", tt.key, got, tt.want)
			}
		} else {
			if exists {
				t.Errorf("Result should not contain optional missing key %q, but has value %q", tt.key, got)
			}
		}
	}
}

// TestValidateMissing_EdgeCases tests edge cases in validation
func TestValidateMissing_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		server   *config.Server
		provided map[string]string
		wantLen  int
	}{
		{
			name: "all optional with empty provided",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"OPT1": {Required: false},
					"OPT2": {Required: false},
					"OPT3": {Required: false},
				},
			},
			provided: map[string]string{},
			wantLen:  0,
		},
		{
			name: "mix of provided and not provided required",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"REQ1": {Required: true},
					"REQ2": {Required: true},
					"REQ3": {Required: true},
					"REQ4": {Required: true},
				},
			},
			provided: map[string]string{
				"REQ1": "val1",
				"REQ3": "val3",
			},
			wantLen: 2, // REQ2 and REQ4 missing
		},
		{
			name:     "empty server env",
			server:   &config.Server{Env: map[string]config.EnvVar{}},
			provided: map[string]string{},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			missing := ValidateMissing(tt.server, tt.provided)
			if len(missing) != tt.wantLen {
				t.Errorf("ValidateMissing() returned %d items, want %d. Got: %v",
					len(missing), tt.wantLen, missing)
			}
		})
	}
}
