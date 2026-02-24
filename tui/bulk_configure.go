package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// bulkInstallCompleteMsg is sent when a single server installation completes
type bulkInstallCompleteMsg struct {
	serverName string
	err        error
}

// installResult tracks the result of installing a single server
type installResult struct {
	serverName string
	success    bool
	err        error
}

// BulkConfigureModel handles sequential configuration and installation of multiple servers
type BulkConfigureModel struct {
	servers       []string              // selected server names to configure
	currentIndex  int                   // which server is being configured
	config        *config.Config        // server registry config
	adapter       adapter.Adapter       // adapter for installation
	scope         adapter.Scope         // installation scope
	envKeys       []string              // env var keys for current server
	envValues     map[string]string     // env var values for current server
	currentField  int                   // current field in configuration form
	textInput     string                // current text input value
	cursorPos     int                   // cursor position in text input
	results       []installResult       // installation results for all servers
	installing    bool                  // whether currently installing
	complete      bool                  // whether all servers are done
	confirmCancel bool                  // whether showing cancel confirmation
	width         int                   // terminal width
	height        int                   // terminal height
}

// NewBulkConfigureModel creates a new bulk configuration model
func NewBulkConfigureModel(servers []string, cfg *config.Config, adptr adapter.Adapter, scope adapter.Scope) BulkConfigureModel {
	m := BulkConfigureModel{
		servers:      servers,
		currentIndex: 0,
		config:       cfg,
		adapter:      adptr,
		scope:        scope,
		envValues:    make(map[string]string),
		results:      make([]installResult, 0, len(servers)),
	}

	// Initialize environment variables for first server
	m.loadServerEnvVars()

	return m
}

// loadServerEnvVars loads the environment variables for the current server
func (m *BulkConfigureModel) loadServerEnvVars() {
	if m.currentIndex >= len(m.servers) {
		return
	}

	serverName := m.servers[m.currentIndex]
	server := m.config.Servers[serverName]

	if server == nil || len(server.Env) == 0 {
		m.envKeys = nil
		m.envValues = make(map[string]string)
		return
	}

	// Extract and sort env var keys
	m.envKeys = make([]string, 0, len(server.Env))
	for key := range server.Env {
		m.envKeys = append(m.envKeys, key)
	}
	sort.Strings(m.envKeys)

	// Reset form state
	m.envValues = make(map[string]string)
	m.currentField = 0
	m.textInput = ""
	m.cursorPos = 0
}

// Init initializes the bulk configure model
func (m BulkConfigureModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m BulkConfigureModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.installing {
			return m.updateInstalling(msg)
		} else if m.complete {
			return m.updateComplete(msg)
		} else if m.confirmCancel {
			return m.updateConfirmCancel(msg)
		} else {
			return m.updateConfiguring(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case bulkInstallCompleteMsg:
		// Record the result
		m.results = append(m.results, installResult{
			serverName: msg.serverName,
			success:    msg.err == nil,
			err:        msg.err,
		})

		m.installing = false

		// Move to next server or complete
		m.currentIndex++
		if m.currentIndex >= len(m.servers) {
			m.complete = true
			return m, nil
		}

		// Load next server's env vars
		m.loadServerEnvVars()

		// If next server has no env vars, install immediately
		if len(m.envKeys) == 0 {
			return m, m.installCurrentServer()
		}
	}

	return m, nil
}

// updateConfiguring handles input during configuration
func (m BulkConfigureModel) updateConfiguring(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		// Show cancel confirmation
		m.confirmCancel = true

	case "enter", "tab":
		// Save current field value
		if m.currentField < len(m.envKeys) {
			m.envValues[m.envKeys[m.currentField]] = m.textInput
		}

		// Move to next field
		m.currentField++
		if m.currentField >= len(m.envKeys) {
			// All fields filled, proceed to installation
			m.installing = true
			return m, m.installCurrentServer()
		} else {
			// Load value for next field (if already set)
			m.textInput = m.envValues[m.envKeys[m.currentField]]
			m.cursorPos = len(m.textInput)
		}

	case "shift+tab":
		// Save current field value
		if m.currentField < len(m.envKeys) {
			m.envValues[m.envKeys[m.currentField]] = m.textInput
		}

		// Move to previous field
		if m.currentField > 0 {
			m.currentField--
			m.textInput = m.envValues[m.envKeys[m.currentField]]
			m.cursorPos = len(m.textInput)
		}

	case "backspace":
		if m.cursorPos > 0 {
			m.textInput = m.textInput[:m.cursorPos-1] + m.textInput[m.cursorPos:]
			m.cursorPos--
		}

	case "left":
		if m.cursorPos > 0 {
			m.cursorPos--
		}

	case "right":
		if m.cursorPos < len(m.textInput) {
			m.cursorPos++
		}

	case "home", "ctrl+a":
		m.cursorPos = 0

	case "end", "ctrl+e":
		m.cursorPos = len(m.textInput)

	default:
		// Handle regular character input
		if len(msg.String()) == 1 {
			m.textInput = m.textInput[:m.cursorPos] + msg.String() + m.textInput[m.cursorPos:]
			m.cursorPos++
		}
	}

	return m, nil
}

// updateConfirmCancel handles the cancel confirmation dialog
func (m BulkConfigureModel) updateConfirmCancel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Mark remaining servers as cancelled and complete
		for i := m.currentIndex; i < len(m.servers); i++ {
			if i > m.currentIndex || !m.installing {
				m.results = append(m.results, installResult{
					serverName: m.servers[i],
					success:    false,
					err:        fmt.Errorf("cancelled by user"),
				})
			}
		}
		m.complete = true
		m.confirmCancel = false

	case "n", "N", "esc":
		// Return to configuration
		m.confirmCancel = false
	}

	return m, nil
}

// updateInstalling handles input during installation
func (m BulkConfigureModel) updateInstalling(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

// updateComplete handles input when all servers are done
func (m BulkConfigureModel) updateComplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "enter":
		return m, tea.Quit
	}

	return m, nil
}

// View renders the current state
func (m BulkConfigureModel) View() string {
	if m.complete {
		return m.viewComplete()
	} else if m.confirmCancel {
		return m.viewConfirmCancel()
	} else if m.installing {
		return m.viewInstalling()
	} else {
		return m.viewConfiguring()
	}
}

// viewConfiguring renders the configuration form
func (m BulkConfigureModel) viewConfiguring() string {
	if m.currentIndex >= len(m.servers) {
		return errorStyle.Render("No servers to configure") + "\n"
	}

	serverName := m.servers[m.currentIndex]
	server := m.config.Servers[serverName]
	if server == nil {
		return errorStyle.Render(fmt.Sprintf("Server not found: %s", serverName)) + "\n"
	}

	var s strings.Builder
	s.WriteString(titleStyle.Render("Bulk Server Configuration") + "\n\n")

	// Progress indicator
	progress := fmt.Sprintf("Configuring server %d of %d", m.currentIndex+1, len(m.servers))
	s.WriteString(infoStyle.Render(progress) + "\n\n")

	s.WriteString(labelStyle.Render("Server: ") + valueStyle.Render(serverName) + "\n")
	if server.Description != "" {
		s.WriteString(labelStyle.Render("Description: ") + normalStyle.Render(server.Description) + "\n")
	}
	s.WriteString("\n")

	// If no env vars, show waiting message
	if len(m.envKeys) == 0 {
		s.WriteString(normalStyle.Render("No configuration needed for this server.\n"))
		s.WriteString(infoStyle.Render("Press Enter to install...") + "\n\n")
		s.WriteString(helpStyle.Render("enter: install • esc: cancel remaining • q: quit"))
		return s.String()
	}

	// Display environment variable form
	for i, key := range m.envKeys {
		env := server.Env[key]

		// Field label
		required := ""
		if env.Required {
			required = infoStyle.Render(" *")
		}

		if i == m.currentField {
			// Current field - highlight
			s.WriteString(selectedStyle.Render("► " + key) + required + "\n")
		} else {
			// Other fields
			s.WriteString("  " + labelStyle.Render(key) + required + "\n")
		}

		// Description
		if env.Description != "" {
			s.WriteString("    " + normalStyle.Render(env.Description) + "\n")
		}

		// Value display
		var displayValue string
		if i == m.currentField {
			// Show current input with cursor
			displayValue = m.renderInputWithCursor(key)
		} else if val, ok := m.envValues[key]; ok && val != "" {
			// Show saved value (masked if secret)
			if isSecretField(key) {
				displayValue = strings.Repeat("*", len(val))
			} else {
				displayValue = val
			}
		} else if env.Default != "" {
			displayValue = normalStyle.Render("(default: " + env.Default + ")")
		} else {
			displayValue = normalStyle.Render("(empty)")
		}

		if i == m.currentField {
			s.WriteString("    " + displayValue + "\n")
		} else {
			s.WriteString("    " + valueStyle.Render(displayValue) + "\n")
		}
		s.WriteString("\n")
	}

	// Field progress
	fieldProgress := fmt.Sprintf("Field %d of %d", m.currentField+1, len(m.envKeys))
	s.WriteString(labelStyle.Render(fieldProgress) + "\n\n")

	// Help text
	if m.currentField < len(m.envKeys)-1 {
		s.WriteString(helpStyle.Render("enter/tab: next • shift+tab: previous • esc: cancel remaining • q: quit"))
	} else {
		s.WriteString(helpStyle.Render("enter: install server • shift+tab: previous • esc: cancel remaining • q: quit"))
	}

	return s.String()
}

// viewConfirmCancel renders the cancel confirmation dialog
func (m BulkConfigureModel) viewConfirmCancel() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Cancel Remaining Servers?") + "\n\n")

	remaining := len(m.servers) - m.currentIndex
	if m.installing {
		remaining-- // Don't count currently installing server
	}

	s.WriteString(normalStyle.Render(fmt.Sprintf("You have %d server(s) remaining to configure.", remaining)) + "\n")
	s.WriteString(normalStyle.Render("Are you sure you want to cancel the remaining installations?") + "\n\n")

	// Show what's been done so far
	if len(m.results) > 0 {
		s.WriteString(labelStyle.Render("Already installed:") + "\n")
		for _, result := range m.results {
			if result.success {
				s.WriteString("  " + successStyle.Render("✓") + " " + result.serverName + "\n")
			} else {
				s.WriteString("  " + errorStyle.Render("✗") + " " + result.serverName + "\n")
			}
		}
		s.WriteString("\n")
	}

	s.WriteString(helpStyle.Render("y: cancel remaining • n: continue • esc: continue"))

	return s.String()
}

// viewInstalling renders the installation state
func (m BulkConfigureModel) viewInstalling() string {
	if m.currentIndex >= len(m.servers) {
		return errorStyle.Render("Invalid state") + "\n"
	}

	serverName := m.servers[m.currentIndex]

	var s strings.Builder
	s.WriteString(titleStyle.Render("Installing Server") + "\n\n")

	// Progress
	progress := fmt.Sprintf("Installing server %d of %d", m.currentIndex+1, len(m.servers))
	s.WriteString(infoStyle.Render(progress) + "\n\n")

	s.WriteString(labelStyle.Render("Server: ") + valueStyle.Render(serverName) + "\n")
	s.WriteString(labelStyle.Render("Scope: ") + valueStyle.Render(string(m.scope)) + "\n")
	s.WriteString(labelStyle.Render("Adapter: ") + valueStyle.Render(m.adapter.Name()) + "\n\n")

	s.WriteString(infoStyle.Render("Installing...") + "\n\n")

	// Show previous results
	if len(m.results) > 0 {
		s.WriteString(labelStyle.Render("Previous results:") + "\n")
		for _, result := range m.results {
			if result.success {
				s.WriteString("  " + successStyle.Render("✓") + " " + result.serverName + "\n")
			} else {
				s.WriteString("  " + errorStyle.Render("✗") + " " + result.serverName + "\n")
			}
		}
		s.WriteString("\n")
	}

	s.WriteString(helpStyle.Render("Please wait..."))

	return s.String()
}

// viewComplete renders the completion summary
func (m BulkConfigureModel) viewComplete() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Bulk Installation Complete") + "\n\n")

	// Summary statistics
	successful := 0
	failed := 0
	cancelled := 0

	for _, result := range m.results {
		if result.success {
			successful++
		} else if result.err != nil && result.err.Error() == "cancelled by user" {
			cancelled++
		} else {
			failed++
		}
	}

	s.WriteString(labelStyle.Render("Summary:") + "\n")
	s.WriteString("  " + successStyle.Render(fmt.Sprintf("✓ %d successful", successful)) + "\n")
	if failed > 0 {
		s.WriteString("  " + errorStyle.Render(fmt.Sprintf("✗ %d failed", failed)) + "\n")
	}
	if cancelled > 0 {
		s.WriteString("  " + infoStyle.Render(fmt.Sprintf("⊘ %d cancelled", cancelled)) + "\n")
	}
	s.WriteString("\n")

	// Detailed results
	s.WriteString(labelStyle.Render("Detailed Results:") + "\n")
	for _, result := range m.results {
		if result.success {
			s.WriteString("  " + successStyle.Render("✓") + " " + valueStyle.Render(result.serverName) + "\n")
		} else if result.err != nil && result.err.Error() == "cancelled by user" {
			s.WriteString("  " + infoStyle.Render("⊘") + " " + valueStyle.Render(result.serverName) + " " + normalStyle.Render("(cancelled)") + "\n")
		} else {
			s.WriteString("  " + errorStyle.Render("✗") + " " + valueStyle.Render(result.serverName) + "\n")
			if result.err != nil {
				s.WriteString("    " + errorStyle.Render(result.err.Error()) + "\n")
			}
		}
	}
	s.WriteString("\n")

	s.WriteString(labelStyle.Render("Scope: ") + valueStyle.Render(string(m.scope)) + "\n")
	s.WriteString(labelStyle.Render("Adapter: ") + valueStyle.Render(m.adapter.Name()) + "\n\n")

	s.WriteString(helpStyle.Render("enter/q: quit"))

	return s.String()
}

// renderInputWithCursor renders the input field with a visible cursor
func (m BulkConfigureModel) renderInputWithCursor(key string) string {
	if isSecretField(key) {
		// For secret fields, show asterisks with cursor
		masked := strings.Repeat("*", len(m.textInput))
		if m.cursorPos < len(masked) {
			return masked[:m.cursorPos] + selectedStyle.Render("_") + masked[m.cursorPos:]
		}
		return masked + selectedStyle.Render("_")
	}

	// For normal fields, show text with cursor
	if m.cursorPos < len(m.textInput) {
		return m.textInput[:m.cursorPos] + selectedStyle.Render(string(m.textInput[m.cursorPos])) + m.textInput[m.cursorPos+1:]
	}
	return m.textInput + selectedStyle.Render("_")
}

// installCurrentServer creates a command that installs the current server
func (m BulkConfigureModel) installCurrentServer() tea.Cmd {
	serverName := m.servers[m.currentIndex]

	return func() tea.Msg {
		server := m.config.Servers[serverName]
		if server == nil {
			return bulkInstallCompleteMsg{
				serverName: serverName,
				err:        fmt.Errorf("server not found: %s", serverName),
			}
		}

		err := m.adapter.AddServer(serverName, server, m.scope, m.envValues)
		return bulkInstallCompleteMsg{
			serverName: serverName,
			err:        err,
		}
	}
}

// isSecretField checks if a field name indicates it should be masked
func isSecretField(key string) bool {
	keyUpper := strings.ToUpper(key)
	secretKeywords := []string{"TOKEN", "KEY", "SECRET", "PASSWORD", "CREDENTIAL"}
	for _, keyword := range secretKeywords {
		if strings.Contains(keyUpper, keyword) {
			return true
		}
	}
	return false
}
