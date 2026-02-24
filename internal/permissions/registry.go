package permissions

import (
	"fmt"
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	adapters = make(map[string]func() Adapter)
)

// Register adds an adapter factory to the registry
// Uses factory pattern for fresh instances and lazy initialization
// (Matches existing pattern in internal/adapter/registry.go)
func Register(name string, factory func() Adapter) {
	mu.Lock()
	defer mu.Unlock()
	adapters[strings.ToLower(name)] = factory
}

// Get retrieves and instantiates an adapter by name (case-insensitive)
func Get(name string) (Adapter, error) {
	mu.RLock()
	defer mu.RUnlock()
	factory, ok := adapters[strings.ToLower(name)]
	if !ok {
		available := List()
		return nil, fmt.Errorf("unknown permissions adapter: %s (available: %s)", name, strings.Join(available, ", "))
	}
	return factory(), nil
}

// GetWithDefault returns adapter by name, falling back to default
func GetWithDefault(name, defaultName string) (Adapter, error) {
	if name == "" {
		name = defaultName
	}
	return Get(name)
}

// List returns all registered adapter names
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(adapters))
	for name := range adapters {
		names = append(names, name)
	}
	return names
}
