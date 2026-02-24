// Package mock provides a configurable mock implementation of adapter.Adapter for testing.
package mock

import (
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// AddServerCall records a call to AddServer for test assertions.
type AddServerCall struct {
	Name   string
	Server *config.Server
	Scope  adapter.Scope
	Env    map[string]string
}

// Adapter is a configurable mock implementation of adapter.Adapter.
type Adapter struct {
	ExistingServers map[string]bool
	AddServerErr    error
	AddServerCalls  []AddServerCall
	ConfigPath      string
}

// New creates a new mock adapter with default configuration.
func New() *Adapter {
	return &Adapter{
		ExistingServers: make(map[string]bool),
	}
}

func (m *Adapter) Name() string { return "mock" }

func (m *Adapter) AddServer(name string, server *config.Server, scope adapter.Scope, env map[string]string) error {
	m.AddServerCalls = append(m.AddServerCalls, AddServerCall{name, server, scope, env})
	return m.AddServerErr
}

func (m *Adapter) DryRun(name string, server *config.Server, scope adapter.Scope, env map[string]string) string {
	return ""
}

func (m *Adapter) GetConfigPath(scope adapter.Scope) string {
	return m.ConfigPath
}

func (m *Adapter) ServerExists(name string, scope adapter.Scope) bool {
	return m.ExistingServers[name]
}

// DefaultTestConfig returns a default configuration for testing.
func DefaultTestConfig() *config.Config {
	return &config.Config{
		Servers: map[string]*config.Server{
			"server-alpha": {Description: "Alpha server", Transport: "stdio", Command: "alpha"},
			"server-beta": {
				Description: "Beta server with env vars",
				Transport:   "stdio",
				Command:     "beta",
				Env:         map[string]config.EnvVar{"API_KEY": {Description: "API key", Required: true}},
			},
			"server-gamma": {Description: "Gamma HTTP server", Transport: "http", URL: "http://localhost:8080"},
		},
		Presets: map[string]*config.Preset{
			"basic": {Description: "Basic preset", Servers: []string{"server-alpha", "server-gamma"}},
		},
	}
}
