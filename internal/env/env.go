// Package env provides utilities for managing environment variables
// for MCP server configurations. It handles validation, collection,
// and interactive prompting for required environment variables.
package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/stephenwilliams/mcp-helper/internal/config"
	"golang.org/x/term"
)

// IsSecret returns true if the environment variable name suggests
// it contains sensitive information. It checks for common patterns
// like TOKEN, KEY, SECRET, PASSWORD, and CREDENTIAL (case-insensitive).
func IsSecret(name string) bool {
	upperName := strings.ToUpper(name)
	secretPatterns := []string{
		"TOKEN",
		"KEY",
		"SECRET",
		"PASSWORD",
		"CREDENTIAL",
	}

	for _, pattern := range secretPatterns {
		if strings.Contains(upperName, pattern) {
			return true
		}
	}
	return false
}

// ValidateMissing checks which required environment variables are missing
// from the provided map. It returns a slice of environment variable names
// that are marked as required in the server configuration but not present
// in the provided map.
func ValidateMissing(server *config.Server, provided map[string]string) []string {
	var missing []string

	for name, envVar := range server.Env {
		if envVar.Required {
			if _, exists := provided[name]; !exists {
				missing = append(missing, name)
			}
		}
	}

	return missing
}

// CollectEnvVars collects all environment variables needed for the server.
// It follows this priority order:
//  1. Values from the provided map
//  2. Values from the current environment (os.Getenv)
//  3. Default values from the server configuration
//  4. If interactive mode is enabled and required vars are still missing,
//     prompts the user for input
//
// Returns an error if non-interactive mode is used and required variables
// are missing after checking all other sources.
func CollectEnvVars(server *config.Server, provided map[string]string, interactive bool) (map[string]string, error) {
	result := make(map[string]string)

	// Collect all env vars with priority: provided > os.Getenv > default
	for name, envVar := range server.Env {
		var value string
		var found bool

		// Priority 1: Provided values
		if v, exists := provided[name]; exists && v != "" {
			value = v
			found = true
		}

		// Priority 2: Current environment
		if !found {
			if v := os.Getenv(name); v != "" {
				value = v
				found = true
			}
		}

		// Priority 3: Default from config
		if !found && envVar.Default != "" {
			value = envVar.Default
			found = true
		}

		if found {
			result[name] = value
		}
	}

	// Check for missing required vars
	missing := ValidateMissing(server, result)

	// If we have missing required vars, handle based on mode
	if len(missing) > 0 {
		if !interactive {
			return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
		}

		// Interactive mode: prompt for each missing var
		for _, name := range missing {
			envVar := server.Env[name]
			value, err := Prompt(name, envVar.Description)
			if err != nil {
				return nil, fmt.Errorf("failed to prompt for %s: %w", name, err)
			}
			if value == "" && envVar.Required {
				return nil, fmt.Errorf("required environment variable %s cannot be empty", name)
			}
			result[name] = value
		}
	}

	return result, nil
}

// Prompt interactively prompts the user for an environment variable value.
// If the variable name matches secret patterns (via IsSecret), the input
// is hidden using terminal password input. Otherwise, normal input is used.
//
// The description parameter is displayed to help the user understand what
// value is needed.
func Prompt(name, description string) (string, error) {
	fmt.Fprintf(os.Stderr, "Enter value for %s", name)
	if description != "" {
		fmt.Fprintf(os.Stderr, " (%s)", description)
	}
	fmt.Fprintf(os.Stderr, ": ")

	if IsSecret(name) {
		// Use password input (no echo)
		byteValue, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // Print newline after hidden input
		if err != nil {
			return "", fmt.Errorf("failed to read password input: %w", err)
		}
		return strings.TrimSpace(string(byteValue)), nil
	}

	// Use normal input with echo
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(value), nil
}
