// Package config provides configuration types and utilities for mcp-helper.
// It defines the structure for MCP server configurations including transport
// settings, environment variables, and server metadata.
package config

// Config represents the mcp-helper configuration.
// It contains the default scope for operations and a map of server configurations.
type Config struct {
	DefaultScope string             `yaml:"default_scope" mapstructure:"default_scope"`
	DefaultAgent string             `yaml:"default_agent" mapstructure:"default_agent"`
	Servers      map[string]*Server `yaml:"servers" mapstructure:"servers"`
	Presets      map[string]*Preset `yaml:"presets" mapstructure:"presets"`
}

// Server represents an MCP server configuration.
// It supports both stdio and HTTP transport protocols and includes
// environment variable configurations.
type Server struct {
	Description string            `yaml:"description" mapstructure:"description"`
	Transport   string            `yaml:"transport" mapstructure:"transport"` // "stdio" or "http"
	Command     string            `yaml:"command" mapstructure:"command"`     // For stdio
	Args        []string          `yaml:"args" mapstructure:"args"`           // For stdio
	URL         string            `yaml:"url" mapstructure:"url"`             // For http
	Headers     map[string]string `yaml:"headers" mapstructure:"headers"`     // For http
	Env         map[string]EnvVar `yaml:"env" mapstructure:"env"`
}

// EnvVar represents an environment variable configuration.
// It includes metadata about whether the variable is required,
// its purpose, and an optional default value.
type EnvVar struct {
	Required    bool   `yaml:"required" mapstructure:"required"`
	Description string `yaml:"description" mapstructure:"description"`
	Default     string `yaml:"default" mapstructure:"default"`
}

// Preset represents a named collection of servers that can be installed together.
type Preset struct {
	Description string   `yaml:"description" mapstructure:"description"`
	Servers     []string `yaml:"servers" mapstructure:"servers"`
}
