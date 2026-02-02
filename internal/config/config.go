package config

import (
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const (
	// AppName is the application name used for XDG paths
	AppName = "mcp-helper"
	// DefaultConfigFileName is the default config file name
	DefaultConfigFileName = "config.yaml"
)

// configDirOverride allows tests to override the config directory.
// When empty, the XDG config directory is used.
var configDirOverride string

// SetConfigDirOverride sets an override for the config directory (for testing).
// Pass empty string to clear the override and use the default XDG directory.
func SetConfigDirOverride(dir string) {
	configDirOverride = dir
}

// GetConfigDir returns the XDG-compliant config directory for mcp-helper
func GetConfigDir() (string, error) {
	if configDirOverride != "" {
		return configDirOverride, nil
	}
	return filepath.Join(xdg.ConfigHome, AppName), nil
}

// GetConfigFilePath returns the full path to the config file
func GetConfigFilePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, DefaultConfigFileName), nil
}

// EnsureConfigDir creates the config directory if it doesn't exist
// This should be called lazily, only when actually needed
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(configDir, 0755)
}

// GetDataDir returns the XDG-compliant data directory for mcp-helper
func GetDataDir() (string, error) {
	return filepath.Join(xdg.DataHome, AppName), nil
}

// EnsureDataDir creates the data directory if it doesn't exist
func EnsureDataDir() error {
	dataDir, err := GetDataDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dataDir, 0755)
}

// GetCacheDir returns the XDG-compliant cache directory for mcp-helper
func GetCacheDir() (string, error) {
	return filepath.Join(xdg.CacheHome, AppName), nil
}
