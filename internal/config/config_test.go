package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	dir, err := GetConfigDir()
	if err != nil {
		t.Errorf("GetConfigDir() error = %v", err)
	}
	if dir == "" {
		t.Error("GetConfigDir() returned empty string")
	}
	if !strings.Contains(dir, AppName) {
		t.Errorf("GetConfigDir() = %q, expected to contain %q", dir, AppName)
	}
}

func TestGetConfigFilePath(t *testing.T) {
	path, err := GetConfigFilePath()
	if err != nil {
		t.Errorf("GetConfigFilePath() error = %v", err)
	}
	if path == "" {
		t.Error("GetConfigFilePath() returned empty string")
	}
	if !strings.Contains(path, AppName) {
		t.Errorf("GetConfigFilePath() = %q, expected to contain %q", path, AppName)
	}
	if !strings.Contains(path, DefaultConfigFileName) {
		t.Errorf("GetConfigFilePath() = %q, expected to contain %q", path, DefaultConfigFileName)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	// EnsureConfigDir uses xdg package which reads env at init time
	// So we can't override it in tests. Instead, just test that it doesn't error.
	err := EnsureConfigDir()
	if err != nil {
		t.Errorf("EnsureConfigDir() error = %v", err)
	}

	// Test idempotency - calling again should not error
	err = EnsureConfigDir()
	if err != nil {
		t.Errorf("EnsureConfigDir() second call error = %v", err)
	}

	// Verify directory exists
	configDir, _ := GetConfigDir()
	info, err := os.Stat(configDir)
	if err != nil {
		t.Errorf("Config directory does not exist: %v", err)
		return
	}
	if !info.IsDir() {
		t.Errorf("Config path exists but is not a directory")
	}
}

func TestGetDataDir(t *testing.T) {
	dir, err := GetDataDir()
	if err != nil {
		t.Errorf("GetDataDir() error = %v", err)
	}
	if dir == "" {
		t.Error("GetDataDir() returned empty string")
	}
	if !strings.Contains(dir, AppName) {
		t.Errorf("GetDataDir() = %q, expected to contain %q", dir, AppName)
	}
}

func TestEnsureDataDir(t *testing.T) {
	// EnsureDataDir uses xdg package which reads env at init time
	// So we can't override it in tests. Instead, just test that it doesn't error.
	err := EnsureDataDir()
	if err != nil {
		t.Errorf("EnsureDataDir() error = %v", err)
	}

	// Verify directory exists
	dataDir, _ := GetDataDir()
	info, err := os.Stat(dataDir)
	if err != nil {
		t.Errorf("Data directory does not exist: %v", err)
		return
	}
	if !info.IsDir() {
		t.Errorf("Data path exists but is not a directory")
	}
}

func TestGetCacheDir(t *testing.T) {
	dir, err := GetCacheDir()
	if err != nil {
		t.Errorf("GetCacheDir() error = %v", err)
	}
	if dir == "" {
		t.Error("GetCacheDir() returned empty string")
	}
	if !strings.Contains(dir, AppName) {
		t.Errorf("GetCacheDir() = %q, expected to contain %q", dir, AppName)
	}
}

func TestLoad_WithEnvOverride(t *testing.T) {
	// Save and restore original env
	oldEnv := os.Getenv("MCP_HELPER_CONFIG")
	oldDir, _ := os.Getwd()
	defer func() {
		os.Setenv("MCP_HELPER_CONFIG", oldEnv)
		os.Chdir(oldDir)
	}()

	// Set env to point to test config
	testConfig := "../../testdata/config_test.yaml"
	absPath, _ := filepath.Abs(testConfig)
	os.Setenv("MCP_HELPER_CONFIG", absPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with env override error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.DefaultScope != "user" {
		t.Errorf("DefaultScope = %q, want %q", cfg.DefaultScope, "user")
	}
}
