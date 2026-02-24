package states

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stephenwilliams/mcp-helper/internal/config"
	envpkg "github.com/stephenwilliams/mcp-helper/internal/env"
	"github.com/stephenwilliams/mcp-helper/tui/components"
)

// ConfiguringHandler handles the configuring state logic
type ConfiguringHandler struct {
	// Styles
	titleStyle    lipgloss.Style
	labelStyle    lipgloss.Style
	valueStyle    lipgloss.Style
	normalStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	infoStyle     lipgloss.Style
	helpStyle     lipgloss.Style
	errorStyle    lipgloss.Style
}

// NewConfiguringHandler creates a new configuring state handler with styles
func NewConfiguringHandler(styles Styles) *ConfiguringHandler {
	return &ConfiguringHandler{
		titleStyle:    styles.Title,
		labelStyle:    styles.Label,
		valueStyle:    styles.Value,
		normalStyle:   styles.Normal,
		selectedStyle: styles.Selected,
		infoStyle:     styles.Info,
		helpStyle:     styles.Help,
		errorStyle:    styles.Error,
	}
}

// ConfiguringUpdateParams contains parameters for updating configuring state
type ConfiguringUpdateParams struct {
	Msg          tea.KeyMsg
	Selected     string
	EnvKeys      []string
	EnvValues    map[string]string
	CurrentField int
	TextInput    string
	CursorPos    int
}

// ConfiguringUpdateResult contains the result of updating configuring state
type ConfiguringUpdateResult struct {
	ShouldQuit   bool
	TransitionTo string // "details", "installing", or ""
	EnvValues    map[string]string
	CurrentField int
	TextInput    string
	CursorPos    int
}

// Update handles input in the configuring state
func (h *ConfiguringHandler) Update(p ConfiguringUpdateParams) ConfiguringUpdateResult {
	result := ConfiguringUpdateResult{
		EnvValues:    p.EnvValues,
		CurrentField: p.CurrentField,
		TextInput:    p.TextInput,
		CursorPos:    p.CursorPos,
	}

	switch p.Msg.String() {
	case "q", "ctrl+c":
		result.ShouldQuit = true

	case "esc":
		result.TransitionTo = "details"
		result.EnvValues = make(map[string]string) // reset values
		result.TextInput = ""
		result.CurrentField = 0

	case "enter", "tab":
		// Save current field value
		if result.CurrentField < len(p.EnvKeys) {
			result.EnvValues[p.EnvKeys[result.CurrentField]] = result.TextInput
		}

		// Move to next field
		result.CurrentField++
		if result.CurrentField >= len(p.EnvKeys) {
			// All fields filled, proceed to installation
			result.TransitionTo = "installing"
		} else {
			// Load value for next field (if already set)
			result.TextInput = result.EnvValues[p.EnvKeys[result.CurrentField]]
			result.CursorPos = len(result.TextInput)
		}

	case "shift+tab":
		// Save current field value
		if result.CurrentField < len(p.EnvKeys) {
			result.EnvValues[p.EnvKeys[result.CurrentField]] = result.TextInput
		}

		// Move to previous field
		if result.CurrentField > 0 {
			result.CurrentField--
			result.TextInput = result.EnvValues[p.EnvKeys[result.CurrentField]]
			result.CursorPos = len(result.TextInput)
		}

	case "backspace":
		if result.CursorPos > 0 {
			result.TextInput = result.TextInput[:result.CursorPos-1] + result.TextInput[result.CursorPos:]
			result.CursorPos--
		}

	case "left":
		if result.CursorPos > 0 {
			result.CursorPos--
		}

	case "right":
		if result.CursorPos < len(result.TextInput) {
			result.CursorPos++
		}

	case "home", "ctrl+a":
		result.CursorPos = 0

	case "end", "ctrl+e":
		result.CursorPos = len(result.TextInput)

	default:
		// Handle regular character input
		if len(p.Msg.String()) == 1 {
			result.TextInput = result.TextInput[:result.CursorPos] + p.Msg.String() + result.TextInput[result.CursorPos:]
			result.CursorPos++
		}
	}

	return result
}

// ConfiguringViewParams contains parameters for rendering configuring state
type ConfiguringViewParams struct {
	Selected     string
	Config       *config.Config
	EnvKeys      []string
	EnvValues    map[string]string
	CurrentField int
	TextInput    string
	CursorPos    int
}

// View renders the configuring state
func (h *ConfiguringHandler) View(p ConfiguringViewParams) string {
	server := p.Config.Servers[p.Selected]
	if server == nil || len(p.EnvKeys) == 0 {
		return h.errorStyle.Render("No configuration needed") + "\n"
	}

	var s strings.Builder
	s.WriteString(h.titleStyle.Render("Configure Environment Variables") + "\n\n")
	s.WriteString(h.labelStyle.Render("Server: ") + h.valueStyle.Render(p.Selected) + "\n\n")

	// Display all env vars with their current state
	for i, key := range p.EnvKeys {
		env := server.Env[key]

		// Field label
		required := ""
		if env.Required {
			required = h.infoStyle.Render(" *")
		}

		if i == p.CurrentField {
			// Current field - highlight
			s.WriteString(h.selectedStyle.Render("► "+key) + required + "\n")
		} else {
			// Other fields
			s.WriteString("  " + h.labelStyle.Render(key) + required + "\n")
		}

		// Description
		if env.Description != "" {
			s.WriteString("    " + h.normalStyle.Render(env.Description) + "\n")
		}

		// Value display
		var displayValue string
		if i == p.CurrentField {
			// Show current input with cursor
			displayValue = components.RenderInputWithCursor(p.TextInput, p.CursorPos, key, h.selectedStyle.Render)
		} else if val, ok := p.EnvValues[key]; ok && val != "" {
			// Show saved value (masked if secret)
			if envpkg.IsSecret(key) {
				displayValue = strings.Repeat("*", len(val))
			} else {
				displayValue = val
			}
		} else if env.Default != "" {
			displayValue = h.normalStyle.Render("(default: " + env.Default + ")")
		} else {
			displayValue = h.normalStyle.Render("(empty)")
		}

		if i == p.CurrentField {
			s.WriteString("    " + displayValue + "\n")
		} else {
			s.WriteString("    " + h.valueStyle.Render(displayValue) + "\n")
		}
		s.WriteString("\n")
	}

	// Progress indicator
	progress := h.labelStyle.Render("Field ") +
		h.valueStyle.Render(string(rune('0'+p.CurrentField+1))) +
		h.labelStyle.Render(" of ") +
		h.valueStyle.Render(string(rune('0'+len(p.EnvKeys))))
	s.WriteString(progress + "\n\n")

	// Help text
	if p.CurrentField < len(p.EnvKeys)-1 {
		s.WriteString(h.helpStyle.Render("enter/tab: next • shift+tab: previous • esc: cancel • q: quit"))
	} else {
		s.WriteString(h.helpStyle.Render("enter: proceed • shift+tab: previous • esc: cancel • q: quit"))
	}

	return s.String()
}
