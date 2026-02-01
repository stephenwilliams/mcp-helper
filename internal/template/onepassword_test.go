package template

import (
	"os/exec"
	"testing"
)

func TestOnePasswordRead_EmptyRef(t *testing.T) {
	_, err := onePasswordRead("")
	if err == nil {
		t.Error("expected error for empty secret reference")
	}
}

func TestOnePasswordRead_Integration(t *testing.T) {
	// Skip if op CLI is not available
	if _, err := exec.LookPath("op"); err != nil {
		t.Skip("1Password CLI (op) not available")
	}

	// This test requires actual 1Password setup - mark as integration test
	t.Skip("Integration test - requires 1Password CLI authentication")
}

func TestOnePasswordRead_WithAccount(t *testing.T) {
	// Verify the function accepts account parameter without error
	if _, err := exec.LookPath("op"); err != nil {
		t.Skip("1Password CLI (op) not available")
	}

	t.Skip("Integration test - requires 1Password CLI authentication")
}
