package permissions

import (
	"fmt"
	"strings"
)

// PermissionRule represents a single permission rule
type PermissionRule string

// FormatMCPToolRule formats a specific tool as a permission rule
// Format: mcp__servername__toolname
func FormatMCPToolRule(serverName, toolName string) PermissionRule {
	return PermissionRule(fmt.Sprintf("mcp__%s__%s", serverName, toolName))
}

// FormatMCPWildcardRule formats a server wildcard rule
// Format: mcp__servername__*
func FormatMCPWildcardRule(serverName string) PermissionRule {
	return PermissionRule(fmt.Sprintf("mcp__%s__*", serverName))
}

// ParseMCPRule extracts server and tool from an MCP rule
// Returns serverName, toolName (or "*" for wildcard), ok
func ParseMCPRule(rule PermissionRule) (server, tool string, ok bool) {
	s := string(rule)
	if !strings.HasPrefix(s, "mcp__") {
		return "", "", false
	}

	parts := strings.Split(s, "__")
	if len(parts) != 3 {
		return "", "", false
	}

	return parts[1], parts[2], true
}

// IsMCPRule checks if a rule is an MCP permission rule
func IsMCPRule(rule string) bool {
	return strings.HasPrefix(rule, "mcp__") && strings.Count(rule, "__") == 2
}

// IsCoveredByWildcard checks if a specific tool rule is covered by a wildcard rule
func IsCoveredByWildcard(toolRule PermissionRule, rules []PermissionRule) bool {
	server, _, ok := ParseMCPRule(toolRule)
	if !ok {
		return false
	}

	wildcardRule := FormatMCPWildcardRule(server)
	for _, rule := range rules {
		if rule == wildcardRule {
			return true
		}
	}
	return false
}

// MergeRules merges new rules into existing rules, avoiding duplicates
// and skipping rules covered by wildcards
func MergeRules(existing, newRules []PermissionRule) []PermissionRule {
	// Build a set of existing rules
	ruleSet := make(map[PermissionRule]bool)
	for _, rule := range existing {
		ruleSet[rule] = true
	}

	// Add new rules if they don't exist and aren't covered by wildcards
	for _, rule := range newRules {
		if ruleSet[rule] {
			continue // Already exists
		}
		if IsCoveredByWildcard(rule, existing) {
			continue // Covered by wildcard
		}
		existing = append(existing, rule)
		ruleSet[rule] = true
	}

	return existing
}
