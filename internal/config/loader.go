package config

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/viper"
)

// Load loads the configuration using Viper.
// It checks MCP_HELPER_CONFIG environment variable for config path override,
// then searches in current directory (.mcp-helper.yaml) and XDG config directory (config.yaml).
// Returns nil config (not error) if no config file is found.
func Load() (*Config, error) {
	v := viper.New()

	// Check for config path override via environment variable
	if configPath := os.Getenv("MCP_HELPER_CONFIG"); configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Get XDG config directory
		configDir, err := GetConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to determine config directory: %w", err)
		}

		// Search in current directory first (.mcp-helper.yaml), then XDG config (config.yaml)
		v.AddConfigPath(".")
		v.SetConfigName(".mcp-helper")
		v.SetConfigType("yaml")

		v.AddConfigPath(configDir)
		v.SetConfigName("config")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// No config file found - return nil config, not an error
			return nil, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// LoadFromPath loads configuration from a specific file path.
func LoadFromPath(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// GetServer retrieves a server configuration by name.
// Returns an error if the server is not found.
func (c *Config) GetServer(name string) (*Server, error) {
	if c.Servers == nil {
		return nil, fmt.Errorf("no servers configured")
	}

	server, exists := c.Servers[name]
	if !exists {
		return nil, fmt.Errorf("server %q not found", name)
	}

	return server, nil
}

// ListServers returns a sorted list of all server names in the configuration.
func (c *Config) ListServers() []string {
	if c.Servers == nil {
		return []string{}
	}

	names := make([]string, 0, len(c.Servers))
	for name := range c.Servers {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetPreset retrieves a preset configuration by name.
// Returns an error if the preset is not found.
func (c *Config) GetPreset(name string) (*Preset, error) {
	if c.Presets == nil {
		return nil, fmt.Errorf("no presets configured")
	}

	preset, exists := c.Presets[name]
	if !exists {
		return nil, fmt.Errorf("preset %q not found", name)
	}

	return preset, nil
}

// ListPresets returns a sorted list of all preset names in the configuration.
func (c *Config) ListPresets() []string {
	if c.Presets == nil {
		return []string{}
	}

	names := make([]string, 0, len(c.Presets))
	for name := range c.Presets {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// ExpandPreset returns the list of server names for a preset.
// It validates that all referenced servers exist in the configuration.
func (c *Config) ExpandPreset(name string) ([]string, error) {
	preset, err := c.GetPreset(name)
	if err != nil {
		return nil, err
	}

	// Validate all servers exist
	for _, serverName := range preset.Servers {
		if _, exists := c.Servers[serverName]; !exists {
			return nil, fmt.Errorf("preset %q references unknown server %q", name, serverName)
		}
	}

	return preset.Servers, nil
}

// Validate checks that the configuration is valid.
// It ensures transport values are "stdio" or "http" and required fields are present.
func (c *Config) Validate() error {
	if c.Servers == nil {
		return nil // Empty config is valid
	}

	for name, server := range c.Servers {
		if server == nil {
			return fmt.Errorf("server %q is nil", name)
		}

		// Validate transport
		if server.Transport != "stdio" && server.Transport != "http" {
			return fmt.Errorf("server %q has invalid transport %q (must be 'stdio' or 'http')", name, server.Transport)
		}

		// Validate stdio transport requirements
		if server.Transport == "stdio" && server.Command == "" {
			return fmt.Errorf("server %q uses stdio transport but has no command specified", name)
		}

		// Validate http transport requirements
		if server.Transport == "http" && server.URL == "" {
			return fmt.Errorf("server %q uses http transport but has no URL specified", name)
		}
	}

	// Validate presets
	for name, preset := range c.Presets {
		if preset == nil {
			return fmt.Errorf("preset %q is nil", name)
		}

		if len(preset.Servers) == 0 {
			return fmt.Errorf("preset %q has no servers", name)
		}

		// Verify all referenced servers exist
		for _, serverName := range preset.Servers {
			if _, exists := c.Servers[serverName]; !exists {
				return fmt.Errorf("preset %q references unknown server %q", name, serverName)
			}
		}
	}

	return nil
}
