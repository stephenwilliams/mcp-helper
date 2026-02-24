package permissions

import (
	"testing"
)

func TestFormatMCPToolRule(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		toolName   string
		want       PermissionRule
	}{
		{
			name:       "basic tool rule",
			serverName: "github",
			toolName:   "search_repositories",
			want:       "mcp__github__search_repositories",
		},
		{
			name:       "filesystem tool",
			serverName: "filesystem",
			toolName:   "read_file",
			want:       "mcp__filesystem__read_file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMCPToolRule(tt.serverName, tt.toolName)
			if got != tt.want {
				t.Errorf("FormatMCPToolRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatMCPWildcardRule(t *testing.T) {
	tests := []struct {
		name       string
		serverName string
		want       PermissionRule
	}{
		{
			name:       "github wildcard",
			serverName: "github",
			want:       "mcp__github__*",
		},
		{
			name:       "filesystem wildcard",
			serverName: "filesystem",
			want:       "mcp__filesystem__*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMCPWildcardRule(tt.serverName)
			if got != tt.want {
				t.Errorf("FormatMCPWildcardRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMCPRule(t *testing.T) {
	tests := []struct {
		name       string
		rule       PermissionRule
		wantServer string
		wantTool   string
		wantOk     bool
	}{
		{
			name:       "valid tool rule",
			rule:       "mcp__github__search_repositories",
			wantServer: "github",
			wantTool:   "search_repositories",
			wantOk:     true,
		},
		{
			name:       "valid wildcard rule",
			rule:       "mcp__github__*",
			wantServer: "github",
			wantTool:   "*",
			wantOk:     true,
		},
		{
			name:       "invalid rule - not mcp",
			rule:       "Bash(npm run *)",
			wantServer: "",
			wantTool:   "",
			wantOk:     false,
		},
		{
			name:       "invalid rule - wrong format",
			rule:       "mcp__github",
			wantServer: "",
			wantTool:   "",
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotServer, gotTool, gotOk := ParseMCPRule(tt.rule)
			if gotServer != tt.wantServer || gotTool != tt.wantTool || gotOk != tt.wantOk {
				t.Errorf("ParseMCPRule() = (%v, %v, %v), want (%v, %v, %v)",
					gotServer, gotTool, gotOk, tt.wantServer, tt.wantTool, tt.wantOk)
			}
		})
	}
}

func TestIsMCPRule(t *testing.T) {
	tests := []struct {
		name string
		rule string
		want bool
	}{
		{
			name: "valid mcp tool rule",
			rule: "mcp__github__search_repositories",
			want: true,
		},
		{
			name: "valid mcp wildcard rule",
			rule: "mcp__github__*",
			want: true,
		},
		{
			name: "bash rule",
			rule: "Bash(npm run *)",
			want: false,
		},
		{
			name: "empty rule",
			rule: "",
			want: false,
		},
		{
			name: "malformed mcp rule",
			rule: "mcp__github",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMCPRule(tt.rule)
			if got != tt.want {
				t.Errorf("IsMCPRule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCoveredByWildcard(t *testing.T) {
	tests := []struct {
		name     string
		toolRule PermissionRule
		rules    []PermissionRule
		want     bool
	}{
		{
			name:     "covered by wildcard",
			toolRule: "mcp__github__search_repositories",
			rules: []PermissionRule{
				"mcp__github__*",
				"mcp__filesystem__read_file",
			},
			want: true,
		},
		{
			name:     "not covered - no wildcard",
			toolRule: "mcp__github__search_repositories",
			rules: []PermissionRule{
				"mcp__filesystem__*",
				"mcp__github__create_issue",
			},
			want: false,
		},
		{
			name:     "not covered - empty rules",
			toolRule: "mcp__github__search_repositories",
			rules:    []PermissionRule{},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCoveredByWildcard(tt.toolRule, tt.rules)
			if got != tt.want {
				t.Errorf("IsCoveredByWildcard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeRules(t *testing.T) {
	tests := []struct {
		name     string
		existing []PermissionRule
		newRules []PermissionRule
		want     []PermissionRule
	}{
		{
			name: "add new rules",
			existing: []PermissionRule{
				"mcp__github__search_repositories",
			},
			newRules: []PermissionRule{
				"mcp__filesystem__read_file",
				"mcp__filesystem__write_file",
			},
			want: []PermissionRule{
				"mcp__github__search_repositories",
				"mcp__filesystem__read_file",
				"mcp__filesystem__write_file",
			},
		},
		{
			name: "skip duplicates",
			existing: []PermissionRule{
				"mcp__github__search_repositories",
				"mcp__filesystem__read_file",
			},
			newRules: []PermissionRule{
				"mcp__filesystem__read_file",
				"mcp__filesystem__write_file",
			},
			want: []PermissionRule{
				"mcp__github__search_repositories",
				"mcp__filesystem__read_file",
				"mcp__filesystem__write_file",
			},
		},
		{
			name: "skip rules covered by wildcard",
			existing: []PermissionRule{
				"mcp__github__*",
			},
			newRules: []PermissionRule{
				"mcp__github__search_repositories",
				"mcp__github__create_issue",
				"mcp__filesystem__read_file",
			},
			want: []PermissionRule{
				"mcp__github__*",
				"mcp__filesystem__read_file",
			},
		},
		{
			name:     "empty existing rules",
			existing: []PermissionRule{},
			newRules: []PermissionRule{
				"mcp__github__search_repositories",
			},
			want: []PermissionRule{
				"mcp__github__search_repositories",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeRules(tt.existing, tt.newRules)
			if len(got) != len(tt.want) {
				t.Errorf("MergeRules() length = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MergeRules()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
