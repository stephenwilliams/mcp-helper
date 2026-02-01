package aws

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewMultiSelectModel(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "dev-mcpro", BaseName: "dev", Mode: "ro", Region: "us-east-1", IsSSO: true},
		{Name: "prod-mcpro", BaseName: "prod", Mode: "ro", Region: "eu-west-1", IsSSO: false},
	}

	m := NewMultiSelectModel(profiles)

	if m.cursor != 0 {
		t.Errorf("expected initial cursor to be 0, got %d", m.cursor)
	}

	if len(m.selected) != len(profiles) {
		t.Errorf("expected %d selected slots, got %d", len(profiles), len(m.selected))
	}

	for i, sel := range m.selected {
		if sel {
			t.Errorf("profile at index %d should not be selected initially", i)
		}
	}

	if m.confirmed {
		t.Error("confirmed should be false initially")
	}

	if m.cancelled {
		t.Error("cancelled should be false initially")
	}
}

func TestMultiSelectModel_Toggle(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "dev-mcpro", BaseName: "dev", Mode: "ro", Region: "us-east-1", IsSSO: true},
		{Name: "prod-mcpro", BaseName: "prod", Mode: "ro", Region: "eu-west-1", IsSSO: false},
	}

	m := NewMultiSelectModel(profiles)

	// Initially not selected
	if m.selected[0] {
		t.Error("profile should not be selected initially")
	}

	// Toggle selection with space
	msg := tea.KeyMsg{Type: tea.KeySpace}
	newModel, _ := m.Update(msg)
	m = newModel.(MultiSelectModel)

	if !m.selected[0] {
		t.Error("profile should be selected after space")
	}

	// Toggle again
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if m.selected[0] {
		t.Error("profile should be deselected after second space")
	}
}

func TestMultiSelectModel_Navigation(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "profile1", BaseName: "p1", Mode: "ro"},
		{Name: "profile2", BaseName: "p2", Mode: "ro"},
		{Name: "profile3", BaseName: "p3", Mode: "ro"},
	}

	m := NewMultiSelectModel(profiles)

	// Test down arrow
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := m.Update(msg)
	m = newModel.(MultiSelectModel)

	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after down, got %d", m.cursor)
	}

	// Test up arrow
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after up, got %d", m.cursor)
	}

	// Test boundary at top
	msg = tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor to stay at 0 at boundary, got %d", m.cursor)
	}

	// Move to bottom
	for i := 0; i < len(profiles); i++ {
		msg = tea.KeyMsg{Type: tea.KeyDown}
		newModel, _ = m.Update(msg)
		m = newModel.(MultiSelectModel)
	}

	// Test boundary at bottom
	expectedBottom := len(profiles) - 1
	if m.cursor != expectedBottom {
		t.Errorf("expected cursor at %d at bottom boundary, got %d", expectedBottom, m.cursor)
	}

	// Test 'j' for down
	m = NewMultiSelectModel(profiles)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if m.cursor != 1 {
		t.Errorf("expected cursor at 1 after 'j', got %d", m.cursor)
	}

	// Test 'k' for up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 after 'k', got %d", m.cursor)
	}
}

func TestMultiSelectModel_SelectAll(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "profile1", BaseName: "p1", Mode: "ro"},
		{Name: "profile2", BaseName: "p2", Mode: "ro"},
		{Name: "profile3", BaseName: "p3", Mode: "ro"},
	}

	m := NewMultiSelectModel(profiles)

	// Send 'a' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := m.Update(msg)
	m = newModel.(MultiSelectModel)

	for i, sel := range m.selected {
		if !sel {
			t.Errorf("profile at index %d should be selected after select all", i)
		}
	}
}

func TestMultiSelectModel_DeselectAll(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "profile1", BaseName: "p1", Mode: "ro"},
		{Name: "profile2", BaseName: "p2", Mode: "ro"},
		{Name: "profile3", BaseName: "p3", Mode: "ro"},
	}

	m := NewMultiSelectModel(profiles)

	// First select all
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, _ := m.Update(msg)
	m = newModel.(MultiSelectModel)

	// Verify all selected
	for i, sel := range m.selected {
		if !sel {
			t.Errorf("profile at index %d should be selected before deselect all", i)
		}
	}

	// Send 'n' key to deselect all
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	for i, sel := range m.selected {
		if sel {
			t.Errorf("profile at index %d should not be selected after deselect all", i)
		}
	}
}

func TestMultiSelectModel_Confirm(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "dev-mcpro", BaseName: "dev", Mode: "ro"},
	}

	m := NewMultiSelectModel(profiles)

	// Select a profile
	msg := tea.KeyMsg{Type: tea.KeySpace}
	newModel, _ := m.Update(msg)
	m = newModel.(MultiSelectModel)

	// Send enter
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if !m.WasConfirmed() {
		t.Error("should be confirmed after enter")
	}
	if m.WasCancelled() {
		t.Error("should not be cancelled")
	}
}

func TestMultiSelectModel_Cancel(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "dev-mcpro", BaseName: "dev", Mode: "ro"},
	}

	m := NewMultiSelectModel(profiles)

	// Send 'q' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, _ := m.Update(msg)
	m = newModel.(MultiSelectModel)

	if !m.WasCancelled() {
		t.Error("should be cancelled after 'q'")
	}
	if m.WasConfirmed() {
		t.Error("should not be confirmed")
	}

	// Test with Ctrl+C as well
	m = NewMultiSelectModel(profiles)
	msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if !m.WasCancelled() {
		t.Error("should be cancelled after Ctrl+C")
	}

	// Test with Esc as well
	m = NewMultiSelectModel(profiles)
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ = m.Update(msg)
	m = newModel.(MultiSelectModel)

	if !m.WasCancelled() {
		t.Error("should be cancelled after Esc")
	}
}

func TestSelectedProfiles(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "profile1", BaseName: "p1", Mode: "ro"},
		{Name: "profile2", BaseName: "p2", Mode: "ro"},
		{Name: "profile3", BaseName: "p3", Mode: "ro"},
	}

	m := NewMultiSelectModel(profiles)

	// Select profiles at index 0 and 2
	m.selected[0] = true
	m.selected[2] = true

	selectedProfiles := m.SelectedProfiles()

	if len(selectedProfiles) != 2 {
		t.Errorf("expected 2 selected profiles, got %d", len(selectedProfiles))
	}

	if selectedProfiles[0].Name != "profile1" {
		t.Errorf("expected first selected profile to be 'profile1', got '%s'", selectedProfiles[0].Name)
	}

	if selectedProfiles[1].Name != "profile3" {
		t.Errorf("expected second selected profile to be 'profile3', got '%s'", selectedProfiles[1].Name)
	}
}

func TestView_SSOBadge(t *testing.T) {
	profiles := []MCPProfile{
		{Name: "dev-mcpro", BaseName: "dev", Mode: "ro", Region: "us-east-1", IsSSO: true},
		{Name: "prod-mcpro", BaseName: "prod", Mode: "ro", Region: "eu-west-1", IsSSO: false},
		{Name: "staging-mcpro", BaseName: "staging", Mode: "ro", Region: "us-west-2", IsSSO: true},
	}

	m := NewMultiSelectModel(profiles)
	view := m.View()

	// Count [SSO] occurrences
	ssoCount := strings.Count(view, "[SSO]")
	expectedSSOCount := 2 // dev and staging are SSO

	if ssoCount != expectedSSOCount {
		t.Errorf("expected %d [SSO] badges in view, got %d", expectedSSOCount, ssoCount)
	}

	// Verify SSO profiles have the badge
	if !strings.Contains(view, "dev-mcpro") {
		t.Error("view should contain dev-mcpro")
	}

	// Very basic check that SSO appears for SSO profiles
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "dev-mcpro") && !strings.Contains(line, "[SSO]") {
			t.Error("dev-mcpro line should contain [SSO] badge")
		}
		if strings.Contains(line, "staging-mcpro") && !strings.Contains(line, "[SSO]") {
			t.Error("staging-mcpro line should contain [SSO] badge")
		}
		if strings.Contains(line, "prod-mcpro") && strings.Contains(line, "[SSO]") {
			t.Error("prod-mcpro line should NOT contain [SSO] badge")
		}
	}
}
