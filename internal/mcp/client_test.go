package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewClient verifies client initialization with defaults
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		cache   *Cache
		wantNil bool
	}{
		{
			name:    "with valid timeout and cache",
			timeout: 5 * time.Second,
			cache:   NewCache(DefaultCacheTTL),
			wantNil: false,
		},
		{
			name:    "with zero timeout defaults to DefaultTimeout",
			timeout: 0,
			cache:   NewCache(DefaultCacheTTL),
			wantNil: false,
		},
		{
			name:    "with nil cache creates default cache",
			timeout: 5 * time.Second,
			cache:   nil,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.timeout, tt.cache)
			if (client == nil) != tt.wantNil {
				t.Errorf("NewClient() = %v, wantNil %v", client, tt.wantNil)
			}
			if client != nil && client.cache == nil {
				t.Error("NewClient() cache is nil")
			}
			if tt.timeout == 0 && client.timeout != DefaultTimeout {
				t.Errorf("NewClient() timeout = %v, want %v", client.timeout, DefaultTimeout)
			}
		})
	}
}

// TestListToolsStdio_HandshakeSequence tests the full MCP handshake
func TestListToolsStdio_HandshakeSequence(t *testing.T) {
	// Create a mock MCP server script
	mockServer := createMockMCPServer(t, mockServerConfig{
		tools: []Tool{
			{Name: "test_tool", Description: "A test tool"},
		},
	})
	defer os.Remove(mockServer)

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	tools, err := client.ListToolsStdio(ctx, mockServer, []string{}, nil)
	if err != nil {
		t.Fatalf("ListToolsStdio() error = %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("ListToolsStdio() returned %d tools, want 1", len(tools))
	}

	if tools[0].Name != "test_tool" {
		t.Errorf("ListToolsStdio() tool name = %q, want %q", tools[0].Name, "test_tool")
	}
}

// TestListToolsStdio_Pagination tests nextCursor handling
func TestListToolsStdio_Pagination(t *testing.T) {
	mockServer := createMockMCPServer(t, mockServerConfig{
		tools: []Tool{
			{Name: "tool1", Description: "First tool"},
			{Name: "tool2", Description: "Second tool"},
			{Name: "tool3", Description: "Third tool"},
		},
		paginate:     true,
		toolsPerPage: 2,
	})
	defer os.Remove(mockServer)

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	tools, err := client.ListToolsStdio(ctx, mockServer, []string{}, nil)
	if err != nil {
		t.Fatalf("ListToolsStdio() with pagination error = %v", err)
	}

	if len(tools) != 3 {
		t.Errorf("ListToolsStdio() returned %d tools, want 3", len(tools))
	}
}

// TestListToolsStdio_Timeout tests timeout handling
func TestListToolsStdio_Timeout(t *testing.T) {
	mockServer := createMockMCPServer(t, mockServerConfig{
		delay: 5 * time.Second, // Longer than client timeout
	})
	defer os.Remove(mockServer)

	client := NewClient(100*time.Millisecond, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	_, err := client.ListToolsStdio(ctx, mockServer, []string{}, nil)
	if err == nil {
		t.Error("ListToolsStdio() expected timeout error, got nil")
	}
}

// TestListToolsStdio_JSONRPCError tests JSON-RPC error responses
func TestListToolsStdio_JSONRPCError(t *testing.T) {
	mockServer := createMockMCPServer(t, mockServerConfig{
		errorOnToolsList: true,
		errorCode:        -32601,
		errorMessage:     "Method not found",
	})
	defer os.Remove(mockServer)

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	_, err := client.ListToolsStdio(ctx, mockServer, []string{}, nil)
	if err == nil {
		t.Error("ListToolsStdio() expected JSON-RPC error, got nil")
	}
	if !strings.Contains(err.Error(), "Method not found") {
		t.Errorf("ListToolsStdio() error = %v, want error containing 'Method not found'", err)
	}
}

// TestListToolsStdio_MalformedResponse tests handling of malformed JSON
func TestListToolsStdio_MalformedResponse(t *testing.T) {
	mockServer := createMockMCPServer(t, mockServerConfig{
		malformedJSON: true,
	})
	defer os.Remove(mockServer)

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	_, err := client.ListToolsStdio(ctx, mockServer, []string{}, nil)
	if err == nil {
		t.Error("ListToolsStdio() expected malformed JSON error, got nil")
	}
}

// TestDiscoverTools tests concurrent tool discovery
func TestDiscoverTools(t *testing.T) {
	server1 := createMockMCPServer(t, mockServerConfig{
		tools: []Tool{{Name: "server1_tool"}},
	})
	defer os.Remove(server1)

	server2 := createMockMCPServer(t, mockServerConfig{
		tools: []Tool{{Name: "server2_tool"}},
	})
	defer os.Remove(server2)

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	servers := []ServerConfig{
		{Name: "server1", Scope: "user", Transport: "stdio", Command: server1, Args: []string{}},
		{Name: "server2", Scope: "project", Transport: "stdio", Command: server2, Args: []string{}},
	}

	results := client.DiscoverTools(ctx, servers, false)

	if len(results) != 2 {
		t.Fatalf("DiscoverTools() returned %d results, want 2", len(results))
	}

	// Verify first server
	if results[0].Name != "server1" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "server1")
	}
	if results[0].Error != nil {
		t.Errorf("results[0].Error = %v, want nil", results[0].Error)
	}
	if len(results[0].Tools) != 1 || results[0].Tools[0].Name != "server1_tool" {
		t.Errorf("results[0].Tools incorrect, got %v", results[0].Tools)
	}

	// Verify second server
	if results[1].Name != "server2" {
		t.Errorf("results[1].Name = %q, want %q", results[1].Name, "server2")
	}
}

// TestDiscoverTools_WithCache tests cache hit/miss behavior
func TestDiscoverTools_WithCache(t *testing.T) {
	mockServer := createMockMCPServer(t, mockServerConfig{
		tools: []Tool{{Name: "cached_tool"}},
	})
	defer os.Remove(mockServer)

	cache := NewCache(1 * time.Hour)
	client := NewClient(10*time.Second, cache)
	ctx := context.Background()

	servers := []ServerConfig{
		{Name: "test-server", Scope: "user", Transport: "stdio", Command: mockServer, Args: []string{}},
	}

	// First call - cache miss
	results1 := client.DiscoverTools(ctx, servers, true)
	if len(results1) != 1 || results1[0].Error != nil {
		t.Fatalf("First DiscoverTools() failed: %v", results1[0].Error)
	}

	// Verify cache was populated
	cached := cache.Get("user", "test-server")
	if cached == nil {
		t.Error("Cache was not populated after first call")
	}

	// Second call - cache hit (change server to verify cache is used)
	servers[0].Command = "/nonexistent/command" // This would fail if cache isn't used
	results2 := client.DiscoverTools(ctx, servers, true)
	if len(results2) != 1 || results2[0].Error != nil {
		t.Error("Second DiscoverTools() should use cache and succeed")
	}
	if len(results2[0].Tools) != 1 || results2[0].Tools[0].Name != "cached_tool" {
		t.Error("Cached tools were not returned")
	}
}

// TestDiscoverTools_PartialFailure tests that one server failure doesn't block others
func TestDiscoverTools_PartialFailure(t *testing.T) {
	goodServer := createMockMCPServer(t, mockServerConfig{
		tools: []Tool{{Name: "good_tool"}},
	})
	defer os.Remove(goodServer)

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	servers := []ServerConfig{
		{Name: "good", Scope: "user", Transport: "stdio", Command: goodServer, Args: []string{}},
		{Name: "bad", Scope: "user", Transport: "stdio", Command: "/nonexistent/command", Args: []string{}},
	}

	results := client.DiscoverTools(ctx, servers, false)

	if len(results) != 2 {
		t.Fatalf("DiscoverTools() returned %d results, want 2", len(results))
	}

	// Good server should succeed
	var goodResult, badResult *ServerInfo
	for i := range results {
		if results[i].Name == "good" {
			goodResult = &results[i]
		} else if results[i].Name == "bad" {
			badResult = &results[i]
		}
	}

	if goodResult == nil || badResult == nil {
		t.Fatal("Could not find good or bad results")
	}

	if goodResult.Error != nil {
		t.Errorf("good server returned error: %v", goodResult.Error)
	}
	if len(goodResult.Tools) != 1 {
		t.Errorf("good server returned %d tools, want 1", len(goodResult.Tools))
	}

	// Bad server should fail
	if badResult.Error == nil {
		t.Error("bad server should return error")
	}
	if len(badResult.Tools) != 0 {
		t.Errorf("bad server returned %d tools, want 0", len(badResult.Tools))
	}
}

// Mock server configuration
type mockServerConfig struct {
	tools            []Tool
	paginate         bool
	toolsPerPage     int
	delay            time.Duration
	errorOnToolsList bool
	errorCode        int
	errorMessage     string
	malformedJSON    bool
}

// createMockMCPServer creates a script that simulates an MCP server
func createMockMCPServer(t *testing.T, config mockServerConfig) string {
	t.Helper()

	// Create temp directory for mock server
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock_mcp_server.sh")

	script := `#!/bin/bash
set -e

# Read and parse initialize request
read -r line
`

	if config.delay > 0 {
		script += fmt.Sprintf("sleep %f\n", config.delay.Seconds())
	}

	// Initialize response
	script += `echo '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"mock-server","version":"1.0.0"}}}'
`

	// Read initialized notification
	script += `read -r line
`

	// Tools list response
	if config.malformedJSON {
		script += `echo '{malformed json'
`
	} else if config.errorOnToolsList {
		script += fmt.Sprintf(`echo '{"jsonrpc":"2.0","id":2,"error":{"code":%d,"message":"%s"}}'
`, config.errorCode, config.errorMessage)
	} else if config.paginate && len(config.tools) > 0 {
		// First page
		page1Tools := config.tools[:config.toolsPerPage]
		page1JSON, _ := json.Marshal(page1Tools)
		script += fmt.Sprintf(`read -r line
echo '{"jsonrpc":"2.0","id":2,"result":{"tools":%s,"nextCursor":"page2"}}'
`, page1JSON)

		// Second page
		page2Tools := config.tools[config.toolsPerPage:]
		page2JSON, _ := json.Marshal(page2Tools)
		script += fmt.Sprintf(`read -r line
echo '{"jsonrpc":"2.0","id":3,"result":{"tools":%s}}'
`, page2JSON)
	} else {
		toolsJSON, _ := json.Marshal(config.tools)
		script += fmt.Sprintf(`read -r line
echo '{"jsonrpc":"2.0","id":2,"result":{"tools":%s}}'
`, toolsJSON)
	}

	// Write script
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create mock server script: %v", err)
	}

	return scriptPath
}

// TestBuildEnvSlice verifies environment variable merging
func TestBuildEnvSlice(t *testing.T) {
	// Save original env
	originalEnv := os.Environ()

	envMap := map[string]string{
		"TEST_VAR": "test_value",
		"API_KEY":  "secret123",
	}

	result := buildEnvSlice(envMap)

	// Should contain original env vars
	if len(result) < len(originalEnv) {
		t.Errorf("buildEnvSlice() returned %d vars, want at least %d", len(result), len(originalEnv))
	}

	// Should contain new vars
	hasTestVar := false
	hasAPIKey := false
	for _, v := range result {
		if v == "TEST_VAR=test_value" {
			hasTestVar = true
		}
		if v == "API_KEY=secret123" {
			hasAPIKey = true
		}
	}

	if !hasTestVar {
		t.Error("buildEnvSlice() missing TEST_VAR")
	}
	if !hasAPIKey {
		t.Error("buildEnvSlice() missing API_KEY")
	}
}

// TestGracefulShutdown tests process cleanup
func TestGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping graceful shutdown test in short mode")
	}

	// Create a long-running process
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))

	// Test graceful shutdown
	client.gracefulShutdown(cmd)

	// Process should be terminated
	if cmd.Process != nil {
		// Try to signal - should fail if process is dead
		err := cmd.Process.Signal(os.Kill)
		if err == nil {
			t.Error("Process still running after gracefulShutdown()")
			cmd.Process.Kill() // Clean up
		}
	}
}

// TestListToolsStdio_EnvVars tests environment variable passing
func TestListToolsStdio_EnvVars(t *testing.T) {
	// Create a mock server that echoes an env var
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "env_echo_server.sh")

	script := `#!/bin/bash
# Initialize response
read -r line
echo '{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"env-test","version":"1.0.0"}}}'

# Initialized notification
read -r line

# Tools list - include env var in description
read -r line
echo "{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"tools\":[{\"name\":\"test\",\"description\":\"TEST_ENV_VAR=$TEST_ENV_VAR\"}]}}"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create env echo server: %v", err)
	}

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	env := map[string]string{
		"TEST_ENV_VAR": "test_value_123",
	}

	tools, err := client.ListToolsStdio(ctx, scriptPath, []string{}, env)
	if err != nil {
		t.Fatalf("ListToolsStdio() error = %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	// Check if env var was passed correctly
	if !strings.Contains(tools[0].Description, "test_value_123") {
		t.Errorf("Tool description = %q, should contain env var value", tools[0].Description)
	}
}

// TestInitializeServer_ProtocolVersion verifies correct protocol version is sent
func TestInitializeServer_ProtocolVersion(t *testing.T) {
	// We can't easily test this in isolation without mocking,
	// but we verify the constant is set correctly
	if ProtocolVersion != "2024-11-05" {
		t.Errorf("ProtocolVersion = %q, want %q", ProtocolVersion, "2024-11-05")
	}
}

// Benchmark for DiscoverTools
func BenchmarkDiscoverTools(b *testing.B) {
	mockServer := createMockMCPServer(&testing.T{}, mockServerConfig{
		tools: []Tool{
			{Name: "tool1", Description: "Test tool 1"},
			{Name: "tool2", Description: "Test tool 2"},
		},
	})
	defer os.Remove(mockServer)

	client := NewClient(10*time.Second, NewCache(DefaultCacheTTL))
	ctx := context.Background()

	servers := []ServerConfig{
		{Name: "test", Scope: "user", Command: mockServer, Args: []string{}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.DiscoverTools(ctx, servers, false)
	}
}
