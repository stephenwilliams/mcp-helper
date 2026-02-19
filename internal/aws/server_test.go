package aws

import (
	"strings"
	"testing"
)

func TestGenerateServerName(t *testing.T) {
	tests := []struct {
		name     string
		profile  MCPProfile
		expected string
	}{
		{
			name:     "read-only profile",
			profile:  MCPProfile{BaseName: "dev", Mode: "ro"},
			expected: "aws-dev-ro",
		},
		{
			name:     "read-write profile",
			profile:  MCPProfile{BaseName: "prod", Mode: "rw"},
			expected: "aws-prod-rw",
		},
		{
			name:     "multi-word baseName",
			profile:  MCPProfile{BaseName: "my-app", Mode: "ro"},
			expected: "aws-my-app-ro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateServerName(tt.profile)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateServer_ReadOnly(t *testing.T) {
	profile := MCPProfile{
		Name:     "dev-mcpro",
		BaseName: "dev",
		Mode:     "ro",
		Region:   "us-east-1",
		IsSSO:    true,
	}

	server := GenerateServer(profile)

	if server.Description != "AWS MCP (read-only) - dev" {
		t.Errorf("unexpected description: %s", server.Description)
	}

	if server.Transport != "stdio" {
		t.Errorf("unexpected transport: %s", server.Transport)
	}

	if server.Command != "uvx" {
		t.Errorf("unexpected command: %s", server.Command)
	}

	// Check args contain expected values
	argsStr := strings.Join(server.Args, " ")
	if !strings.Contains(argsStr, "mcp-proxy-for-aws@latest") {
		t.Error("args should contain mcp-proxy-for-aws@latest")
	}

	// Verify correct flags are present
	if !strings.Contains(argsStr, "--profile dev-mcpro") {
		t.Error("args should contain --profile with profile name")
	}
	if !strings.Contains(argsStr, "--region us-east-1") {
		t.Error("args should contain --region with region value")
	}
	// Verify invalid flags are NOT present
	if strings.Contains(argsStr, "--permission") {
		t.Error("args should NOT contain --permission flag")
	}
	if strings.Contains(argsStr, "--metadata") {
		t.Error("args should NOT contain --metadata flag")
	}

	if !strings.Contains(argsStr, "https://aws-mcp.us-east-1.api.aws/mcp") {
		t.Error("args should contain AWS MCP endpoint URL")
	}

	// Check env
	envVar, ok := server.Env["AWS_PROFILE"]
	if !ok {
		t.Fatal("AWS_PROFILE env var not found")
	}
	if envVar.Default != "dev-mcpro" {
		t.Errorf("unexpected default: %s", envVar.Default)
	}
	if !envVar.Required {
		t.Error("AWS_PROFILE should be required")
	}
}

func TestGenerateServer_ReadWrite(t *testing.T) {
	profile := MCPProfile{
		Name:     "prod-mcprw",
		BaseName: "prod",
		Mode:     "rw",
		Region:   "eu-west-1",
		IsSSO:    false,
	}

	server := GenerateServer(profile)

	if server.Description != "AWS MCP (read-write) - prod" {
		t.Errorf("unexpected description: %s", server.Description)
	}

	// Check args contain expected values
	argsStr := strings.Join(server.Args, " ")

	// Verify correct flags are present
	if !strings.Contains(argsStr, "--profile prod-mcprw") {
		t.Error("args should contain --profile with profile name")
	}
	if !strings.Contains(argsStr, "--region eu-west-1") {
		t.Error("args should contain --region with region value")
	}

	// Verify invalid flags are NOT present
	if strings.Contains(argsStr, "--permission") {
		t.Error("args should NOT contain --permission flag")
	}
	if strings.Contains(argsStr, "--metadata") {
		t.Error("args should NOT contain --metadata flag")
	}

	// Check env
	envVar, ok := server.Env["AWS_PROFILE"]
	if !ok {
		t.Fatal("AWS_PROFILE env var not found")
	}
	if envVar.Default != "prod-mcprw" {
		t.Errorf("unexpected default: %s", envVar.Default)
	}
}

func TestGenerateServer_DifferentRegions(t *testing.T) {
	regions := []string{"us-east-1", "eu-west-1", "ap-southeast-1"}

	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			profile := MCPProfile{
				Name:     "test-profile",
				BaseName: "test",
				Mode:     "ro",
				Region:   region,
				IsSSO:    true,
			}

			server := GenerateServer(profile)
			argsStr := strings.Join(server.Args, " ")

			expectedRegion := "--region " + region
			if !strings.Contains(argsStr, expectedRegion) {
				t.Errorf("args should contain %s", expectedRegion)
			}
		})
	}
}
