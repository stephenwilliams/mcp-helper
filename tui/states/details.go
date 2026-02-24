package states

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// DetailsHandler handles the details state logic
type DetailsHandler struct {
	// Styles
	titleStyle  lipgloss.Style
	labelStyle  lipgloss.Style
	valueStyle  lipgloss.Style
	normalStyle lipgloss.Style
	infoStyle   lipgloss.Style
	helpStyle   lipgloss.Style
	errorStyle  lipgloss.Style
}

// NewDetailsHandler creates a new details state handler with styles
func NewDetailsHandler(styles Styles) *DetailsHandler {
	return &DetailsHandler{
		titleStyle:  styles.Title,
		labelStyle:  styles.Label,
		valueStyle:  styles.Value,
		normalStyle: styles.Normal,
		infoStyle:   styles.Info,
		helpStyle:   styles.Help,
		errorStyle:  styles.Error,
	}
}

// DetailsUpdateParams contains parameters for updating details state
type DetailsUpdateParams struct {
	Msg      tea.KeyMsg
	Selected string
	Config   *config.Config
}

// DetailsUpdateResult contains the result of updating details state
type DetailsUpdateResult struct {
	ShouldQuit   bool
	TransitionTo string // "browsing", "configuring", "installing", or ""
	EnvKeys      []string
	CurrentField int
	TextInput    string
	CursorPos    int
}

// Update handles input in the details state
func (h *DetailsHandler) Update(p DetailsUpdateParams) DetailsUpdateResult {
	result := DetailsUpdateResult{}

	switch p.Msg.String() {
	case "q", "ctrl+c":
		result.ShouldQuit = true

	case "esc", "backspace":
		result.TransitionTo = "browsing"

	case "enter":
		// Move to configuration if server has env vars
		server := p.Config.Servers[p.Selected]
		if server != nil && len(server.Env) > 0 {
			// Initialize env keys sorted
			result.EnvKeys = make([]string, 0, len(server.Env))
			for key := range server.Env {
				result.EnvKeys = append(result.EnvKeys, key)
			}
			sort.Strings(result.EnvKeys)

			result.CurrentField = 0
			result.TextInput = ""
			result.CursorPos = 0
			result.TransitionTo = "configuring"
		} else {
			// No env vars, skip to installing
			result.TransitionTo = "installing"
		}
	}

	return result
}

// DetailsViewParams contains parameters for rendering details state
type DetailsViewParams struct {
	Selected string
	Config   *config.Config
}

// View renders the details state
func (h *DetailsHandler) View(p DetailsViewParams) string {
	server := p.Config.Servers[p.Selected]
	if server == nil {
		return h.errorStyle.Render("Server not found") + "\n"
	}

	var s strings.Builder
	s.WriteString(h.titleStyle.Render("Server Details") + "\n\n")

	// Server name and description
	s.WriteString(h.labelStyle.Render("Name: ") + h.valueStyle.Render(p.Selected) + "\n")
	if server.Description != "" {
		s.WriteString(h.labelStyle.Render("Description: ") + h.valueStyle.Render(server.Description) + "\n")
	}
	s.WriteString("\n")

	// Transport details
	s.WriteString(h.labelStyle.Render("Transport: ") + h.valueStyle.Render(server.Transport) + "\n")

	if server.Transport == "stdio" {
		s.WriteString(h.labelStyle.Render("Command: ") + h.valueStyle.Render(server.Command) + "\n")
		if len(server.Args) > 0 {
			s.WriteString(h.labelStyle.Render("Arguments: ") + h.valueStyle.Render(strings.Join(server.Args, " ")) + "\n")
		}
	} else if server.Transport == "http" {
		s.WriteString(h.labelStyle.Render("URL: ") + h.valueStyle.Render(server.URL) + "\n")
	}
	s.WriteString("\n")

	// Environment variables
	if len(server.Env) > 0 {
		s.WriteString(h.labelStyle.Render("Environment Variables:") + "\n")

		// Sort env vars for consistent display
		envKeys := make([]string, 0, len(server.Env))
		for key := range server.Env {
			envKeys = append(envKeys, key)
		}
		sort.Strings(envKeys)

		for _, key := range envKeys {
			env := server.Env[key]
			required := ""
			if env.Required {
				required = h.infoStyle.Render(" (required)")
			}
			s.WriteString("  " + h.labelStyle.Render(key) + required + "\n")
			if env.Description != "" {
				s.WriteString("    " + h.normalStyle.Render(env.Description) + "\n")
			}
		}
		s.WriteString("\n")
	}

	s.WriteString(h.helpStyle.Render("enter: configure • esc: back • q: quit"))

	return s.String()
}
