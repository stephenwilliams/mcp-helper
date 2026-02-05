// Package adapter provides a registry pattern for AI agent adapters.
//
// The registry allows different AI providers (claudecode, claude-ai, etc.) to register
// themselves at initialization time, enabling dynamic adapter selection at runtime.
//
// Usage:
//
//	// Register an adapter (typically in init())
//	adapter.Register("claudecode", func() adapter.Adapter {
//	    return &ClaudeCodeAdapter{}
//	})
//
//	// Get an adapter by name
//	a, err := adapter.Get("claudecode")
//
//	// Get with fallback to default
//	a, err := adapter.GetWithDefault(flagValue, configValue)
package adapter

import (
	"fmt"
	"strings"
)

var registry = map[string]func() Adapter{}

// FallbackDefault is used when no config default is set
const FallbackDefault = "claudecode"

// Register registers an adapter factory with the given name.
// Adapters should call this in their init() function.
func Register(name string, factory func() Adapter) {
	registry[strings.ToLower(name)] = factory
}

// Get returns an adapter by name (case-insensitive).
// Returns an error if the adapter is not found.
func Get(name string) (Adapter, error) {
	factory, ok := registry[strings.ToLower(name)]
	if !ok {
		available := List()
		return nil, fmt.Errorf("unknown agent %q, available: %s", name, strings.Join(available, ", "))
	}
	return factory(), nil
}

// List returns all registered adapter names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// GetWithDefault returns the adapter for the given name, or the default if name is empty.
// Priority: flagName > configDefault > FallbackDefault
func GetWithDefault(flagName string, configDefault string) (Adapter, error) {
	name := flagName
	if name == "" {
		name = configDefault
	}
	if name == "" {
		name = FallbackDefault
	}
	return Get(name)
}
