package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// MCPProfile represents an AWS profile configured for MCP access
type MCPProfile struct {
	Name     string // Original profile name (e.g., "dev-mcpro")
	BaseName string // Base name without suffix (e.g., "dev")
	Mode     string // "ro" or "rw"
	Region   string // Detected region
	IsSSO    bool   // True if profile uses SSO authentication
}

// ProfileManager handles AWS profile discovery and parsing
type ProfileManager struct {
	configPath      string
	credentialsPath string
}

// NewProfileManager creates a ProfileManager with default AWS paths
func NewProfileManager() *ProfileManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}

	return &ProfileManager{
		configPath:      filepath.Join(homeDir, ".aws", "config"),
		credentialsPath: filepath.Join(homeDir, ".aws", "credentials"),
	}
}

// NewProfileManagerWithPaths creates a ProfileManager with custom paths (for testing)
func NewProfileManagerWithPaths(configPath, credentialsPath string) *ProfileManager {
	return &ProfileManager{
		configPath:      configPath,
		credentialsPath: credentialsPath,
	}
}

// ListProfiles returns all AWS profile names from config and credentials files
func (pm *ProfileManager) ListProfiles() ([]string, error) {
	profileMap := make(map[string]bool)

	// Parse config file
	if _, err := os.Stat(pm.configPath); err == nil {
		cfg, err := ini.Load(pm.configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		for _, section := range cfg.Sections() {
			name := section.Name()
			if name == "DEFAULT" {
				profileMap["default"] = true
			} else if strings.HasPrefix(name, "profile ") {
				// AWS config uses "[profile name]" format
				profileName := strings.TrimPrefix(name, "profile ")
				profileMap[profileName] = true
			}
		}
	}

	// Parse credentials file
	if _, err := os.Stat(pm.credentialsPath); err == nil {
		creds, err := ini.Load(pm.credentialsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse credentials file: %w", err)
		}

		for _, section := range creds.Sections() {
			name := section.Name()
			if name != "DEFAULT" {
				profileMap[name] = true
			}
		}
	}

	// Convert map to slice
	profiles := make([]string, 0, len(profileMap))
	for profile := range profileMap {
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// ListMCPProfiles returns only profiles with -mcpro or -mcprw suffix
func (pm *ProfileManager) ListMCPProfiles() ([]MCPProfile, error) {
	allProfiles, err := pm.ListProfiles()
	if err != nil {
		return nil, err
	}

	mcpProfiles := make([]MCPProfile, 0)

	for _, profileName := range allProfiles {
		var mode, baseName string

		if strings.HasSuffix(profileName, "-mcpro") {
			mode = "ro"
			baseName = strings.TrimSuffix(profileName, "-mcpro")
		} else if strings.HasSuffix(profileName, "-mcprw") {
			mode = "rw"
			baseName = strings.TrimSuffix(profileName, "-mcprw")
		} else {
			continue // Not an MCP profile
		}

		region, err := pm.GetRegion(profileName)
		if err != nil {
			// Default to us-east-1 if region detection fails
			region = "us-east-1"
		}

		isSSO := pm.IsSSO(profileName)

		mcpProfiles = append(mcpProfiles, MCPProfile{
			Name:     profileName,
			BaseName: baseName,
			Mode:     mode,
			Region:   region,
			IsSSO:    isSSO,
		})
	}

	return mcpProfiles, nil
}

// GetRegion returns the region for a given profile
// Priority: profile region > AWS_DEFAULT_REGION env > AWS_REGION env > "us-east-1"
func (pm *ProfileManager) GetRegion(profileName string) (string, error) {
	// Try to get region from config file
	if _, err := os.Stat(pm.configPath); err == nil {
		cfg, err := ini.Load(pm.configPath)
		if err != nil {
			return "", fmt.Errorf("failed to parse config file: %w", err)
		}

		var section *ini.Section
		if profileName == "default" {
			section = cfg.Section("default")
		} else {
			section = cfg.Section(fmt.Sprintf("profile %s", profileName))
		}

		if section != nil {
			// Use 'region' key, NOT 'sso_region'
			if key := section.Key("region"); key != nil && key.String() != "" {
				return key.String(), nil
			}
		}
	}

	// Fallback to environment variables
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region, nil
	}

	if region := os.Getenv("AWS_REGION"); region != "" {
		return region, nil
	}

	// Final fallback
	return "us-east-1", nil
}

// IsSSO checks if a profile uses SSO authentication
// A profile is SSO-based if it contains sso_start_url, sso_session, or sso_account_id
func (pm *ProfileManager) IsSSO(profileName string) bool {
	if _, err := os.Stat(pm.configPath); err != nil {
		return false
	}

	cfg, err := ini.Load(pm.configPath)
	if err != nil {
		return false
	}

	var section *ini.Section
	if profileName == "default" {
		section = cfg.Section("default")
	} else {
		section = cfg.Section(fmt.Sprintf("profile %s", profileName))
	}

	if section == nil {
		return false
	}

	// Check for SSO indicators
	ssoIndicators := []string{"sso_start_url", "sso_session", "sso_account_id"}
	for _, indicator := range ssoIndicators {
		if key := section.Key(indicator); key != nil && key.String() != "" {
			return true
		}
	}

	return false
}
