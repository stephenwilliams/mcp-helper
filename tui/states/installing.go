package states

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
)

// InstallingHandler handles the installing and complete state logic
type InstallingHandler struct {
	// Styles
	titleStyle   lipgloss.Style
	labelStyle   lipgloss.Style
	valueStyle   lipgloss.Style
	infoStyle    lipgloss.Style
	errorStyle   lipgloss.Style
	successStyle lipgloss.Style
	normalStyle  lipgloss.Style
	helpStyle    lipgloss.Style
}

// NewInstallingHandler creates a new installing state handler with styles
func NewInstallingHandler(styles Styles) *InstallingHandler {
	return &InstallingHandler{
		titleStyle:   styles.Title,
		labelStyle:   styles.Label,
		valueStyle:   styles.Value,
		infoStyle:    styles.Info,
		errorStyle:   styles.Error,
		successStyle: styles.Success,
		normalStyle:  styles.Normal,
		helpStyle:    styles.Help,
	}
}

// InstallingUpdateParams contains parameters for updating installing state
type InstallingUpdateParams struct {
	Msg tea.KeyMsg
}

// InstallingUpdateResult contains the result of updating installing state
type InstallingUpdateResult struct {
	ShouldQuit bool
}

// UpdateInstalling handles input in the installing state
func (h *InstallingHandler) UpdateInstalling(p InstallingUpdateParams) InstallingUpdateResult {
	result := InstallingUpdateResult{}

	switch p.Msg.String() {
	case "q", "ctrl+c":
		result.ShouldQuit = true
	}

	return result
}

// CompleteUpdateParams contains parameters for updating complete state
type CompleteUpdateParams struct {
	Msg tea.KeyMsg
}

// CompleteUpdateResult contains the result of updating complete state
type CompleteUpdateResult struct {
	ShouldQuit bool
}

// UpdateComplete handles input in the complete state
func (h *InstallingHandler) UpdateComplete(p CompleteUpdateParams) CompleteUpdateResult {
	result := CompleteUpdateResult{}

	switch p.Msg.String() {
	case "q", "ctrl+c", "enter":
		result.ShouldQuit = true
	}

	return result
}

// InstallingViewParams contains parameters for rendering installing state
type InstallingViewParams struct {
	Selected    string
	Scope       adapter.Scope
	AdapterName string
}

// ViewInstalling renders the installing state
func (h *InstallingHandler) ViewInstalling(p InstallingViewParams) string {
	var s strings.Builder
	s.WriteString(h.titleStyle.Render("Installing Server") + "\n\n")
	s.WriteString(h.labelStyle.Render("Server: ") + h.valueStyle.Render(p.Selected) + "\n")
	s.WriteString(h.labelStyle.Render("Scope: ") + h.valueStyle.Render(string(p.Scope)) + "\n")
	s.WriteString(h.labelStyle.Render("Adapter: ") + h.valueStyle.Render(p.AdapterName) + "\n\n")
	s.WriteString(h.infoStyle.Render("Installing...") + "\n")
	s.WriteString("\n" + h.helpStyle.Render("Please wait..."))
	return s.String()
}

// CompleteViewParams contains parameters for rendering complete state
type CompleteViewParams struct {
	Selected   string
	Scope      adapter.Scope
	Err        error
	InstallMsg string
}

// ViewComplete renders the complete state
func (h *InstallingHandler) ViewComplete(p CompleteViewParams) string {
	var s strings.Builder
	s.WriteString(h.titleStyle.Render("Installation Complete") + "\n\n")

	if p.Err != nil {
		s.WriteString(h.errorStyle.Render("✗ "+p.InstallMsg) + "\n\n")
		s.WriteString(h.labelStyle.Render("Details:") + "\n")
		s.WriteString(h.normalStyle.Render(p.Err.Error()) + "\n")
	} else {
		s.WriteString(h.successStyle.Render("✓ "+p.InstallMsg) + "\n\n")
		s.WriteString(h.labelStyle.Render("Server: ") + h.valueStyle.Render(p.Selected) + "\n")
		s.WriteString(h.labelStyle.Render("Scope: ") + h.valueStyle.Render(string(p.Scope)) + "\n")
	}

	s.WriteString("\n" + h.helpStyle.Render("enter/q: quit"))
	return s.String()
}
