package env

import (
	"os"
	"testing"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

func TestIsSecret(t *testing.T) {
	tests := []struct {
		name     string
		envName  string
		expected bool
	}{
		// Secret patterns
		{"contains TOKEN", "API_TOKEN", true},
		{"contains token lowercase", "api_token", true},
		{"contains KEY", "API_KEY", true},
		{"contains key lowercase", "secret_key", true},
		{"contains SECRET", "MY_SECRET", true},
		{"contains secret lowercase", "app_secret", true},
		{"contains PASSWORD", "DB_PASSWORD", true},
		{"contains password lowercase", "user_password", true},
		{"contains CREDENTIAL", "AWS_CREDENTIAL", true},
		{"contains credential lowercase", "git_credential", true},
		{"contains AUTHORIZATION", "Authorization", true},
		{"contains authorization lowercase", "x-authorization", true},

		// Non-secret patterns
		{"plain name", "DEBUG", false},
		{"contains PORT", "SERVER_PORT", false},
		{"contains HOST", "DATABASE_HOST", false},
		{"contains PATH", "CONFIG_PATH", false},
		{"contains URL", "API_URL", false},
		{"contains NAME", "SERVICE_NAME", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSecret(tt.envName)
			if got != tt.expected {
				t.Errorf("IsSecret(%q) = %v, want %v", tt.envName, got, tt.expected)
			}
		})
	}
}

func TestValidateMissing(t *testing.T) {
	tests := []struct {
		name     string
		server   *config.Server
		provided map[string]string
		want     []string
	}{
		{
			name: "all required provided",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"REQUIRED1": {Required: true},
					"REQUIRED2": {Required: true},
					"OPTIONAL":  {Required: false},
				},
			},
			provided: map[string]string{
				"REQUIRED1": "value1",
				"REQUIRED2": "value2",
			},
			want: []string{},
		},
		{
			name: "some required missing",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"REQUIRED1": {Required: true},
					"REQUIRED2": {Required: true},
					"OPTIONAL":  {Required: false},
				},
			},
			provided: map[string]string{
				"REQUIRED1": "value1",
			},
			want: []string{"REQUIRED2"},
		},
		{
			name: "all required missing",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"REQUIRED1": {Required: true},
					"REQUIRED2": {Required: true},
				},
			},
			provided: map[string]string{},
			want:     []string{"REQUIRED1", "REQUIRED2"},
		},
		{
			name: "only optional vars",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"OPTIONAL1": {Required: false},
					"OPTIONAL2": {Required: false},
				},
			},
			provided: map[string]string{},
			want:     []string{},
		},
		{
			name: "no env vars",
			server: &config.Server{
				Env: map[string]config.EnvVar{},
			},
			provided: map[string]string{},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateMissing(tt.server, tt.provided)

			// Check length
			if len(got) != len(tt.want) {
				t.Errorf("ValidateMissing() returned %d items, want %d\nGot: %v\nWant: %v",
					len(got), len(tt.want), got, tt.want)
				return
			}

			// Check that all expected items are present (order may vary)
			gotMap := make(map[string]bool)
			for _, item := range got {
				gotMap[item] = true
			}

			for _, item := range tt.want {
				if !gotMap[item] {
					t.Errorf("ValidateMissing() missing expected item %q\nGot: %v\nWant: %v",
						item, got, tt.want)
				}
			}
		})
	}
}

func TestCollectEnvVars_NonInteractive(t *testing.T) {
	// Save original environment
	origValues := make(map[string]string)
	testEnvVars := []string{"TEST_FROM_OS", "TEST_OVERRIDE", "TEST_DEFAULT"}
	for _, key := range testEnvVars {
		origValues[key] = os.Getenv(key)
	}
	defer func() {
		for key, val := range origValues {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Set test environment variable
	os.Setenv("TEST_FROM_OS", "os_value")

	tests := []struct {
		name        string
		server      *config.Server
		provided    map[string]string
		wantErr     bool
		wantEnv     map[string]string
	}{
		{
			name: "priority: provided > os.Getenv > default",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"TEST_PROVIDED": {
						Required:    false,
						Description: "From provided",
					},
					"TEST_FROM_OS": {
						Required:    false,
						Description: "From OS env",
					},
					"TEST_DEFAULT": {
						Required:    false,
						Description: "From default",
						Default:     "default_value",
					},
					"TEST_OVERRIDE": {
						Required:    false,
						Description: "Override default with provided",
						Default:     "default_value",
					},
				},
			},
			provided: map[string]string{
				"TEST_PROVIDED": "provided_value",
				"TEST_OVERRIDE": "override_value",
			},
			wantErr: false,
			wantEnv: map[string]string{
				"TEST_PROVIDED": "provided_value",
				"TEST_FROM_OS":  "os_value",
				"TEST_DEFAULT":  "default_value",
				"TEST_OVERRIDE": "override_value",
			},
		},
		{
			name: "missing required returns error",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"REQUIRED_VAR": {
						Required:    true,
						Description: "Must be provided",
					},
				},
			},
			provided: map[string]string{},
			wantErr:  true,
		},
		{
			name: "required with provided value succeeds",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"REQUIRED_VAR": {
						Required:    true,
						Description: "Must be provided",
					},
				},
			},
			provided: map[string]string{
				"REQUIRED_VAR": "provided",
			},
			wantErr: false,
			wantEnv: map[string]string{
				"REQUIRED_VAR": "provided",
			},
		},
		{
			name: "required with default value succeeds",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"REQUIRED_WITH_DEFAULT": {
						Required:    true,
						Description: "Required but has default",
						Default:     "default_val",
					},
				},
			},
			provided: map[string]string{},
			wantErr:  false,
			wantEnv: map[string]string{
				"REQUIRED_WITH_DEFAULT": "default_val",
			},
		},
		{
			name: "empty provided value ignored",
			server: &config.Server{
				Env: map[string]config.EnvVar{
					"VAR_WITH_EMPTY": {
						Required:    false,
						Description: "Has empty provided",
						Default:     "default_val",
					},
				},
			},
			provided: map[string]string{
				"VAR_WITH_EMPTY": "",
			},
			wantErr: false,
			wantEnv: map[string]string{
				"VAR_WITH_EMPTY": "default_val",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CollectEnvVars(tt.server, tt.provided, false)

			if (err != nil) != tt.wantErr {
				t.Errorf("CollectEnvVars() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Check that all expected vars are present with correct values
			for key, wantVal := range tt.wantEnv {
				gotVal, exists := got[key]
				if !exists {
					t.Errorf("CollectEnvVars() missing key %q", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("CollectEnvVars()[%q] = %q, want %q", key, gotVal, wantVal)
				}
			}

			// Check that no unexpected vars are present
			for key := range got {
				if _, expected := tt.wantEnv[key]; !expected {
					t.Errorf("CollectEnvVars() unexpected key %q = %q", key, got[key])
				}
			}
		})
	}
}

func TestCollectEnvVars_OsGetenvPriority(t *testing.T) {
	// Save and set test environment variable
	origValue := os.Getenv("TEST_OS_VAR")
	defer func() {
		if origValue == "" {
			os.Unsetenv("TEST_OS_VAR")
		} else {
			os.Setenv("TEST_OS_VAR", origValue)
		}
	}()

	os.Setenv("TEST_OS_VAR", "from_os")

	server := &config.Server{
		Env: map[string]config.EnvVar{
			"TEST_OS_VAR": {
				Required:    false,
				Description: "From OS",
				Default:     "from_default",
			},
		},
	}

	// Test that OS env takes priority over default
	got, err := CollectEnvVars(server, map[string]string{}, false)
	if err != nil {
		t.Fatalf("CollectEnvVars() error = %v", err)
	}

	if got["TEST_OS_VAR"] != "from_os" {
		t.Errorf("CollectEnvVars()[TEST_OS_VAR] = %q, want %q (os.Getenv should override default)",
			got["TEST_OS_VAR"], "from_os")
	}

	// Test that provided takes priority over OS env
	got, err = CollectEnvVars(server, map[string]string{"TEST_OS_VAR": "from_provided"}, false)
	if err != nil {
		t.Fatalf("CollectEnvVars() error = %v", err)
	}

	if got["TEST_OS_VAR"] != "from_provided" {
		t.Errorf("CollectEnvVars()[TEST_OS_VAR] = %q, want %q (provided should override os.Getenv)",
			got["TEST_OS_VAR"], "from_provided")
	}
}

func TestCollectEnvVars_NoEnvVars(t *testing.T) {
	server := &config.Server{
		Env: map[string]config.EnvVar{},
	}

	got, err := CollectEnvVars(server, map[string]string{}, false)
	if err != nil {
		t.Errorf("CollectEnvVars() with no env vars error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("CollectEnvVars() with no env vars returned %d items, want 0", len(got))
	}
}

func TestCollectEnvVars_AllOptional(t *testing.T) {
	server := &config.Server{
		Env: map[string]config.EnvVar{
			"OPTIONAL1": {
				Required:    false,
				Description: "Optional 1",
			},
			"OPTIONAL2": {
				Required:    false,
				Description: "Optional 2",
				Default:     "default2",
			},
		},
	}

	got, err := CollectEnvVars(server, map[string]string{}, false)
	if err != nil {
		t.Errorf("CollectEnvVars() with all optional error = %v", err)
	}

	// Should only have OPTIONAL2 with default value
	if len(got) != 1 {
		t.Errorf("CollectEnvVars() returned %d items, want 1", len(got))
	}

	if got["OPTIONAL2"] != "default2" {
		t.Errorf("CollectEnvVars()[OPTIONAL2] = %q, want %q", got["OPTIONAL2"], "default2")
	}
}

func TestValidateMissing_WithNilEnv(t *testing.T) {
	server := &config.Server{
		Env: nil,
	}

	missing := ValidateMissing(server, map[string]string{})
	if len(missing) != 0 {
		t.Errorf("ValidateMissing() with nil env returned %d items, want 0", len(missing))
	}
}

func TestCollectEnvVars_RequiredFromOsEnv(t *testing.T) {
	// Save and set test environment variable
	origValue := os.Getenv("REQUIRED_FROM_OS")
	defer func() {
		if origValue == "" {
			os.Unsetenv("REQUIRED_FROM_OS")
		} else {
			os.Setenv("REQUIRED_FROM_OS", origValue)
		}
	}()

	os.Setenv("REQUIRED_FROM_OS", "os_value")

	server := &config.Server{
		Env: map[string]config.EnvVar{
			"REQUIRED_FROM_OS": {
				Required:    true,
				Description: "Required from OS",
			},
		},
	}

	got, err := CollectEnvVars(server, map[string]string{}, false)
	if err != nil {
		t.Fatalf("CollectEnvVars() error = %v, expected required var from os.Getenv", err)
	}

	if got["REQUIRED_FROM_OS"] != "os_value" {
		t.Errorf("CollectEnvVars()[REQUIRED_FROM_OS] = %q, want %q", got["REQUIRED_FROM_OS"], "os_value")
	}
}

func TestCollectEnvVars_MultipleVarsFromDifferentSources(t *testing.T) {
	// Save and restore environment
	origEnvVar := os.Getenv("FROM_ENV")
	defer func() {
		if origEnvVar == "" {
			os.Unsetenv("FROM_ENV")
		} else {
			os.Setenv("FROM_ENV", origEnvVar)
		}
	}()

	os.Setenv("FROM_ENV", "env_value")

	server := &config.Server{
		Env: map[string]config.EnvVar{
			"FROM_PROVIDED": {
				Required:    true,
				Description: "Provided directly",
			},
			"FROM_ENV": {
				Required:    false,
				Description: "From environment",
			},
			"FROM_DEFAULT": {
				Required:    false,
				Description: "From default",
				Default:     "default_value",
			},
			"OPTIONAL_MISSING": {
				Required:    false,
				Description: "Optional and not provided",
			},
		},
	}

	provided := map[string]string{
		"FROM_PROVIDED": "provided_value",
	}

	got, err := CollectEnvVars(server, provided, false)
	if err != nil {
		t.Fatalf("CollectEnvVars() error = %v", err)
	}

	expected := map[string]string{
		"FROM_PROVIDED": "provided_value",
		"FROM_ENV":      "env_value",
		"FROM_DEFAULT":  "default_value",
	}

	for key, wantVal := range expected {
		gotVal, exists := got[key]
		if !exists {
			t.Errorf("CollectEnvVars() missing key %q", key)
			continue
		}
		if gotVal != wantVal {
			t.Errorf("CollectEnvVars()[%q] = %q, want %q", key, gotVal, wantVal)
		}
	}

	// OPTIONAL_MISSING should not be present
	if _, exists := got["OPTIONAL_MISSING"]; exists {
		t.Errorf("CollectEnvVars() should not contain OPTIONAL_MISSING")
	}
}

func TestCollectEnvVars_EmptyDefault(t *testing.T) {
	server := &config.Server{
		Env: map[string]config.EnvVar{
			"VAR_WITH_EMPTY_DEFAULT": {
				Required:    false,
				Description: "Has empty default",
				Default:     "",
			},
		},
	}

	got, err := CollectEnvVars(server, map[string]string{}, false)
	if err != nil {
		t.Errorf("CollectEnvVars() error = %v", err)
	}

	// Should not include var with empty default
	if _, exists := got["VAR_WITH_EMPTY_DEFAULT"]; exists {
		t.Errorf("CollectEnvVars() should not contain var with empty default")
	}
}

func TestValidateMissing_MultipleRequired(t *testing.T) {
	server := &config.Server{
		Env: map[string]config.EnvVar{
			"REQ1": {Required: true, Description: "Required 1"},
			"REQ2": {Required: true, Description: "Required 2"},
			"REQ3": {Required: true, Description: "Required 3"},
			"OPT1": {Required: false, Description: "Optional 1"},
		},
	}

	provided := map[string]string{
		"REQ1": "value1",
		// REQ2 and REQ3 missing
		"OPT1": "opt_value",
	}

	missing := ValidateMissing(server, provided)

	// Should have exactly 2 missing
	if len(missing) != 2 {
		t.Errorf("ValidateMissing() returned %d missing, want 2", len(missing))
	}

	// Check both REQ2 and REQ3 are in missing
	missingMap := make(map[string]bool)
	for _, m := range missing {
		missingMap[m] = true
	}

	if !missingMap["REQ2"] {
		t.Errorf("ValidateMissing() should include REQ2")
	}
	if !missingMap["REQ3"] {
		t.Errorf("ValidateMissing() should include REQ3")
	}
}
