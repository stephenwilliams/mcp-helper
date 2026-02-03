package aws

import (
	"fmt"

	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// GenerateServerName generates a server name from an MCP profile.
// Format: "aws-{BaseName}-{Mode}"
func GenerateServerName(profile MCPProfile) string {
	return fmt.Sprintf("aws-%s-%s", profile.BaseName, profile.Mode)
}

// GenerateServer creates a complete Server configuration from an MCP profile.
func GenerateServer(profile MCPProfile) *config.Server {
	// Determine mode description based on profile mode
	var modeDesc string

	switch profile.Mode {
	case "ro":
		modeDesc = "read-only"
	case "rw":
		modeDesc = "read-write"
	default:
		modeDesc = "read-only"
	}

	// Build description
	description := fmt.Sprintf("AWS MCP (%s) - %s", modeDesc, profile.BaseName)

	// Build args array
	args := []string{
		"mcp-proxy-for-aws@latest",
		"--profile",
		profile.Name,
		"--region",
		profile.Region,
	}

	// Add --read-only flag for read-only mode (no value, just the flag)
	if profile.Mode == "ro" {
		args = append(args, "--read-only")
	}

	// URL must be last
	args = append(args, "https://aws-mcp.us-east-1.api.aws/mcp")

	// Build env map
	env := map[string]config.EnvVar{
		"AWS_PROFILE": {
			Required:    true,
			Description: "AWS profile to use",
			Default:     profile.Name,
		},
	}

	return &config.Server{
		Description: description,
		Transport:   "stdio",
		Command:     "uvx",
		Args:        args,
		Env:         env,
	}
}
