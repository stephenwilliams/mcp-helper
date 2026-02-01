package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// State represents the current state of the TUI
type State int

const (
	StateBrowsing State = iota
	StateDetails
	StateConfiguring
	StateInstalling
	StateComplete
)

// installCompleteMsg is sent when installation completes
type installCompleteMsg struct {
	err error
}

// Model represents the TUI application state
type Model struct {
	state         State
	config        *config.Config
	adapter       adapter.Adapter
	servers       []string // sorted server names
	cursor        int
	selected      string // selected server name
	scope         adapter.Scope
	envValues     map[string]string
	envKeys       []string // sorted env var keys for iteration
	currentField  int      // current field in configuration form
	textInput     string   // current text input value
	cursorPos     int      // cursor position in text input
	err           error
	installMsg    string   // installation status message
	width         int
	height        int
}

// NewModel creates a new TUI model
func NewModel(cfg *config.Config, adptr adapter.Adapter) Model {
	// Extract and sort server names
	servers := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		servers = append(servers, name)
	}
	sort.Strings(servers)

	return Model{
		state:     StateBrowsing,
		config:    cfg,
		adapter:   adptr,
		servers:   servers,
		cursor:    0,
		envValues: make(map[string]string),
		scope:     adapter.ScopeUser, // default scope
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case StateBrowsing:
			return m.updateBrowsing(msg)
		case StateDetails:
			return m.updateDetails(msg)
		case StateConfiguring:
			return m.updateConfiguring(msg)
		case StateInstalling:
			return m.updateInstalling(msg)
		case StateComplete:
			return m.updateComplete(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case installCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.installMsg = "Installation failed: " + msg.err.Error()
		} else {
			m.installMsg = "Server installed successfully!"
		}
		m.state = StateComplete
		return m, nil
	}

	return m, nil
}

// updateBrowsing handles input in the browsing state
func (m Model) updateBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.servers)-1 {
			m.cursor++
		}

	case "enter":
		if len(m.servers) > 0 {
			m.selected = m.servers[m.cursor]
			m.state = StateDetails
		}
	}

	return m, nil
}

// updateDetails handles input in the details state
func (m Model) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc", "backspace":
		m.state = StateBrowsing
		m.selected = ""

	case "enter":
		// Move to configuration if server has env vars
		server := m.config.Servers[m.selected]
		if server != nil && len(server.Env) > 0 {
			// Initialize env keys sorted
			m.envKeys = make([]string, 0, len(server.Env))
			for key := range server.Env {
				m.envKeys = append(m.envKeys, key)
			}
			sort.Strings(m.envKeys)

			m.currentField = 0
			m.textInput = ""
			m.cursorPos = 0
			m.state = StateConfiguring
		} else {
			// No env vars, skip to installing
			m.state = StateInstalling
			return m, m.installServer()
		}
	}

	return m, nil
}

// updateConfiguring handles input in the configuring state
func (m Model) updateConfiguring(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.state = StateDetails
		m.envValues = make(map[string]string) // reset values
		m.textInput = ""
		m.currentField = 0

	case "enter", "tab":
		// Save current field value
		if m.currentField < len(m.envKeys) {
			m.envValues[m.envKeys[m.currentField]] = m.textInput
		}

		// Move to next field
		m.currentField++
		if m.currentField >= len(m.envKeys) {
			// All fields filled, proceed to installation
			m.state = StateInstalling
			return m, m.installServer()
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

// updateInstalling handles input in the installing state
func (m Model) updateInstalling(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

// updateComplete handles input in the complete state
func (m Model) updateComplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "enter":
		return m, tea.Quit
	}

	return m, nil
}

// View renders the current state
func (m Model) View() string {
	switch m.state {
	case StateBrowsing:
		return m.viewBrowsing()
	case StateDetails:
		return m.viewDetails()
	case StateConfiguring:
		return m.viewConfiguring()
	case StateInstalling:
		return m.viewInstalling()
	case StateComplete:
		return m.viewComplete()
	}

	return ""
}

// viewBrowsing renders the browsing state
func (m Model) viewBrowsing() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("MCP Server Browser") + "\n\n")

	if len(m.servers) == 0 {
		s.WriteString(errorStyle.Render("No servers found in registry") + "\n")
	} else {
		// Calculate visible range for scrolling
		visibleHeight := m.height - 6 // Account for title, help text, and margins
		if visibleHeight < 1 {
			visibleHeight = 10 // Default minimum
		}

		start := 0
		end := len(m.servers)

		// Implement scrolling if list is longer than screen
		if len(m.servers) > visibleHeight {
			// Keep cursor centered when possible
			start = m.cursor - visibleHeight/2
			if start < 0 {
				start = 0
			}
			end = start + visibleHeight
			if end > len(m.servers) {
				end = len(m.servers)
				start = end - visibleHeight
				if start < 0 {
					start = 0
				}
			}
		}

		// Render visible servers
		for i := start; i < end; i++ {
			name := m.servers[i]
			server := m.config.Servers[name]

			var cursor string
			if i == m.cursor {
				cursor = "► "
			} else {
				cursor = "  "
			}

			// Server name (prominent)
			serverName := cursor + name

			// Transport indicator
			var transportBadge string
			switch server.Transport {
			case "stdio":
				transportBadge = transportStdioStyle.Render("[stdio]")
			case "http", "https":
				transportBadge = transportHTTPStyle.Render("[http]")
			default:
				transportBadge = transportUnknownStyle.Render("[" + server.Transport + "]")
			}

			// Description (truncate if needed)
			description := server.Description
			maxDescLen := 60
			if m.width > 0 && m.width < 80 {
				maxDescLen = m.width - 25 // Adjust for smaller screens
			}
			if len(description) > maxDescLen {
				description = description[:maxDescLen-3] + "..."
			}

			// Build the server line
			if i == m.cursor {
				s.WriteString(selectedStyle.Render(serverName) + " " + transportBadge + "\n")
				if description != "" {
					s.WriteString(descriptionStyle.Render("    " + description) + "\n")
				}
			} else {
				s.WriteString(normalStyle.Render(serverName) + " " + transportBadge + "\n")
				if description != "" {
					s.WriteString(descriptionDimStyle.Render("    " + description) + "\n")
				}
			}
		}

		// Show scroll indicator if needed
		if len(m.servers) > visibleHeight {
			s.WriteString("\n" + infoStyle.Render("  Showing "+
				string(rune('0'+(start+1)/10))+string(rune('0'+(start+1)%10))+
				"-"+
				string(rune('0'+end/10))+string(rune('0'+end%10))+
				" of "+
				string(rune('0'+len(m.servers)/10))+string(rune('0'+len(m.servers)%10))))
		}
	}

	s.WriteString("\n" + helpStyle.Render("↑/k: up • ↓/j: down • enter: select • q: quit"))

	return s.String()
}

// viewDetails renders the details state
func (m Model) viewDetails() string {
	server := m.config.Servers[m.selected]
	if server == nil {
		return errorStyle.Render("Server not found") + "\n"
	}

	var s strings.Builder
	s.WriteString(titleStyle.Render("Server Details") + "\n\n")

	// Server name and description
	s.WriteString(labelStyle.Render("Name: ") + valueStyle.Render(m.selected) + "\n")
	if server.Description != "" {
		s.WriteString(labelStyle.Render("Description: ") + valueStyle.Render(server.Description) + "\n")
	}
	s.WriteString("\n")

	// Transport details
	s.WriteString(labelStyle.Render("Transport: ") + valueStyle.Render(server.Transport) + "\n")

	if server.Transport == "stdio" {
		s.WriteString(labelStyle.Render("Command: ") + valueStyle.Render(server.Command) + "\n")
		if len(server.Args) > 0 {
			s.WriteString(labelStyle.Render("Arguments: ") + valueStyle.Render(strings.Join(server.Args, " ")) + "\n")
		}
	} else if server.Transport == "http" {
		s.WriteString(labelStyle.Render("URL: ") + valueStyle.Render(server.URL) + "\n")
	}
	s.WriteString("\n")

	// Environment variables
	if len(server.Env) > 0 {
		s.WriteString(labelStyle.Render("Environment Variables:") + "\n")

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
				required = infoStyle.Render(" (required)")
			}
			s.WriteString("  " + labelStyle.Render(key) + required + "\n")
			if env.Description != "" {
				s.WriteString("    " + normalStyle.Render(env.Description) + "\n")
			}
		}
		s.WriteString("\n")
	}

	s.WriteString(helpStyle.Render("enter: configure • esc: back • q: quit"))

	return s.String()
}

// viewConfiguring renders the configuring state
func (m Model) viewConfiguring() string {
	server := m.config.Servers[m.selected]
	if server == nil || len(m.envKeys) == 0 {
		return errorStyle.Render("No configuration needed") + "\n"
	}

	var s strings.Builder
	s.WriteString(titleStyle.Render("Configure Environment Variables") + "\n\n")
	s.WriteString(labelStyle.Render("Server: ") + valueStyle.Render(m.selected) + "\n\n")

	// Display all env vars with their current state
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
			if m.isSecretField(key) {
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

	// Progress indicator
	progress := labelStyle.Render("Field ") +
		valueStyle.Render(string(rune('0'+m.currentField+1))) +
		labelStyle.Render(" of ") +
		valueStyle.Render(string(rune('0'+len(m.envKeys))))
	s.WriteString(progress + "\n\n")

	// Help text
	if m.currentField < len(m.envKeys)-1 {
		s.WriteString(helpStyle.Render("enter/tab: next • shift+tab: previous • esc: cancel • q: quit"))
	} else {
		s.WriteString(helpStyle.Render("enter: proceed • shift+tab: previous • esc: cancel • q: quit"))
	}

	return s.String()
}

// renderInputWithCursor renders the input field with a visible cursor
func (m Model) renderInputWithCursor(key string) string {
	if m.isSecretField(key) {
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

// isSecretField checks if a field name indicates it should be masked
func (m Model) isSecretField(key string) bool {
	keyUpper := strings.ToUpper(key)
	secretKeywords := []string{"TOKEN", "KEY", "SECRET", "PASSWORD", "CREDENTIAL", "API_KEY"}
	for _, keyword := range secretKeywords {
		if strings.Contains(keyUpper, keyword) {
			return true
		}
	}
	return false
}

// viewInstalling renders the installing state
func (m Model) viewInstalling() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Installing Server") + "\n\n")
	s.WriteString(labelStyle.Render("Server: ") + valueStyle.Render(m.selected) + "\n")
	s.WriteString(labelStyle.Render("Scope: ") + valueStyle.Render(string(m.scope)) + "\n")
	s.WriteString(labelStyle.Render("Adapter: ") + valueStyle.Render(m.adapter.Name()) + "\n\n")
	s.WriteString(infoStyle.Render("Installing...") + "\n")
	s.WriteString("\n" + helpStyle.Render("Please wait..."))
	return s.String()
}

// viewComplete renders the complete state
func (m Model) viewComplete() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("Installation Complete") + "\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render("✗ " + m.installMsg) + "\n\n")
		s.WriteString(labelStyle.Render("Details:") + "\n")
		s.WriteString(normalStyle.Render(m.err.Error()) + "\n")
	} else {
		s.WriteString(successStyle.Render("✓ " + m.installMsg) + "\n\n")
		s.WriteString(labelStyle.Render("Server: ") + valueStyle.Render(m.selected) + "\n")
		s.WriteString(labelStyle.Render("Scope: ") + valueStyle.Render(string(m.scope)) + "\n")
	}

	s.WriteString("\n" + helpStyle.Render("enter/q: quit"))
	return s.String()
}

// installServer creates a command that installs the server in the background
func (m Model) installServer() tea.Cmd {
	return func() tea.Msg {
		server := m.config.Servers[m.selected]
		if server == nil {
			return installCompleteMsg{err: fmt.Errorf("server not found: %s", m.selected)}
		}

		err := m.adapter.AddServer(m.selected, server, m.scope, m.envValues)
		return installCompleteMsg{err: err}
	}
}
