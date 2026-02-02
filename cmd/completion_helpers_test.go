package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

func TestServerNameCompletion(t *testing.T) {
	testConfig := `servers:
  server-a:
    description: First server
    transport: stdio
    command: /bin/test-a
  server-b:
    description: Second server
    transport: http
    url: http://example.com
  redis-cache:
    description: Redis server
    transport: stdio
    command: redis-cli
`

	tests := []struct {
		name           string
		args           []string
		toComplete     string
		expectedResult []string
		expectedDir    cobra.ShellCompDirective
	}{
		{
			name:           "complete first argument returns all server names",
			args:           []string{},
			toComplete:     "",
			expectedResult: []string{"server-a", "server-b", "redis-cache"},
			expectedDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:           "partial match still returns all servers",
			args:           []string{},
			toComplete:     "ser",
			expectedResult: []string{"server-a", "server-b", "redis-cache"},
			expectedDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:           "no completion when args already provided",
			args:           []string{"server-a"},
			toComplete:     "",
			expectedResult: nil,
			expectedDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:           "no completion when multiple args provided",
			args:           []string{"server-a", "extra"},
			toComplete:     "",
			expectedResult: nil,
			expectedDir:    cobra.ShellCompDirectiveNoFileComp,
		},
	}

	configPath, cleanup := setupTestConfig(t, testConfig)
	defer cleanup()

	// Set config file path
	cfgFile = configPath

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			results, directive := ServerNameCompletion(cmd, tt.args, tt.toComplete)

			if directive != tt.expectedDir {
				t.Errorf("Expected directive %v, got %v", tt.expectedDir, directive)
			}

			if len(results) != len(tt.expectedResult) {
				t.Errorf("Expected %d results, got %d. Results: %v", len(tt.expectedResult), len(results), results)
			}

			// Check that all expected results are present
			for _, expected := range tt.expectedResult {
				found := false
				for _, result := range results {
					if result == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected %q in results, got: %v", expected, results)
				}
			}
		})
	}
}

func TestServerNameCompletionNoConfig(t *testing.T) {
	// Reset config state to simulate no config
	oldCfg := cfg
	oldCfgFile := cfgFile
	defer func() {
		cfg = oldCfg
		cfgFile = oldCfgFile
	}()

	cfg = nil
	cfgFile = ""

	cmd := &cobra.Command{}
	results, directive := ServerNameCompletion(cmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("Expected NoFileComp directive, got %v", directive)
	}

	if len(results) != 0 {
		t.Errorf("Expected no results when config unavailable, got: %v", results)
	}
}

func TestScopeCompletion(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		toComplete     string
		expectedResult []string
		expectedDir    cobra.ShellCompDirective
	}{
		{
			name:           "complete scope flag values",
			args:           []string{},
			toComplete:     "",
			expectedResult: []string{"local", "user", "project"},
			expectedDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:           "partial scope match",
			args:           []string{},
			toComplete:     "l",
			expectedResult: []string{"local", "user", "project"},
			expectedDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:           "always return all scopes regardless of args",
			args:           []string{"some-arg"},
			toComplete:     "",
			expectedResult: []string{"local", "user", "project"},
			expectedDir:    cobra.ShellCompDirectiveNoFileComp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			results, directive := ScopeCompletion(cmd, tt.args, tt.toComplete)

			if directive != tt.expectedDir {
				t.Errorf("Expected directive %v, got %v", tt.expectedDir, directive)
			}

			if len(results) != len(tt.expectedResult) {
				t.Errorf("Expected %d results, got %d. Results: %v", len(tt.expectedResult), len(results), results)
			}

			// Check that all expected results are present in order
			for i, expected := range tt.expectedResult {
				if i >= len(results) {
					t.Errorf("Expected %q at index %d, but not enough results", expected, i)
					break
				}
				if results[i] != expected {
					t.Errorf("Expected %q at index %d, got %q", expected, i, results[i])
				}
			}
		})
	}
}

func TestServerNameCompletionIntegration(t *testing.T) {
	// Test that completion works with actual config loading
	testConfig := `default_scope: user
servers:
  github:
    description: GitHub API server
    transport: stdio
    command: github-mcp
  aws-bedrock:
    description: AWS Bedrock server
    transport: http
    url: http://localhost:9000
`

	configPath, cleanup := setupTestConfig(t, testConfig)
	defer cleanup()

	cfgFile = configPath

	cmd := &cobra.Command{}
	results, directive := ServerNameCompletion(cmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("Expected NoFileComp directive, got %v", directive)
	}

	expectedServers := []string{"github", "aws-bedrock"}
	if len(results) != len(expectedServers) {
		t.Errorf("Expected %d servers, got %d. Results: %v", len(expectedServers), len(results), results)
	}

	for _, expected := range expectedServers {
		found := false
		for _, result := range results {
			if result == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected server %q not found in results: %v", expected, results)
		}
	}
}

func TestScopeCompletionValues(t *testing.T) {
	// Ensure ScopeCompletion always returns exactly the expected scopes
	cmd := &cobra.Command{}
	results, _ := ScopeCompletion(cmd, []string{}, "")

	if len(results) != 3 {
		t.Errorf("Expected exactly 3 scope values, got %d", len(results))
	}

	expectedScopes := map[string]bool{
		"local":   false,
		"user":    false,
		"project": false,
	}

	for _, result := range results {
		if _, ok := expectedScopes[result]; !ok {
			t.Errorf("Unexpected scope value: %q", result)
		}
		expectedScopes[result] = true
	}

	for scope, found := range expectedScopes {
		if !found {
			t.Errorf("Expected scope %q not found in results", scope)
		}
	}
}

func TestCompletionWithCustomConfigPath(t *testing.T) {
	// Test ServerNameCompletion with custom config path
	testConfig := `servers:
  custom-server:
    description: Custom server
    transport: stdio
    command: /bin/custom
`

	configPath, cleanup := setupTestConfig(t, testConfig)
	defer cleanup()

	// Load config directly to test
	cfg, _ = config.LoadFromPath(configPath)

	cmd := &cobra.Command{}
	results, directive := ServerNameCompletion(cmd, []string{}, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("Expected NoFileComp directive, got %v", directive)
	}

	if len(results) == 0 {
		t.Errorf("Expected server names from custom config, got none")
	}

	found := false
	for _, result := range results {
		if result == "custom-server" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected custom-server in results: %v", results)
	}

	// Reset for next tests
	cfg = nil
}
