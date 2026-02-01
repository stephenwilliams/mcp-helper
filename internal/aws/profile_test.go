package aws

import (
	"sort"
	"testing"
)

func TestListProfiles(t *testing.T) {
	pm := NewProfileManagerWithPaths(
		"../../testdata/aws/config",
		"../../testdata/aws/credentials",
	)

	profiles, err := pm.ListProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find all profiles: default, dev-mcpro, dev-mcprw, prod-mcpro, regular-profile, no-region-mcpro
	expectedProfiles := []string{"default", "dev-mcpro", "dev-mcprw", "prod-mcpro", "regular-profile", "no-region-mcpro"}
	if len(profiles) != len(expectedProfiles) {
		t.Errorf("expected %d profiles, got %d", len(expectedProfiles), len(profiles))
	}

	// Convert to map for easier checking
	profileMap := make(map[string]bool)
	for _, p := range profiles {
		profileMap[p] = true
	}

	for _, expected := range expectedProfiles {
		if !profileMap[expected] {
			t.Errorf("expected profile '%s' not found", expected)
		}
	}
}

func TestListMCPProfiles(t *testing.T) {
	pm := NewProfileManagerWithPaths(
		"../../testdata/aws/config",
		"../../testdata/aws/credentials",
	)

	profiles, err := pm.ListMCPProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 4 MCP profiles: dev-mcpro, dev-mcprw, prod-mcpro, no-region-mcpro
	if len(profiles) != 4 {
		t.Errorf("expected 4 MCP profiles, got %d", len(profiles))
	}

	// Verify that regular-profile and default are NOT included
	for _, p := range profiles {
		if p.Name == "regular-profile" || p.Name == "default" {
			t.Errorf("non-MCP profile '%s' should not be included", p.Name)
		}
	}

	// Create map for easier lookup
	profileMap := make(map[string]MCPProfile)
	for _, p := range profiles {
		profileMap[p.Name] = p
	}

	// Verify dev-mcpro
	if p, ok := profileMap["dev-mcpro"]; ok {
		if p.Mode != "ro" {
			t.Errorf("dev-mcpro: expected mode 'ro', got '%s'", p.Mode)
		}
		if p.BaseName != "dev" {
			t.Errorf("dev-mcpro: expected baseName 'dev', got '%s'", p.BaseName)
		}
		if p.Region != "us-east-1" {
			t.Errorf("dev-mcpro: expected region 'us-east-1', got '%s'", p.Region)
		}
	} else {
		t.Error("dev-mcpro profile not found")
	}

	// Verify dev-mcprw
	if p, ok := profileMap["dev-mcprw"]; ok {
		if p.Mode != "rw" {
			t.Errorf("dev-mcprw: expected mode 'rw', got '%s'", p.Mode)
		}
		if p.BaseName != "dev" {
			t.Errorf("dev-mcprw: expected baseName 'dev', got '%s'", p.BaseName)
		}
		if p.Region != "us-east-1" {
			t.Errorf("dev-mcprw: expected region 'us-east-1', got '%s'", p.Region)
		}
	} else {
		t.Error("dev-mcprw profile not found")
	}

	// Verify prod-mcpro
	if p, ok := profileMap["prod-mcpro"]; ok {
		if p.Mode != "ro" {
			t.Errorf("prod-mcpro: expected mode 'ro', got '%s'", p.Mode)
		}
		if p.BaseName != "prod" {
			t.Errorf("prod-mcpro: expected baseName 'prod', got '%s'", p.BaseName)
		}
		if p.Region != "eu-west-1" {
			t.Errorf("prod-mcpro: expected region 'eu-west-1', got '%s'", p.Region)
		}
	} else {
		t.Error("prod-mcpro profile not found")
	}

	// Verify no-region-mcpro
	if p, ok := profileMap["no-region-mcpro"]; ok {
		if p.Mode != "ro" {
			t.Errorf("no-region-mcpro: expected mode 'ro', got '%s'", p.Mode)
		}
		if p.BaseName != "no-region" {
			t.Errorf("no-region-mcpro: expected baseName 'no-region', got '%s'", p.BaseName)
		}
		// Should default to us-east-1
		if p.Region != "us-east-1" {
			t.Errorf("no-region-mcpro: expected default region 'us-east-1', got '%s'", p.Region)
		}
	} else {
		t.Error("no-region-mcpro profile not found")
	}
}

func TestGetRegion(t *testing.T) {
	pm := NewProfileManagerWithPaths(
		"../../testdata/aws/config",
		"../../testdata/aws/credentials",
	)

	tests := []struct {
		name           string
		profileName    string
		expectedRegion string
	}{
		{
			name:           "profile with region in config",
			profileName:    "dev-mcpro",
			expectedRegion: "us-east-1",
		},
		{
			name:           "profile without region falls back to default",
			profileName:    "no-region-mcpro",
			expectedRegion: "us-east-1",
		},
		{
			name:           "default profile region detection",
			profileName:    "default",
			expectedRegion: "us-west-2",
		},
		{
			name:           "prod profile with different region",
			profileName:    "prod-mcpro",
			expectedRegion: "eu-west-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, err := pm.GetRegion(tt.profileName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if region != tt.expectedRegion {
				t.Errorf("expected region '%s', got '%s'", tt.expectedRegion, region)
			}
		})
	}
}

func TestIsSSO(t *testing.T) {
	pm := NewProfileManagerWithPaths(
		"../../testdata/aws/config",
		"../../testdata/aws/credentials",
	)

	tests := []struct {
		name        string
		profileName string
		expectedSSO bool
	}{
		{
			name:        "dev-mcpro should be SSO",
			profileName: "dev-mcpro",
			expectedSSO: true,
		},
		{
			name:        "dev-mcprw should be SSO",
			profileName: "dev-mcprw",
			expectedSSO: true,
		},
		{
			name:        "prod-mcpro should not be SSO (IAM credentials)",
			profileName: "prod-mcpro",
			expectedSSO: false,
		},
		{
			name:        "default should not be SSO",
			profileName: "default",
			expectedSSO: false,
		},
		{
			name:        "no-region-mcpro should not be SSO",
			profileName: "no-region-mcpro",
			expectedSSO: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSSO := pm.IsSSO(tt.profileName)
			if isSSO != tt.expectedSSO {
				t.Errorf("expected IsSSO=%v, got %v", tt.expectedSSO, isSSO)
			}
		})
	}
}

func TestMCPProfileFields(t *testing.T) {
	pm := NewProfileManagerWithPaths(
		"../../testdata/aws/config",
		"../../testdata/aws/credentials",
	)

	profiles, err := pm.ListMCPProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find dev-mcpro profile
	var devProfile *MCPProfile
	for i := range profiles {
		if profiles[i].Name == "dev-mcpro" {
			devProfile = &profiles[i]
			break
		}
	}

	if devProfile == nil {
		t.Fatal("dev-mcpro profile not found")
	}

	// Verify all fields
	if devProfile.Name != "dev-mcpro" {
		t.Errorf("Name: expected 'dev-mcpro', got '%s'", devProfile.Name)
	}
	if devProfile.BaseName != "dev" {
		t.Errorf("BaseName: expected 'dev', got '%s'", devProfile.BaseName)
	}
	if devProfile.Mode != "ro" {
		t.Errorf("Mode: expected 'ro', got '%s'", devProfile.Mode)
	}
	if devProfile.Region != "us-east-1" {
		t.Errorf("Region: expected 'us-east-1', got '%s'", devProfile.Region)
	}
	if !devProfile.IsSSO {
		t.Error("IsSSO: expected true, got false")
	}
}

func TestEmptyConfig(t *testing.T) {
	pm := NewProfileManagerWithPaths(
		"/nonexistent/path/config",
		"/nonexistent/path/credentials",
	)

	t.Run("ListProfiles with empty config", func(t *testing.T) {
		profiles, err := pm.ListProfiles()
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(profiles) != 0 {
			t.Errorf("expected empty slice, got %d profiles", len(profiles))
		}
	})

	t.Run("ListMCPProfiles with empty config", func(t *testing.T) {
		mcpProfiles, err := pm.ListMCPProfiles()
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(mcpProfiles) != 0 {
			t.Errorf("expected empty slice, got %d MCP profiles", len(mcpProfiles))
		}
	})
}

func TestListProfilesOrder(t *testing.T) {
	pm := NewProfileManagerWithPaths(
		"../../testdata/aws/config",
		"../../testdata/aws/credentials",
	)

	profiles, err := pm.ListProfiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify we can sort profiles (they should be in a predictable order)
	sort.Strings(profiles)

	expected := []string{"default", "dev-mcpro", "dev-mcprw", "no-region-mcpro", "prod-mcpro", "regular-profile"}
	for i, exp := range expected {
		if profiles[i] != exp {
			t.Errorf("index %d: expected '%s', got '%s'", i, exp, profiles[i])
		}
	}
}
