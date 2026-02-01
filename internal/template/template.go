// Package template provides Go template processing for MCP server configurations.
// It supports slim-sprig functions and custom functions like onePasswordRead.
package template

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	sprig "github.com/go-task/slim-sprig/v3"
)

// templatePattern matches Go template syntax {{ }}.
// Uses non-greedy matching and requires at least one character between braces.
var templatePattern = regexp.MustCompile(`\{\{[^{}]+\}\}`)

// HasTemplateSyntax returns true if the string contains valid Go template syntax.
// Note: This detects {{ content }} patterns but not malformed patterns like {{{{.
func HasTemplateSyntax(s string) bool {
	return templatePattern.MatchString(s)
}

// ProcessString processes a string as a Go template if it contains template syntax.
// Returns the original string unchanged if no template syntax is detected.
func ProcessString(value string, data interface{}) (string, error) {
	if !HasTemplateSyntax(value) {
		return value, nil // Fast path: no template syntax
	}
	return executeTemplate(value, data)
}

// executeTemplate executes a Go template string with slim-sprig functions
// and custom functions.
func executeTemplate(templateStr string, data interface{}) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["onePasswordRead"] = onePasswordRead

	tpl, err := template.New("config").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// onePasswordRead retrieves a secret from 1Password using the op CLI.
// Arguments:
//   - secretRef: The 1Password secret reference (e.g., "op://vault/item/field")
//   - account: Optional account identifier for multi-account setups
func onePasswordRead(secretRef string, account ...string) (string, error) {
	if secretRef == "" {
		return "", fmt.Errorf("onePasswordRead: secret reference cannot be empty")
	}

	args := []string{"read", "-n"} // -n for no trailing newline
	if len(account) > 0 && account[0] != "" {
		args = append(args, "--account", account[0])
	}
	args = append(args, secretRef)

	cmd := exec.Command("op", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("onePasswordRead: op command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("onePasswordRead: failed to execute op command: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
