// Package app provides application-level services and use cases for mcp-helper.
// It defines the business logic layer that orchestrates interactions between
// the presentation layer (cmd, tui) and the infrastructure layer (adapter).
package app

import "errors"

// Sentinel errors for the application layer.
// These provide consistent error handling across the codebase.
var (
	// ErrServerNotFound indicates the requested server does not exist in the configuration.
	ErrServerNotFound = errors.New("server not found")

	// ErrAlreadyInstalled indicates the server is already installed in the target scope.
	ErrAlreadyInstalled = errors.New("server already installed")

	// ErrInvalidScope indicates an invalid or unsupported scope was specified.
	ErrInvalidScope = errors.New("invalid scope")

	// ErrMissingRequiredEnv indicates a required environment variable was not provided.
	ErrMissingRequiredEnv = errors.New("missing required environment variable")

	// ErrInstallFailed indicates the server installation failed.
	ErrInstallFailed = errors.New("installation failed")
)
