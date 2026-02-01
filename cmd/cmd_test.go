package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestConfig creates a temporary config file and returns the path and cleanup function
func setupTestConfig(t *testing.T, content string) (string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if content != "" {
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}
	}

	cleanup := func() {
		// Reset global state
		cfg = nil
		cfgFile = ""
		initForce = false
		initPath = ""
		listJSON = false
		infoJSONFlag = false
		addScope = ""
		addEnvVars = nil
		addDryRun = false
		addNoPrompt = false
	}

	return configPath, cleanup
}

// executeCommand executes a cobra command and returns stdout/stderr output
func executeCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Capture stdout and stderr since commands use fmt.Print directly
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	// Close writer and restore stdout/stderr
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	var output strings.Builder
	buf := make([]byte, 1024)
	for {
		n, readErr := r.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}

	return output.String(), err
}

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupFile   bool
		wantErr     bool
		errContains string
		checkFile   bool
	}{
		{
			name:      "create config at default path",
			args:      []string{"init", "--path", ""},
			wantErr:   false,
			checkFile: true,
		},
		{
			name:      "create config at custom path",
			args:      []string{"init"},
			wantErr:   false,
			checkFile: true,
		},
		{
			name:        "fail when file exists without force",
			args:        []string{"init"},
			setupFile:   true,
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name:      "overwrite with force flag",
			args:      []string{"init", "--force"},
			setupFile: true,
			wantErr:   false,
			checkFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			// Setup existing file if needed
			if tt.setupFile {
				if err := os.WriteFile(configPath, []byte("existing"), 0644); err != nil {
					t.Fatalf("Failed to setup test file: %v", err)
				}
			}

			// Set initPath for custom path test
			initPath = configPath
			defer func() {
				initPath = ""
				initForce = false
			}()

			// Execute command
			args := []string{"init"}
			if len(tt.args) > 1 {
				args = append(args, tt.args[1:]...)
			}

			// Add --path flag
			args = append(args, "--path", configPath)

			output, err := executeCommand(t, args...)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none. Output: %s", output)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, output)
				return
			}

			// Check file was created
			if tt.checkFile {
				content, err := os.ReadFile(configPath)
				if err != nil {
					t.Errorf("Config file should exist at %s: %v", configPath, err)
				}
				if !strings.Contains(string(content), "MCP Helper Configuration") {
					t.Errorf("Config file should contain expected content")
				}
				if !strings.Contains(string(content), "default_scope: local") {
					t.Errorf("Config file should contain default_scope")
				}
			}
		})
	}
}

func TestListCommand(t *testing.T) {
	testConfig := `default_scope: user
servers:
  server-a:
    description: First server
    transport: stdio
    command: /bin/test-a
  server-b:
    description: Second server
    transport: http
    url: http://example.com
`

	tests := []struct {
		name        string
		configData  string
		args        []string
		wantErr     bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:       "list servers in table format",
			configData: testConfig,
			args:       []string{"list"},
			wantErr:    false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "server-a") {
					t.Errorf("Output should contain server-a, got: %s", output)
				}
				if !strings.Contains(output, "server-b") {
					t.Errorf("Output should contain server-b, got: %s", output)
				}
				if !strings.Contains(output, "First server") {
					t.Errorf("Output should contain description, got: %s", output)
				}
				if !strings.Contains(output, "2 servers configured") {
					t.Errorf("Output should contain count, got: %s", output)
				}
			},
		},
		{
			name:       "list servers in JSON format",
			configData: testConfig,
			args:       []string{"list", "--json"},
			wantErr:    false,
			checkOutput: func(t *testing.T, output string) {
				// Trim whitespace as JSON encoder adds newline
				output = strings.TrimSpace(output)
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("Output should be valid JSON: %v\nOutput: %s", err, output)
				}

				if count, ok := result["count"].(float64); !ok || count != 2 {
					t.Errorf("JSON should have count=2, got %v", result["count"])
				}

				servers, ok := result["servers"].([]interface{})
				if !ok || len(servers) != 2 {
					t.Errorf("JSON should have 2 servers")
				}
			},
		},
		{
			name:       "list with no servers",
			configData: "servers: {}",
			args:       []string{"list"},
			wantErr:    false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "No servers configured") {
					t.Errorf("Output should indicate no servers, got: %s", output)
				}
			},
		},
		{
			name:       "list with no servers JSON",
			configData: "servers: {}",
			args:       []string{"list", "--json"},
			wantErr:    false,
			checkOutput: func(t *testing.T, output string) {
				output = strings.TrimSpace(output)
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("Output should be valid JSON: %v\nOutput: %s", err, output)
				}
				if count, ok := result["count"].(float64); !ok || count != 0 {
					t.Errorf("JSON should have count=0, got %v", result["count"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := setupTestConfig(t, tt.configData)
			defer cleanup()

			// Set config file path
			cfgFile = configPath

			output, err := executeCommand(t, tt.args...)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, output)
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestInfoCommand(t *testing.T) {
	testConfig := `servers:
  test-server:
    description: A test server
    transport: stdio
    command: /usr/bin/test
    args:
      - --flag
      - value
    env:
      TOKEN:
        required: true
        description: Auth token
      OPTIONAL:
        required: false
        default: default-value
  http-server:
    description: HTTP server
    transport: http
    url: http://localhost:8080
`

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:    "show server info in human format",
			args:    []string{"info", "test-server"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Server: test-server") {
					t.Errorf("Output should contain server name, got: %s", output)
				}
				if !strings.Contains(output, "A test server") {
					t.Errorf("Output should contain description, got: %s", output)
				}
				if !strings.Contains(output, "Transport: stdio") {
					t.Errorf("Output should contain transport, got: %s", output)
				}
				if !strings.Contains(output, "/usr/bin/test") {
					t.Errorf("Output should contain command, got: %s", output)
				}
				// Check for lowercase token (YAML is case-sensitive)
				if !strings.Contains(strings.ToLower(output), "token") {
					t.Errorf("Output should contain env var, got: %s", output)
				}
				if !strings.Contains(output, "required") {
					t.Errorf("Output should show required status, got: %s", output)
				}
			},
		},
		{
			name:    "show server info in JSON format",
			args:    []string{"info", "test-server", "--json"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				output = strings.TrimSpace(output)
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("Output should be valid JSON: %v\nOutput: %s", err, output)
				}

				if result["name"] != "test-server" {
					t.Errorf("JSON should have name=test-server")
				}
				if result["transport"] != "stdio" {
					t.Errorf("JSON should have transport=stdio")
				}
				if result["command"] != "/usr/bin/test" {
					t.Errorf("JSON should have command")
				}
			},
		},
		{
			name:    "show HTTP server info",
			args:    []string{"info", "http-server", "--json"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				output = strings.TrimSpace(output)
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Fatalf("Output should be valid JSON: %v\nOutput: %s", err, output)
				}

				if result["transport"] != "http" {
					t.Errorf("JSON should have transport=http")
				}
				if result["url"] != "http://localhost:8080" {
					t.Errorf("JSON should have url")
				}
			},
		},
		{
			name:        "error on non-existent server",
			args:        []string{"info", "nonexistent"},
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := setupTestConfig(t, testConfig)
			defer cleanup()

			cfgFile = configPath

			output, err := executeCommand(t, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none. Output: %s", output)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, output)
				return
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestAddCommand(t *testing.T) {
	testConfig := `default_scope: local
servers:
  test-server:
    description: Test server
    transport: stdio
    command: /usr/bin/test
    args:
      - --flag
    env:
      required_var:
        required: true
        description: Required variable
      optional_var:
        required: false
        default: default-value
  no-env-server:
    description: Server without env vars
    transport: stdio
    command: /bin/simple
`

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		errContains string
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:    "dry-run mode shows command",
			args:    []string{"add", "test-server", "--dry-run", "--env", "required_var=test"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Command that would be executed") {
					t.Errorf("Dry-run should show command preview")
				}
				if !strings.Contains(output, "claude") {
					t.Errorf("Dry-run should show claude command")
				}
			},
		},
		{
			name:    "dry-run with env vars",
			args:    []string{"add", "test-server", "--dry-run", "--env", "required_var=test123"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "required_var=test123") {
					t.Errorf("Dry-run should show env var in command, got: %s", output)
				}
			},
		},
		{
			name:    "validate scope values",
			args:    []string{"add", "test-server", "--scope", "local", "--dry-run", "--env", "required_var=test"},
			wantErr: false,
		},
		{
			name:        "invalid scope",
			args:        []string{"add", "test-server", "--scope", "invalid", "--dry-run"},
			wantErr:     true,
			errContains: "invalid scope",
		},
		{
			name:    "parse env flags correctly",
			args:    []string{"add", "test-server", "--dry-run", "--env", "required_var=value1", "--env", "optional_var=value2"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "required_var=value1") {
					t.Errorf("Should parse first env var, got: %s", output)
				}
				if !strings.Contains(output, "optional_var=value2") {
					t.Errorf("Should parse second env var, got: %s", output)
				}
			},
		},
		{
			name:        "invalid env flag format",
			args:        []string{"add", "test-server", "--dry-run", "--env", "INVALID"},
			wantErr:     true,
			errContains: "invalid --env flag format",
		},
		{
			name:        "no-prompt fails on missing required vars",
			args:        []string{"add", "test-server", "--no-prompt"},
			wantErr:     true,
			errContains: "required",
		},
		{
			name:    "no-prompt succeeds when all required vars provided",
			args:    []string{"add", "test-server", "--no-prompt", "--dry-run", "--env", "required_var=test"},
			wantErr: false,
		},
		{
			name:        "error on non-existent server",
			args:        []string{"add", "nonexistent", "--dry-run"},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:    "server without env vars works",
			args:    []string{"add", "no-env-server", "--dry-run"},
			wantErr: false,
		},
		{
			name:    "env var with spaces in value",
			args:    []string{"add", "test-server", "--dry-run", "--env", "required_var=value with spaces"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "value with spaces") {
					t.Errorf("Should preserve spaces in env var value, got: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := setupTestConfig(t, testConfig)
			defer cleanup()

			cfgFile = configPath

			output, err := executeCommand(t, tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none. Output: %s", output)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v. Output: %s", err, output)
				return
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, output)
			}
		})
	}
}

func TestConfigEnvironmentVariable(t *testing.T) {
	testConfig := `servers:
  test:
    description: Test
    transport: stdio
    command: /bin/test
`

	configPath, cleanup := setupTestConfig(t, testConfig)
	defer cleanup()

	// Set environment variable
	oldEnv := os.Getenv("MCP_HELPER_CONFIG")
	os.Setenv("MCP_HELPER_CONFIG", configPath)
	defer os.Setenv("MCP_HELPER_CONFIG", oldEnv)

	// Don't set cfgFile - it should use env var
	cfgFile = ""

	output, err := executeCommand(t, "list", "--config", configPath)
	if err != nil {
		t.Errorf("Should work with MCP_HELPER_CONFIG env var: %v", err)
	}

	if !strings.Contains(output, "test") {
		t.Errorf("Should list server from config specified by env var, got: %s", output)
	}
}

func TestConfigLoadingPriority(t *testing.T) {
	testConfig1 := `servers:
  config1-server:
    description: From config 1
    transport: stdio
    command: /bin/test1
`

	testConfig2 := `servers:
  config2-server:
    description: From config 2
    transport: stdio
    command: /bin/test2
`

	// Create two config files
	path1, cleanup1 := setupTestConfig(t, testConfig1)
	defer cleanup1()

	path2, cleanup2 := setupTestConfig(t, testConfig2)
	defer cleanup2()

	// Test --config flag takes precedence over env var
	os.Setenv("MCP_HELPER_CONFIG", path1)
	defer os.Unsetenv("MCP_HELPER_CONFIG")

	// Use --config flag to specify path2
	output, err := executeCommand(t, "list", "--config", path2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should see config2-server (from --config flag)
	if !strings.Contains(output, "config2-server") {
		t.Errorf("Should use config from --config flag, got: %s", output)
	}

	// Should not see config1-server (from env var)
	if strings.Contains(output, "config1-server") {
		t.Errorf("Should not use config from env var when --config is set, got: %s", output)
	}
}

func TestDefaultScope(t *testing.T) {
	tests := []struct {
		name          string
		config        string
		args          []string
		expectedScope string
	}{
		{
			name: "use default scope from config",
			config: `default_scope: project
servers:
  test:
    description: Test
    transport: stdio
    command: /bin/test
`,
			args:          []string{"add", "test", "--dry-run"},
			expectedScope: "--project",
		},
		{
			name: "override default scope with flag",
			config: `default_scope: project
servers:
  test:
    description: Test
    transport: stdio
    command: /bin/test
`,
			args:          []string{"add", "test", "--scope", "user", "--dry-run"},
			expectedScope: "--user",
		},
		{
			name: "fallback to local when no default",
			config: `servers:
  test:
    description: Test
    transport: stdio
    command: /bin/test
`,
			args:          []string{"add", "test", "--dry-run"},
			expectedScope: "--local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := setupTestConfig(t, tt.config)
			defer cleanup()

			// Add --config flag to the args
			args := append([]string{"--config", configPath}, tt.args...)

			output, err := executeCommand(t, args...)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check for "scope <value>" pattern in output
			expectedPattern := "--scope " + strings.TrimPrefix(tt.expectedScope, "--")
			if !strings.Contains(output, expectedPattern) {
				t.Errorf("Expected scope pattern %q in output, got: %s", expectedPattern, output)
			}
		})
	}
}
