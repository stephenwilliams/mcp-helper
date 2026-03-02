package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stephenwilliams/mcp-helper/internal/mcp"
	"github.com/stephenwilliams/mcp-helper/internal/permissions"
	"github.com/stephenwilliams/mcp-helper/tui/components"
)

// ToolsApproveState represents the current state of the tools approve TUI
type ToolsApproveState int

const (
	ToolsStateDiscovering ToolsApproveState = iota
	ToolsStateSelectTools
	ToolsStateSelectTarget
	ToolsStatePreview
	ToolsStateApplying
	ToolsStateComplete
)

// SelectType represents the selection state of a server
type SelectType int

const (
	SelectNone SelectType = iota
	SelectSome
	SelectAll // Generates wildcard rule
)

// discoveryCompleteMsg is sent when tool discovery completes
type discoveryCompleteMsg struct {
	servers []mcp.ServerInfo
}

// applyCompleteMsg is sent when permissions are applied
type applyCompleteMsg struct {
	err error
}

// ToolsApproveModel handles the tools approval TUI
type ToolsApproveModel struct {
	state         ToolsApproveState
	adapter       permissions.Adapter
	mcpClient     *mcp.Client
	servers       []mcp.ServerInfo
	cursor        int
	viewport      *components.ViewportManager // Viewport for scrolling
	expanded      map[string]bool             // server name -> expanded
	selected      map[string]SelectType       // server -> none/some/all(wildcard)
	selectedTools map[string]map[string]bool  // server -> tool -> selected
	targetFile    string
	targetOptions []permissions.SettingsPath
	targetCursor  int
	dryRun        bool
	preview       string
	err           error
	width         int
	height        int
	spinner       spinner.Model
	showDetail    bool      // Show tool detail dialog
	detailTool    *mcp.Tool // Tool being shown in detail
	detailServer  string    // Server name of tool being shown

	// Public field for error checking
	Err error
}

// NewToolsApproveModel creates a new tools approve model
func NewToolsApproveModel(adapter permissions.Adapter, dryRun bool, targetFile string) ToolsApproveModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorCyan)

	return ToolsApproveModel{
		state:         ToolsStateDiscovering,
		adapter:       adapter,
		mcpClient:     mcp.NewClient(mcp.DefaultTimeout, mcp.NewCache(mcp.DefaultCacheTTL)),
		viewport:      components.NewViewportManager(7, 10), // 7 reserved lines, 10 minimum height
		expanded:      make(map[string]bool),
		selected:      make(map[string]SelectType),
		selectedTools: make(map[string]map[string]bool),
		dryRun:        dryRun,
		targetFile:    targetFile,
		spinner:       s,
	}
}

// Init initializes the model and starts tool discovery
func (m ToolsApproveModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.discoverTools(),
	)
}

// Update handles messages and updates the model
func (m ToolsApproveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case ToolsStateDiscovering:
			return m.updateDiscovering(msg)
		case ToolsStateSelectTools:
			return m.updateSelectTools(msg)
		case ToolsStateSelectTarget:
			return m.updateSelectTarget(msg)
		case ToolsStatePreview:
			return m.updatePreview(msg)
		case ToolsStateApplying:
			return m.updateApplying(msg)
		case ToolsStateComplete:
			return m.updateComplete(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.SetTerminalHeight(msg.Height)

	case spinner.TickMsg:
		if m.state == ToolsStateDiscovering || m.state == ToolsStateApplying {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case discoveryCompleteMsg:
		m.servers = msg.servers
		m.state = ToolsStateSelectTools
		// Keep servers collapsed by default so user can see overview first
		for _, srv := range m.servers {
			m.expanded[srv.Name] = false
		}
		return m, nil

	case applyCompleteMsg:
		m.err = msg.err
		m.Err = msg.err
		m.state = ToolsStateComplete
		return m, nil
	}

	return m, nil
}

// View renders the current state
func (m ToolsApproveModel) View() string {
	switch m.state {
	case ToolsStateDiscovering:
		return m.viewDiscovering()
	case ToolsStateSelectTools:
		return m.viewSelectTools()
	case ToolsStateSelectTarget:
		return m.viewSelectTarget()
	case ToolsStatePreview:
		return m.viewPreview()
	case ToolsStateApplying:
		return m.viewApplying()
	case ToolsStateComplete:
		return m.viewComplete()
	}
	return ""
}

// discoverTools starts tool discovery in the background
func (m ToolsApproveModel) discoverTools() tea.Cmd {
	return func() tea.Msg {
		// Get MCP server configurations
		serverConfigs, err := m.adapter.GetMCPServers()
		if err != nil {
			return discoveryCompleteMsg{
				servers: []mcp.ServerInfo{{
					Name:  "error",
					Error: fmt.Errorf("failed to get MCP servers: %w", err),
				}},
			}
		}

		// Discover tools from all servers
		ctx := context.Background()
		servers := m.mcpClient.DiscoverTools(ctx, serverConfigs, true)

		return discoveryCompleteMsg{servers: servers}
	}
}

// updateDiscovering handles input during discovery
func (m ToolsApproveModel) updateDiscovering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// updateSelectTools handles input during tool selection
func (m ToolsApproveModel) updateSelectTools(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If detail dialog is open, handle its keys first
	if m.showDetail {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "d", "?":
			// Close dialog
			m.showDetail = false
			m.detailTool = nil
			m.detailServer = ""
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.adjustViewport()
		}

	case "down", "j":
		maxCursor := m.getMaxCursor()
		if m.cursor < maxCursor {
			m.cursor++
			m.adjustViewport()
		}

	case "enter", "right", "l":
		// Toggle expansion or proceed
		serverIdx, toolIdx := m.cursorToServerAndTool()
		if serverIdx >= 0 && serverIdx < len(m.servers) {
			srv := m.servers[serverIdx]
			if srv.Error == nil {
				wasExpanded := m.expanded[srv.Name]
				m.expanded[srv.Name] = !m.expanded[srv.Name]

				// If collapsing and cursor was on a tool, move cursor to server row
				if wasExpanded && toolIdx >= 0 {
					m.cursor = m.serverToRow(serverIdx)
				}
				m.adjustViewport()
			}
		}

	case " ":
		// Toggle selection
		serverIdx, toolIdx := m.cursorToServerAndTool()
		if serverIdx >= 0 && serverIdx < len(m.servers) {
			srv := m.servers[serverIdx]
			if srv.Error != nil {
				return m, nil
			}

			if toolIdx == -1 {
				// Server row - toggle all tools in server
				m.toggleServerSelection(srv.Name)
			} else if toolIdx >= 0 && toolIdx < len(srv.Tools) {
				// Tool row - toggle individual tool
				m.toggleToolSelection(srv.Name, srv.Tools[toolIdx].Name)
			}
		}

	case "*":
		// Wildcard selection for current server
		serverIdx, _ := m.cursorToServerAndTool()
		if serverIdx >= 0 && serverIdx < len(m.servers) {
			srv := m.servers[serverIdx]
			if srv.Error == nil {
				m.setWildcardSelection(srv.Name)
			}
		}

	case "tab":
		// Proceed to target selection
		if m.hasSelection() {
			m.targetOptions = m.adapter.GetSettingsPaths()
			m.targetCursor = 0

			// Pre-select target: use specified file, or default to local scope
			if m.targetFile != "" {
				for i, opt := range m.targetOptions {
					if opt.Path == m.targetFile {
						m.targetCursor = i
						break
					}
				}
			} else {
				// Default to local scope (.claude/settings.local.json)
				for i, opt := range m.targetOptions {
					if opt.Scope == "local" {
						m.targetCursor = i
						break
					}
				}
			}

			m.state = ToolsStateSelectTarget
		}

	case "d", "?":
		// Show tool detail dialog (only when cursor is on a tool)
		serverIdx, toolIdx := m.cursorToServerAndTool()
		if serverIdx >= 0 && toolIdx >= 0 {
			srv := m.servers[serverIdx]
			if toolIdx < len(srv.Tools) {
				m.detailTool = &srv.Tools[toolIdx]
				m.detailServer = srv.Name
				m.showDetail = true
			}
		}
	}

	return m, nil
}

// updateSelectTarget handles input during target selection
func (m ToolsApproveModel) updateSelectTarget(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.state = ToolsStateSelectTools

	case "up", "k":
		if m.targetCursor > 0 {
			m.targetCursor--
		}

	case "down", "j":
		if m.targetCursor < len(m.targetOptions)-1 {
			m.targetCursor++
		}

	case "enter":
		if m.targetCursor >= 0 && m.targetCursor < len(m.targetOptions) {
			m.targetFile = m.targetOptions[m.targetCursor].Path
			m.preview = m.generatePreview()
			m.state = ToolsStatePreview
		}
	}

	return m, nil
}

// updatePreview handles input during preview
func (m ToolsApproveModel) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc":
		m.state = ToolsStateSelectTarget

	case "enter":
		if m.dryRun {
			// Dry run mode - just quit
			return m, tea.Quit
		}
		// Apply changes
		m.state = ToolsStateApplying
		return m, tea.Batch(
			m.spinner.Tick,
			m.applyPermissions(),
		)
	}

	return m, nil
}

// updateApplying handles input during application
func (m ToolsApproveModel) updateApplying(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// updateComplete handles input when complete
func (m ToolsApproveModel) updateComplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "enter":
		return m, tea.Quit
	}
	return m, nil
}

// viewDiscovering renders the discovery state
func (m ToolsApproveModel) viewDiscovering() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("MCP Tool Pre-Approval") + "\n\n")
	s.WriteString(m.spinner.View() + " Discovering tools from configured servers...\n\n")
	s.WriteString(helpStyle.Render("Please wait..."))
	return s.String()
}

// viewSelectTools renders the tool selection state
func (m ToolsApproveModel) viewSelectTools() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("MCP Tool Pre-Approval") + "\n\n")

	// If showing detail dialog, render compact view with dialog
	if m.showDetail && m.detailTool != nil {
		s.WriteString(m.renderDetailDialog())
		s.WriteString("\n\n")
		s.WriteString(helpStyle.Render("d/?/Esc: close details | q: quit"))
		return s.String()
	}

	s.WriteString(normalStyle.Render("Select tools to pre-approve (Space to toggle, * for all from server):") + "\n\n")

	if len(m.servers) == 0 {
		s.WriteString(errorStyle.Render("No MCP servers configured") + "\n")
	} else {
		viewportHeight := m.getViewportHeight()
		viewportEnd := m.viewport.Start + viewportHeight
		currentRow := 0
		renderedRows := 0

		for _, srv := range m.servers {
			// Server row
			if currentRow >= m.viewport.Start && currentRow < viewportEnd {
				cursor := "  "
				if currentRow == m.cursor {
					cursor = "► "
				}

				// Selection indicator
				var checkbox string
				selType := m.selected[srv.Name]
				switch selType {
				case SelectAll:
					checkbox = checkboxCheckedStyle.Render("◉") // Wildcard
				case SelectSome:
					checkbox = checkboxCheckedStyle.Render("◐") // Partial
				case SelectNone:
					checkbox = checkboxUncheckedStyle.Render("○")
				}

				// Expansion indicator
				expandIcon := "▸"
				if m.expanded[srv.Name] {
					expandIcon = "▾"
				}

				// Server name with status
				serverLine := fmt.Sprintf("%s %s %s %s [%s]", cursor, checkbox, expandIcon, srv.Name, srv.Scope)
				if srv.Error != nil {
					// Truncate error message
					errMsg := srv.Error.Error()
					if len(errMsg) > 40 {
						errMsg = errMsg[:37] + "..."
					}
					serverLine += lipgloss.NewStyle().Foreground(colorRed).Render(fmt.Sprintf(" (error: %s)", errMsg))
				} else {
					toolCount := len(srv.Tools)
					selectedCount := 0
					if tools, ok := m.selectedTools[srv.Name]; ok {
						selectedCount = len(tools)
					}
					if selType == SelectAll {
						serverLine += lipgloss.NewStyle().Foreground(colorYellow).Render(fmt.Sprintf(" (*/%d - wildcard)", toolCount))
					} else {
						serverLine += lipgloss.NewStyle().Foreground(colorYellow).Render(fmt.Sprintf(" (%d/%d selected)", selectedCount, toolCount))
					}
				}

				if currentRow == m.cursor {
					s.WriteString(lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(serverLine) + "\n")
				} else {
					s.WriteString(serverLine + "\n")
				}
				renderedRows++
			}
			currentRow++

			// Tool rows (if expanded and no error)
			if m.expanded[srv.Name] && srv.Error == nil {
				for _, tool := range srv.Tools {
					if currentRow >= m.viewport.Start && currentRow < viewportEnd {
						toolCursor := "  "
						if currentRow == m.cursor {
							toolCursor = "► "
						}

						// Tool selection indicator
						toolSelected := false
						if tools, ok := m.selectedTools[srv.Name]; ok {
							toolSelected = tools[tool.Name]
						}

						var toolCheckbox string
						if toolSelected {
							toolCheckbox = checkboxCheckedStyle.Render("●")
						} else {
							toolCheckbox = checkboxUncheckedStyle.Render("○")
						}

						// Truncate description to prevent line overflow
						desc := tool.Description
						maxDescLen := 60
						if m.width > 0 {
							// Adjust based on terminal width, leaving room for checkbox and name
							maxDescLen = m.width - len(tool.Name) - 15
							if maxDescLen < 20 {
								maxDescLen = 20
							}
							if maxDescLen > 80 {
								maxDescLen = 80
							}
						}
						if len(desc) > maxDescLen {
							desc = desc[:maxDescLen-3] + "..."
						}

						// Build tool line with consistent indentation
						toolLine := fmt.Sprintf("  %s %s %s", toolCursor, toolCheckbox, tool.Name)
						if desc != "" {
							toolLine += " - " + desc
						}

						if currentRow == m.cursor {
							s.WriteString(lipgloss.NewStyle().Foreground(colorCyan).Bold(true).Render(toolLine) + "\n")
						} else {
							// Use gray for description part only
							basePart := fmt.Sprintf("  %s %s %s", toolCursor, toolCheckbox, tool.Name)
							s.WriteString(basePart)
							if desc != "" {
								s.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render(" - " + desc))
							}
							s.WriteString("\n")
						}
						renderedRows++
					}
					currentRow++
				}
			}
		}

		// Show scroll indicators
		totalRows := m.getMaxCursor() + 1
		if m.viewport.Start > 0 {
			s.WriteString(infoStyle.Render(fmt.Sprintf("  ↑ %d more above", m.viewport.Start)) + "\n")
		}
		if viewportEnd < totalRows {
			s.WriteString(infoStyle.Render(fmt.Sprintf("  ↓ %d more below", totalRows-viewportEnd)) + "\n")
		}
	}

	// Status
	s.WriteString("\n")
	wildcardCount, toolCount := m.getSelectionCounts()
	s.WriteString(selectedCountStyle.Render(fmt.Sprintf("Selected: %d wildcard + %d tools", wildcardCount, toolCount)) + "\n")

	// Help
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑↓: navigate | Space: toggle | *: wildcard | Enter/→: expand/collapse | d: details | Tab: continue | q: quit"))

	return s.String()
}

// renderDetailDialog renders a dialog showing full tool details
func (m ToolsApproveModel) renderDetailDialog() string {
	var s strings.Builder

	// Dialog box style
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorCyan).
		Padding(1, 2).
		Width(min(m.width-4, 80))

	// Content
	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorCyan).Render("Tool Details") + "\n\n")
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Server: ") + m.detailServer + "\n")
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Tool: ") + m.detailTool.Name + "\n\n")

	if m.detailTool.Description != "" {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render("Description:") + "\n")
		// Word wrap the description
		desc := m.detailTool.Description
		maxWidth := min(m.width-10, 76)
		wrapped := wordWrap(desc, maxWidth)
		s.WriteString(wrapped + "\n")
	} else {
		s.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("(no description)") + "\n")
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(colorGray).Render("Press 'd' or '?' to close"))

	return dialogStyle.Render(s.String())
}

// wordWrap wraps text to the specified width
func wordWrap(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	var lineLen int

	words := strings.Fields(text)
	for i, word := range words {
		wordLen := len(word)
		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		} else if i > 0 && lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}
		result.WriteString(word)
		lineLen += wordLen
	}

	return result.String()
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// viewSelectTarget renders the target selection state
func (m ToolsApproveModel) viewSelectTarget() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("MCP Tool Pre-Approval") + "\n\n")
	s.WriteString(normalStyle.Render("Select target settings file:") + "\n\n")

	for i, opt := range m.targetOptions {
		cursor := "  "
		if i == m.targetCursor {
			cursor = "► "
		}

		existsStr := ""
		if opt.Exists {
			existsStr = successStyle.Render(" (exists)")
			if opt.MCPRuleCount > 0 {
				existsStr += infoStyle.Render(fmt.Sprintf(" - %d MCP rules", opt.MCPRuleCount))
			}
		} else {
			existsStr = infoStyle.Render(" (will create)")
		}

		line := fmt.Sprintf("%s%s [%s]%s", cursor, opt.Path, opt.Scope, existsStr)
		if i == m.targetCursor {
			s.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			s.WriteString(line + "\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("↑↓: navigate | Enter: select | Esc: back | q: quit"))

	return s.String()
}

// viewPreview renders the preview state
func (m ToolsApproveModel) viewPreview() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("MCP Tool Pre-Approval") + "\n\n")
	s.WriteString(labelStyle.Render("Changes to be applied to "+m.targetFile+":") + "\n\n")
	s.WriteString(m.preview + "\n\n")

	if m.dryRun {
		s.WriteString(infoStyle.Render("DRY RUN MODE - No changes will be applied") + "\n\n")
		s.WriteString(helpStyle.Render("Enter: quit | Esc: back | q: quit"))
	} else {
		s.WriteString(helpStyle.Render("Enter: apply changes | Esc: back | q: quit"))
	}

	return s.String()
}

// viewApplying renders the applying state
func (m ToolsApproveModel) viewApplying() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("MCP Tool Pre-Approval") + "\n\n")
	s.WriteString(m.spinner.View() + " Applying permissions...\n\n")
	s.WriteString(helpStyle.Render("Please wait..."))
	return s.String()
}

// viewComplete renders the complete state
func (m ToolsApproveModel) viewComplete() string {
	var s strings.Builder
	s.WriteString(titleStyle.Render("MCP Tool Pre-Approval Complete") + "\n\n")

	if m.err != nil {
		s.WriteString(errorStyle.Render("✗ Failed to apply permissions") + "\n\n")
		s.WriteString(labelStyle.Render("Error: ") + m.err.Error() + "\n")
	} else {
		s.WriteString(successStyle.Render("✓ Permissions applied successfully") + "\n\n")
		wildcardCount, toolCount := m.getSelectionCounts()
		s.WriteString(labelStyle.Render("Applied: ") + fmt.Sprintf("%d wildcard + %d specific tool permissions\n", wildcardCount, toolCount))
		s.WriteString(labelStyle.Render("Target: ") + m.targetFile + "\n")
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("enter/q: quit"))

	return s.String()
}

// Helper methods

func (m *ToolsApproveModel) getMaxCursor() int {
	count := 0
	for _, srv := range m.servers {
		count++ // Server row
		if m.expanded[srv.Name] && srv.Error == nil {
			count += len(srv.Tools)
		}
	}
	return count - 1
}

// getViewportHeight returns the number of rows available for the list
func (m *ToolsApproveModel) getViewportHeight() int {
	return m.viewport.GetViewportHeight()
}

// adjustViewport ensures the cursor is visible within the viewport
func (m *ToolsApproveModel) adjustViewport() {
	m.viewport.AdjustForCursor(m.cursor, m.getMaxCursor()+1)
}

func (m *ToolsApproveModel) cursorToServerAndTool() (serverIdx, toolIdx int) {
	currentRow := 0
	for srvIdx, srv := range m.servers {
		if currentRow == m.cursor {
			return srvIdx, -1 // Server row
		}
		currentRow++

		if m.expanded[srv.Name] && srv.Error == nil {
			for tIdx := range srv.Tools {
				if currentRow == m.cursor {
					return srvIdx, tIdx // Tool row
				}
				currentRow++
			}
		}
	}
	return -1, -1
}

// serverToRow converts a server index to the row number it occupies
func (m *ToolsApproveModel) serverToRow(serverIdx int) int {
	row := 0
	for i, srv := range m.servers {
		if i == serverIdx {
			return row
		}
		row++ // Server row
		if m.expanded[srv.Name] && srv.Error == nil {
			row += len(srv.Tools) // Tool rows
		}
	}
	return 0
}

func (m *ToolsApproveModel) toggleServerSelection(serverName string) {
	selType := m.selected[serverName]
	if selType == SelectAll || selType == SelectSome {
		// Deselect all (handles both wildcard and partial selection)
		m.selected[serverName] = SelectNone
		delete(m.selectedTools, serverName)
	} else {
		// Select all individual tools (not wildcard)
		for _, srv := range m.servers {
			if srv.Name == serverName {
				if m.selectedTools[serverName] == nil {
					m.selectedTools[serverName] = make(map[string]bool)
				}
				for _, tool := range srv.Tools {
					m.selectedTools[serverName][tool.Name] = true
				}
				m.updateSelectionType(serverName)
				break
			}
		}
	}
}

func (m *ToolsApproveModel) toggleToolSelection(serverName, toolName string) {
	// Can't toggle individual tools if wildcard is set
	if m.selected[serverName] == SelectAll {
		return
	}

	if m.selectedTools[serverName] == nil {
		m.selectedTools[serverName] = make(map[string]bool)
	}

	m.selectedTools[serverName][toolName] = !m.selectedTools[serverName][toolName]
	if !m.selectedTools[serverName][toolName] {
		delete(m.selectedTools[serverName], toolName)
	}

	m.updateSelectionType(serverName)
}

func (m *ToolsApproveModel) setWildcardSelection(serverName string) {
	if m.selected[serverName] == SelectAll {
		// Already wildcard, deselect
		m.selected[serverName] = SelectNone
		delete(m.selectedTools, serverName)
	} else {
		// Set wildcard
		m.selected[serverName] = SelectAll
		delete(m.selectedTools, serverName) // Clear individual selections
	}
}

func (m *ToolsApproveModel) updateSelectionType(serverName string) {
	tools := m.selectedTools[serverName]
	if len(tools) == 0 {
		m.selected[serverName] = SelectNone
		return
	}

	// Check if all tools are selected
	for _, srv := range m.servers {
		if srv.Name == serverName {
			if len(tools) == len(srv.Tools) {
				// Could be SelectAll, but we keep it as SelectSome for individual selections
				// User must explicitly use * for wildcard
				m.selected[serverName] = SelectSome
			} else {
				m.selected[serverName] = SelectSome
			}
			break
		}
	}
}

func (m *ToolsApproveModel) hasSelection() bool {
	for _, selType := range m.selected {
		if selType != SelectNone {
			return true
		}
	}
	for _, tools := range m.selectedTools {
		if len(tools) > 0 {
			return true
		}
	}
	return false
}

func (m *ToolsApproveModel) getSelectionCounts() (wildcardCount, toolCount int) {
	for serverName, selType := range m.selected {
		if selType == SelectAll {
			wildcardCount++
		} else if tools, ok := m.selectedTools[serverName]; ok {
			toolCount += len(tools)
		}
	}
	return
}

func (m *ToolsApproveModel) generatePreview() string {
	var s strings.Builder

	// Load existing permissions
	existing, err := m.adapter.LoadPermissions(m.targetFile)
	if err != nil {
		// File might not exist yet
		existing = []permissions.PermissionRule{}
	}

	// Generate new rules
	var newRules []permissions.PermissionRule

	// Add wildcard rules
	serverNames := make([]string, 0, len(m.selected))
	for name := range m.selected {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	for _, serverName := range serverNames {
		if m.selected[serverName] == SelectAll {
			rule := m.adapter.FormatWildcardRule(serverName)
			newRules = append(newRules, rule)
		}
	}

	// Add individual tool rules
	for _, serverName := range serverNames {
		if tools, ok := m.selectedTools[serverName]; ok {
			toolNames := make([]string, 0, len(tools))
			for toolName := range tools {
				toolNames = append(toolNames, toolName)
			}
			sort.Strings(toolNames)

			for _, toolName := range toolNames {
				rule := m.adapter.FormatToolRule(serverName, toolName)
				newRules = append(newRules, rule)
			}
		}
	}

	// Show preview
	s.WriteString("  \"permissions\": {\n")
	s.WriteString("    \"allow\": [\n")

	if len(existing) > 0 {
		s.WriteString(normalStyle.Render("      // ... existing rules preserved ...\n"))
	}

	for _, rule := range newRules {
		s.WriteString(successStyle.Render("+     \"" + string(rule) + "\",\n"))
	}

	s.WriteString("    ]\n")
	s.WriteString("  }\n\n")

	wildcardCount, toolCount := m.getSelectionCounts()
	s.WriteString(fmt.Sprintf("%d new permissions will be added (%d wildcard, %d specific).\n", wildcardCount+toolCount, wildcardCount, toolCount))
	s.WriteString("Existing permissions will be preserved.")

	return s.String()
}

func (m *ToolsApproveModel) applyPermissions() tea.Cmd {
	return func() tea.Msg {
		// Generate rules
		var newRules []permissions.PermissionRule

		// Add wildcard rules
		serverNames := make([]string, 0, len(m.selected))
		for name := range m.selected {
			serverNames = append(serverNames, name)
		}
		sort.Strings(serverNames)

		for _, serverName := range serverNames {
			if m.selected[serverName] == SelectAll {
				rule := m.adapter.FormatWildcardRule(serverName)
				newRules = append(newRules, rule)
			}
		}

		// Add individual tool rules
		for _, serverName := range serverNames {
			if tools, ok := m.selectedTools[serverName]; ok {
				toolNames := make([]string, 0, len(tools))
				for toolName := range tools {
					toolNames = append(toolNames, toolName)
				}
				sort.Strings(toolNames)

				for _, toolName := range toolNames {
					rule := m.adapter.FormatToolRule(serverName, toolName)
					newRules = append(newRules, rule)
				}
			}
		}

		// Load existing permissions and merge with new rules
		existing, err := m.adapter.LoadPermissions(m.targetFile)
		if err != nil {
			// File might not exist yet, treat as empty
			existing = []permissions.PermissionRule{}
		}

		// Merge new rules with existing (handles deduplication and wildcard coverage)
		merged := permissions.MergeRules(existing, newRules)

		// Save merged permissions
		err = m.adapter.SavePermissions(m.targetFile, merged)
		return applyCompleteMsg{err: err}
	}
}
