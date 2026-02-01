package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAwsDiscoverCommand_Exists(t *testing.T) {
	// Find aws command in root
	var foundAws *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "aws" {
			foundAws = cmd
			break
		}
	}

	if foundAws == nil {
		t.Fatal("aws command not found in root")
	}

	// Find discover subcommand
	var foundDiscover *cobra.Command
	for _, cmd := range foundAws.Commands() {
		if cmd.Use == "discover" {
			foundDiscover = cmd
			break
		}
	}

	if foundDiscover == nil {
		t.Fatal("discover subcommand not found in aws")
	}
}

func TestAwsDiscoverCommand_Flags(t *testing.T) {
	// Find discover command
	var discoverCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "aws" {
			for _, sub := range cmd.Commands() {
				if sub.Use == "discover" {
					discoverCmd = sub
					break
				}
			}
			break
		}
	}

	if discoverCmd == nil {
		t.Fatal("discover command not found")
	}

	// Check flags
	flags := []string{"all", "scope", "json", "dry-run", "force"}
	for _, flag := range flags {
		if discoverCmd.Flags().Lookup(flag) == nil {
			t.Errorf("flag --%s not found", flag)
		}
	}
}

func TestAwsDiscoverCommand_Help(t *testing.T) {
	// Find discover command
	var discoverCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "aws" {
			for _, sub := range cmd.Commands() {
				if sub.Use == "discover" {
					discoverCmd = sub
					break
				}
			}
			break
		}
	}

	if discoverCmd == nil {
		t.Fatal("discover command not found")
	}

	// Verify short description exists
	if discoverCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	// Verify long description exists
	if discoverCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}
