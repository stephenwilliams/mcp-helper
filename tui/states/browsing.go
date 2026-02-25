package states

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// BrowsingHandler handles the browsing state logic
type BrowsingHandler struct {
	// Styles
	titleStyle             lipgloss.Style
	selectedStyle          lipgloss.Style
	valueStyle             lipgloss.Style
	errorStyle             lipgloss.Style
	infoStyle              lipgloss.Style
	labelStyle             lipgloss.Style
	helpStyle              lipgloss.Style
	checkboxCheckedStyle   lipgloss.Style
	checkboxUncheckedStyle lipgloss.Style
	selectedCountStyle     lipgloss.Style
	tabActiveStyle         lipgloss.Style
	tabInactiveStyle       lipgloss.Style
	transportStdioStyle    lipgloss.Style
	transportHTTPStyle     lipgloss.Style
	transportUnknownStyle  lipgloss.Style
	normalStyle            lipgloss.Style
}

// NewBrowsingHandler creates a new browsing state handler with styles
func NewBrowsingHandler(styles Styles) *BrowsingHandler {
	return &BrowsingHandler{
		titleStyle:             styles.Title,
		selectedStyle:          styles.Selected,
		valueStyle:             styles.Value,
		errorStyle:             styles.Error,
		infoStyle:              styles.Info,
		labelStyle:             styles.Label,
		helpStyle:              styles.Help,
		checkboxCheckedStyle:   styles.CheckboxChecked,
		checkboxUncheckedStyle: styles.CheckboxUnchecked,
		selectedCountStyle:     styles.SelectedCount,
		tabActiveStyle:         styles.TabActive,
		tabInactiveStyle:       styles.TabInactive,
		transportStdioStyle:    styles.TransportStdio,
		transportHTTPStyle:     styles.TransportHTTP,
		transportUnknownStyle:  styles.TransportUnknown,
		normalStyle:            styles.Normal,
	}
}

// BrowsingUpdateParams contains parameters for updating browsing state
type BrowsingUpdateParams struct {
	Msg             tea.KeyMsg
	Cursor          int
	PresetCursor    int
	Filtering       bool
	FilterText      string
	MultiSelectMode bool
	ActiveTab       int
	Servers         []string
	FilteredServers []string
	FilteredPresets []string
	MultiSelect     map[string]bool
	Config          *config.Config
	Adapter         adapter.Adapter
	Scope           adapter.Scope
}

// BrowsingUpdateResult contains the result of updating browsing state
type BrowsingUpdateResult struct {
	Cursor          int
	PresetCursor    int
	Filtering       bool
	FilterText      string
	ActiveTab       int
	FilteredServers []string
	FilteredPresets []string
	MultiSelect     map[string]bool
	ShouldQuit      bool
	TransitionTo    string // "details", "bulk_configure", or ""
	Selected        string
	SelectedServers []string
}

// Update handles input in the browsing state
func (h *BrowsingHandler) Update(p BrowsingUpdateParams, updateFilter, updatePresetFilter func()) BrowsingUpdateResult {
	result := BrowsingUpdateResult{
		Cursor:          p.Cursor,
		PresetCursor:    p.PresetCursor,
		Filtering:       p.Filtering,
		FilterText:      p.FilterText,
		ActiveTab:       p.ActiveTab,
		FilteredServers: p.FilteredServers,
		FilteredPresets: p.FilteredPresets,
		MultiSelect:     p.MultiSelect,
	}

	// Get the list to navigate (filtered or all)
	serverList := p.Servers
	if p.MultiSelectMode {
		serverList = p.FilteredServers
	}

	// Handle filter input mode
	if p.Filtering {
		switch p.Msg.String() {
		case "esc":
			result.Filtering = false
			result.FilterText = ""
			updateFilter()
			return result
		case "enter":
			result.Filtering = false
			return result
		case "backspace":
			if len(p.FilterText) > 0 {
				result.FilterText = p.FilterText[:len(p.FilterText)-1]
				updateFilter()
			}
			return result
		case "up", "down", "k", "j", "tab", " ":
			// Exit filter mode and let these keys be handled by navigation below
			result.Filtering = false
			// Don't return - fall through to handle the key
		default:
			if len(p.Msg.String()) == 1 && p.Msg.String() != " " {
				result.FilterText += p.Msg.String()
				updateFilter()
			}
			return result
		}
	}

	switch p.Msg.String() {
	case "q", "ctrl+c":
		result.ShouldQuit = true

	case "esc":
		if p.MultiSelectMode && p.FilterText != "" {
			result.FilterText = ""
			updateFilter()
			updatePresetFilter()
			return result
		}
		result.ShouldQuit = true

	case "tab":
		// Switch tabs (only in multi-select mode)
		if p.MultiSelectMode {
			result.ActiveTab = (p.ActiveTab + 1) % 2
			// Clear filter on tab switch
			result.FilterText = ""
			result.Filtering = false
			updateFilter()
			updatePresetFilter()
		}

	case "up", "k":
		if p.ActiveTab == 0 {
			if result.Cursor > 0 {
				result.Cursor--
			}
		} else {
			if result.PresetCursor > 0 {
				result.PresetCursor--
			}
		}

	case "down", "j":
		if p.ActiveTab == 0 {
			if result.Cursor < len(serverList)-1 {
				result.Cursor++
			}
		} else {
			if result.PresetCursor < len(p.FilteredPresets)-1 {
				result.PresetCursor++
			}
		}

	case "pgup":
		// Move cursor up by page (10 items)
		pageSize := 10
		if p.ActiveTab == 0 {
			result.Cursor -= pageSize
			if result.Cursor < 0 {
				result.Cursor = 0
			}
		} else {
			result.PresetCursor -= pageSize
			if result.PresetCursor < 0 {
				result.PresetCursor = 0
			}
		}

	case "pgdown":
		// Move cursor down by page (10 items)
		pageSize := 10
		if p.ActiveTab == 0 {
			result.Cursor += pageSize
			if result.Cursor >= len(serverList) {
				result.Cursor = len(serverList) - 1
			}
			if result.Cursor < 0 {
				result.Cursor = 0
			}
		} else {
			result.PresetCursor += pageSize
			if result.PresetCursor >= len(p.FilteredPresets) {
				result.PresetCursor = len(p.FilteredPresets) - 1
			}
			if result.PresetCursor < 0 {
				result.PresetCursor = 0
			}
		}

	case "home":
		// Move cursor to first item
		if p.ActiveTab == 0 {
			result.Cursor = 0
		} else {
			result.PresetCursor = 0
		}

	case "end":
		// Move cursor to last item
		if p.ActiveTab == 0 {
			if len(serverList) > 0 {
				result.Cursor = len(serverList) - 1
			}
		} else {
			if len(p.FilteredPresets) > 0 {
				result.PresetCursor = len(p.FilteredPresets) - 1
			}
		}

	case "left", "right":
		// No horizontal navigation in list view - ignore these keys
		// Prevents accidental filter activation from arrow key presses

	case " ":
		// Toggle selection in multi-select mode
		if p.MultiSelectMode {
			if p.ActiveTab == 0 {
				// Servers tab
				if len(serverList) > 0 && result.Cursor < len(serverList) {
					name := serverList[result.Cursor]
					result.MultiSelect[name] = !result.MultiSelect[name]
				}
			} else {
				// Presets tab - toggle all servers in preset
				if len(p.FilteredPresets) > 0 && result.PresetCursor < len(p.FilteredPresets) {
					presetName := p.FilteredPresets[result.PresetCursor]
					preset := p.Config.Presets[presetName]
					if preset != nil {
						// Get available servers in this preset
						var availableInPreset []string
						for _, srvName := range preset.Servers {
							// Only include servers that are in p.Servers (uninstalled)
							for _, s := range p.Servers {
								if s == srvName {
									availableInPreset = append(availableInPreset, srvName)
									break
								}
							}
						}
						// Check if all are selected
						allSelected := len(availableInPreset) > 0
						for _, srvName := range availableInPreset {
							if !result.MultiSelect[srvName] {
								allSelected = false
								break
							}
						}
						// Toggle: if all selected -> deselect all, else select all
						for _, srvName := range availableInPreset {
							result.MultiSelect[srvName] = !allSelected
						}
					}
				}
			}
		}

	case "/":
		// Enter filter mode
		if p.MultiSelectMode {
			result.Filtering = true
		}

	case "enter":
		if p.MultiSelectMode {
			// In multi-select mode, proceed with selected servers
			selectedServers := getSelectedServers(result.MultiSelect)
			if len(selectedServers) > 0 {
				result.TransitionTo = "bulk_configure"
				result.SelectedServers = selectedServers
			}
			// No selection - do nothing
			return result
		}
		// Single-select mode - go to details
		if len(serverList) > 0 && result.Cursor < len(serverList) {
			result.Selected = serverList[result.Cursor]
			result.TransitionTo = "details"
		}

	default:
		// In multi-select mode, typing starts filter
		if p.MultiSelectMode && len(p.Msg.String()) == 1 {
			result.Filtering = true
			result.FilterText = p.Msg.String()
			updateFilter()
		}
	}

	return result
}

// BrowsingViewParams contains parameters for rendering browsing state
type BrowsingViewParams struct {
	MultiSelectMode bool
	ActiveTab       int
	Filtering       bool
	FilterText      string
	FilteredServers []string
	FilteredPresets []string
	Servers         []string
	Cursor          int
	PresetCursor    int
	MultiSelect     map[string]bool
	Config          *config.Config
	Width           int
	Height          int
	AllInstalled    bool
}

// View renders the browsing state
func (h *BrowsingHandler) View(p BrowsingViewParams) string {
	var s strings.Builder

	// Title
	if p.MultiSelectMode {
		s.WriteString(h.titleStyle.Render("Select MCP Servers") + "\n")
	} else {
		s.WriteString(h.titleStyle.Render("MCP Server Browser") + "\n")
	}

	// Tab header (multi-select mode only)
	if p.MultiSelectMode {
		serverTabLabel := "Servers"
		presetTabLabel := "Presets"
		if p.ActiveTab == 0 {
			s.WriteString(h.tabActiveStyle.Render(" " + serverTabLabel + " "))
			s.WriteString(h.tabInactiveStyle.Render(" " + presetTabLabel + " "))
		} else {
			s.WriteString(h.tabInactiveStyle.Render(" " + serverTabLabel + " "))
			s.WriteString(h.tabActiveStyle.Render(" " + presetTabLabel + " "))
		}
		s.WriteString("\n")
	}

	// Filter bar (in multi-select mode)
	if p.MultiSelectMode {
		if p.Filtering {
			s.WriteString(h.labelStyle.Render("Filter: ") + h.valueStyle.Render(p.FilterText) + h.selectedStyle.Render("_") + "\n")
		} else if p.FilterText != "" {
			s.WriteString(h.labelStyle.Render("Filter: ") + h.valueStyle.Render(p.FilterText) + "\n")
		}
	}
	s.WriteString("\n")

	if p.MultiSelectMode && p.ActiveTab == 1 {
		// Render presets tab
		return h.viewPresetsTab(&s, p)
	}

	// Get the list to display
	serverList := p.Servers
	if p.MultiSelectMode {
		serverList = p.FilteredServers
	}

	if len(serverList) == 0 {
		if p.FilterText != "" {
			s.WriteString(h.errorStyle.Render("No servers match filter") + "\n")
		} else if p.AllInstalled {
			s.WriteString(h.infoStyle.Render("All servers are already installed") + "\n")
		} else {
			s.WriteString(h.errorStyle.Render("No servers found in registry") + "\n")
		}
	} else {
		// Calculate visible range for scrolling
		// Reserve lines: title(1) + tabs(1) + filter(1) + blank(1) + scroll indicator(2) + status(1) + help(2) = ~9
		reservedLines := 9
		if p.MultiSelectMode {
			reservedLines = 11 // Extra lines for tabs, filter bar, and selected count
		}
		availableLines := p.Height - reservedLines
		if availableLines < 3 {
			availableLines = 3 // Minimum to show at least a few items
		}

		// Each item takes 2 lines (name + description), so divide available lines by 2
		// to get the number of items we can display
		visibleItems := availableLines / 2
		if visibleItems < 1 {
			visibleItems = 1
		}

		start := 0
		end := len(serverList)

		// Implement scrolling if list is longer than screen
		if len(serverList) > visibleItems {
			// Keep cursor centered when possible
			start = p.Cursor - visibleItems/2
			if start < 0 {
				start = 0
			}
			end = start + visibleItems
			if end > len(serverList) {
				end = len(serverList)
				start = end - visibleItems
				if start < 0 {
					start = 0
				}
			}
		}

		// Render visible servers
		for i := start; i < end; i++ {
			name := serverList[i]
			server := p.Config.Servers[name]
			isCursor := i == p.Cursor

			// Cursor indicator
			cursor := "  "
			if isCursor {
				cursor = "► "
			}

			// Checkbox (multi-select mode only)
			checkbox := ""
			if p.MultiSelectMode {
				if p.MultiSelect[name] {
					checkbox = h.checkboxCheckedStyle.Render("[✓]") + " "
				} else {
					checkbox = h.checkboxUncheckedStyle.Render("[ ]") + " "
				}
			}

			// Transport indicator
			var transportBadge string
			switch server.Transport {
			case "stdio":
				transportBadge = h.transportStdioStyle.Render("[stdio]")
			case "http", "https":
				transportBadge = h.transportHTTPStyle.Render("[http]")
			default:
				transportBadge = h.transportUnknownStyle.Render("[" + server.Transport + "]")
			}

			// Description (truncate if needed)
			description := server.Description
			maxDescLen := 60
			if p.Width > 0 && p.Width < 80 {
				maxDescLen = p.Width - 30
			}
			if len(description) > maxDescLen {
				description = description[:maxDescLen-3] + "..."
			}

			// Build the line without style padding - control alignment manually
			if isCursor {
				// Highlighted row - use white bold to distinguish from transport badges
				highlightedName := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
				s.WriteString(cursor)
				s.WriteString(checkbox)
				s.WriteString(highlightedName.Render(name) + " ")
				s.WriteString(transportBadge)
				s.WriteString("\n")
				if description != "" {
					indent := "  "
					if p.MultiSelectMode {
						indent = "      " // account for cursor + checkbox
					}
					s.WriteString(h.valueStyle.Render(indent+description) + "\n")
				}
			} else {
				// Normal row
				s.WriteString(cursor)
				s.WriteString(checkbox)
				s.WriteString(name + " ")
				s.WriteString(transportBadge)
				s.WriteString("\n")
				if description != "" {
					indent := "  "
					if p.MultiSelectMode {
						indent = "      " // account for cursor + checkbox
					}
					// Use simple gray color without padding
					dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
					s.WriteString(dimStyle.Render(indent+description) + "\n")
				}
			}
		}

		// Show scroll indicator if needed
		if len(serverList) > visibleItems {
			s.WriteString("\n" + h.infoStyle.Render(fmt.Sprintf("  Showing %d-%d of %d", start+1, end, len(serverList))))
		}
	}

	// Status bar (multi-select mode)
	if p.MultiSelectMode {
		selectedCount := getSelectedCount(p.MultiSelect)
		s.WriteString("\n" + h.selectedCountStyle.Render(fmt.Sprintf("Selected: %d", selectedCount)))
	}

	// Help text
	s.WriteString("\n")
	if p.MultiSelectMode {
		if p.Filtering {
			s.WriteString(h.helpStyle.Render("Type to filter • Enter: done • Esc: clear filter"))
		} else {
			s.WriteString(h.helpStyle.Render("Tab: switch tabs • ↑/↓: navigate • Space: toggle • Enter: install selected • Type: filter • Esc: quit"))
		}
	} else {
		s.WriteString(h.helpStyle.Render("↑/k: up • ↓/j: down • enter: select • q: quit"))
	}

	return s.String()
}

// viewPresetsTab renders the presets tab content
func (h *BrowsingHandler) viewPresetsTab(s *strings.Builder, p BrowsingViewParams) string {
	// Filter bar
	if p.Filtering {
		s.WriteString(h.labelStyle.Render("Filter: ") + h.valueStyle.Render(p.FilterText) + h.selectedStyle.Render("_") + "\n")
	} else if p.FilterText != "" {
		s.WriteString(h.labelStyle.Render("Filter: ") + h.valueStyle.Render(p.FilterText) + "\n")
	}
	s.WriteString("\n")

	if len(p.FilteredPresets) == 0 {
		if p.FilterText != "" {
			s.WriteString(h.errorStyle.Render("No presets match filter") + "\n")
		} else {
			s.WriteString(h.infoStyle.Render("No presets available") + "\n")
		}
	} else {
		for i, name := range p.FilteredPresets {
			preset := p.Config.Presets[name]
			isCursor := i == p.PresetCursor

			cursor := "  "
			if isCursor {
				cursor = "► "
			}

			// Count available/total servers
			total := len(preset.Servers)
			available := 0
			selected := 0
			for _, srvName := range preset.Servers {
				for _, s := range p.Servers {
					if s == srvName {
						available++
						if p.MultiSelect[srvName] {
							selected++
						}
						break
					}
				}
			}

			// Build display
			if isCursor {
				// Highlighted row - use white bold to distinguish from other cyan elements
				highlightedName := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
				s.WriteString(cursor)
				s.WriteString(highlightedName.Render(name))
				s.WriteString(fmt.Sprintf(" (%d/%d available", available, total))
				if selected > 0 {
					s.WriteString(fmt.Sprintf(", %d selected", selected))
				}
				s.WriteString(")\n")
				if preset.Description != "" {
					s.WriteString(h.valueStyle.Render("      "+preset.Description) + "\n")
				}
			} else {
				s.WriteString(cursor + name)
				s.WriteString(fmt.Sprintf(" (%d/%d available)\n", available, total))
				if preset.Description != "" {
					dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
					s.WriteString(dimStyle.Render("      "+preset.Description) + "\n")
				}
			}
		}
	}

	// Status and help
	selectedCount := getSelectedCount(p.MultiSelect)
	s.WriteString("\n" + h.selectedCountStyle.Render(fmt.Sprintf("Selected: %d servers", selectedCount)))
	s.WriteString("\n")
	if p.Filtering {
		s.WriteString(h.helpStyle.Render("Type to filter • Enter: done • Esc: clear filter"))
	} else {
		s.WriteString(h.helpStyle.Render("Tab: switch tabs • ↑/↓: navigate • Space: toggle preset servers • Enter: install • Esc: quit"))
	}

	return s.String()
}

// Helper functions

func getSelectedCount(multiSelect map[string]bool) int {
	count := 0
	for _, selected := range multiSelect {
		if selected {
			count++
		}
	}
	return count
}

func getSelectedServers(multiSelect map[string]bool) []string {
	var selected []string
	for name, isSelected := range multiSelect {
		if isSelected {
			selected = append(selected, name)
		}
	}
	sort.Strings(selected)
	return selected
}
