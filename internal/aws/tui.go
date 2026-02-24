package aws

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiSelectModel represents the TUI model for selecting AWS profiles
type MultiSelectModel struct {
	profiles  []MCPProfile
	cursor    int
	selected  map[int]bool // index -> selected
	confirmed bool
	cancelled bool
	width     int
	height    int
}

// Styles
var (
	titleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	cursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	checkedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	uncheckedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	ssoStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	readOnlyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	readWriteStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

// NewMultiSelectModel creates a new multi-select model
func NewMultiSelectModel(profiles []MCPProfile) MultiSelectModel {
	selected := make(map[int]bool)
	// Initialize all profiles as not selected
	for i := range profiles {
		selected[i] = false
	}

	return MultiSelectModel{
		profiles: profiles,
		cursor:   0,
		selected: selected,
	}
}

// Init initializes the model
func (m MultiSelectModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m MultiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEsc:
			m.cancelled = true
			return m, tea.Quit
		}

		switch msg.String() {
		case "q":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			m.confirmed = true
			return m, tea.Quit

		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]

		case "a":
			for i := range m.profiles {
				m.selected[i] = true
			}

		case "n":
			m.selected = make(map[int]bool)

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.profiles)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

// View renders the TUI
func (m MultiSelectModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("AWS MCP Profile Discovery"))
	b.WriteString("\n\n")
	b.WriteString("Select profiles to add (space to toggle, enter to confirm):\n\n")

	// Profile list
	for i, profile := range m.profiles {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("► ")
		}

		var checkbox string
		if m.selected[i] {
			checkbox = checkedStyle.Render("[x]")
		} else {
			checkbox = uncheckedStyle.Render("[ ]")
		}

		// Format mode
		mode := profile.Mode
		modeDisplay := ""
		if mode == "ro" {
			modeDisplay = readOnlyStyle.Render("read-only")
		} else if mode == "rw" {
			modeDisplay = readWriteStyle.Render("read-write")
		}

		// SSO badge
		ssoBadge := ""
		if profile.IsSSO {
			ssoBadge = ssoStyle.Render("[SSO]")
		}

		// Format line with proper spacing
		line := fmt.Sprintf("%s%s %-15s %-12s %-11s %s",
			cursor,
			checkbox,
			profile.Name,
			profile.Region,
			modeDisplay,
			ssoBadge,
		)

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Footer with count
	b.WriteString("\n")
	selectedCount := 0
	ssoCount := 0
	for i := range m.selected {
		if m.selected[i] {
			selectedCount++
			if m.profiles[i].IsSSO {
				ssoCount++
			}
		}
	}

	countText := fmt.Sprintf("%d of %d selected", selectedCount, len(m.profiles))
	if ssoCount > 0 {
		countText += fmt.Sprintf(" (%d SSO profiles - run 'aws sso login' first)", ssoCount)
	}
	b.WriteString(countText)
	b.WriteString("\n\n")

	// Help text
	help := helpStyle.Render("a: select all • n: deselect all • space: toggle • enter: confirm • q: cancel")
	b.WriteString(help)

	return b.String()
}

// SelectedProfiles returns the list of selected profiles
func (m MultiSelectModel) SelectedProfiles() []MCPProfile {
	var selected []MCPProfile
	for i, profile := range m.profiles {
		if m.selected[i] {
			selected = append(selected, profile)
		}
	}
	return selected
}

// WasCancelled returns true if the user cancelled
func (m MultiSelectModel) WasCancelled() bool {
	return m.cancelled
}

// WasConfirmed returns true if the user confirmed
func (m MultiSelectModel) WasConfirmed() bool {
	return m.confirmed
}
