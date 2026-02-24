package tui

import (
	"context"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stephenwilliams/mcp-helper/internal/adapter"
	"github.com/stephenwilliams/mcp-helper/internal/app"
	"github.com/stephenwilliams/mcp-helper/internal/config"
	"github.com/stephenwilliams/mcp-helper/tui/states"
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
	installer     ServerInstaller   // service layer for installation
	servers       []string          // sorted server names
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

	// Multi-select mode fields
	multiSelectMode  bool              // whether multi-select is enabled
	multiSelect      map[string]bool   // tracks selected servers
	filterText       string            // current filter text
	filteredServers  []string          // servers matching filter
	filtering        bool              // whether in filter input mode
	allInstalled     bool              // true when all servers from config are already installed

	// Tab state (only used in multiSelectMode)
	activeTab       int      // 0 = servers, 1 = presets
	presets         []string // available preset names (filtered for availability)
	filteredPresets []string // filtered preset names (from filterText)
	presetCursor    int      // cursor position in presets tab

	// State handlers
	browsingHandler    *states.BrowsingHandler
	detailsHandler     *states.DetailsHandler
	configuringHandler *states.ConfiguringHandler
	installingHandler  *states.InstallingHandler
}

// NewModel creates a new TUI model
func NewModel(cfg *config.Config, adptr adapter.Adapter) Model {
	return NewModelWithOptions(cfg, adptr, adapter.ScopeUser, false)
}

// NewModelWithOptions creates a new TUI model with options
func NewModelWithOptions(cfg *config.Config, adptr adapter.Adapter, scope adapter.Scope, multiSelect bool) Model {
	// Create the server installer service
	installer := app.NewServerInstaller(cfg, adptr)
	return NewModelWithInstaller(cfg, adptr, installer, scope, multiSelect)
}

// NewModelWithInstaller creates a new TUI model with a custom installer
func NewModelWithInstaller(cfg *config.Config, adptr adapter.Adapter, installer ServerInstaller, scope adapter.Scope, multiSelect bool) Model {
	// Extract and sort server names, filtering out already-installed servers
	servers := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		if adptr.ServerExists(name, scope) {
			continue
		}
		servers = append(servers, name)
	}
	sort.Strings(servers)

	allInstalled := len(cfg.Servers) > 0 && len(servers) == 0

	// Initialize styles for state handlers
	stylesConfig := states.Styles{
		Title:             titleStyle,
		Selected:          selectedStyle,
		Normal:            normalStyle,
		Help:              helpStyle,
		Error:             errorStyle,
		Success:           successStyle,
		Info:              infoStyle,
		Label:             labelStyle,
		Value:             valueStyle,
		TransportStdio:    transportStdioStyle,
		TransportHTTP:     transportHTTPStyle,
		TransportUnknown:  transportUnknownStyle,
		CheckboxChecked:   checkboxCheckedStyle,
		CheckboxUnchecked: checkboxUncheckedStyle,
		SelectedCount:     selectedCountStyle,
		TabActive:         tabActiveStyle,
		TabInactive:       tabInactiveStyle,
	}

	m := Model{
		state:           StateBrowsing,
		config:          cfg,
		adapter:         adptr,
		installer:       installer,
		servers:         servers,
		cursor:          0,
		envValues:       make(map[string]string),
		scope:           scope,
		multiSelectMode: multiSelect,
		multiSelect:     make(map[string]bool),
		filteredServers: servers, // Initially show all
		allInstalled:    allInstalled,

		// Initialize state handlers
		browsingHandler:    states.NewBrowsingHandler(stylesConfig),
		detailsHandler:     states.NewDetailsHandler(stylesConfig),
		configuringHandler: states.NewConfiguringHandler(stylesConfig),
		installingHandler:  states.NewInstallingHandler(stylesConfig),
	}

	// Initialize presets for multi-select mode
	if multiSelect && cfg.Presets != nil {
		allPresets := cfg.ListPresets()
		// Filter presets where at least one server is available (uninstalled)
		for _, pName := range allPresets {
			preset := cfg.Presets[pName]
			if preset == nil {
				continue
			}
			hasAvailable := false
			for _, srvName := range preset.Servers {
				if !adptr.ServerExists(srvName, scope) {
					hasAvailable = true
					break
				}
			}
			if hasAvailable {
				m.presets = append(m.presets, pName)
			}
		}
		m.filteredPresets = m.presets
	}

	return m
}

// updateFilter filters the server list based on filterText
func (m *Model) updateFilter() {
	result := FilterItems(m.servers, m.filterText, m.cursor, func(name string, lower string) bool {
		server := m.config.Servers[name]
		return strings.Contains(strings.ToLower(name), lower) ||
			(server != nil && strings.Contains(strings.ToLower(server.Description), lower))
	})
	m.filteredServers = result.Items
	m.cursor = result.Cursor
}

// updatePresetFilter filters the preset list based on filterText
func (m *Model) updatePresetFilter() {
	result := FilterItems(m.presets, m.filterText, m.presetCursor, func(name string, lower string) bool {
		preset := m.config.Presets[name]
		return strings.Contains(strings.ToLower(name), lower) ||
			(preset != nil && strings.Contains(strings.ToLower(preset.Description), lower))
	})
	m.filteredPresets = result.Items
	m.presetCursor = result.Cursor
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
	params := states.BrowsingUpdateParams{
		Msg:             msg,
		Cursor:          m.cursor,
		PresetCursor:    m.presetCursor,
		Filtering:       m.filtering,
		FilterText:      m.filterText,
		MultiSelectMode: m.multiSelectMode,
		ActiveTab:       m.activeTab,
		Servers:         m.servers,
		FilteredServers: m.filteredServers,
		FilteredPresets: m.filteredPresets,
		MultiSelect:     m.multiSelect,
		Config:  m.config,
		Adapter: m.adapter,
		Scope:   m.scope,
	}

	result := m.browsingHandler.Update(params, m.updateFilter, m.updatePresetFilter)

	// Update model state from result
	m.cursor = result.Cursor
	m.presetCursor = result.PresetCursor
	m.filtering = result.Filtering
	m.filterText = result.FilterText
	m.activeTab = result.ActiveTab
	m.filteredServers = result.FilteredServers
	m.filteredPresets = result.FilteredPresets
	m.multiSelect = result.MultiSelect

	if result.ShouldQuit {
		return m, tea.Quit
	}

	switch result.TransitionTo {
	case "details":
		m.selected = result.Selected
		m.state = StateDetails
	case "bulk_configure":
		bulkModel := NewBulkConfigureModelWithInstaller(result.SelectedServers, m.config, m.adapter, m.installer, m.scope)
		return bulkModel, bulkModel.Init()
	}

	return m, nil
}

// updateDetails handles input in the details state
func (m Model) updateDetails(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	params := states.DetailsUpdateParams{
		Msg:      msg,
		Selected: m.selected,
		Config:   m.config,
	}

	result := m.detailsHandler.Update(params)

	if result.ShouldQuit {
		return m, tea.Quit
	}

	switch result.TransitionTo {
	case "browsing":
		m.state = StateBrowsing
		m.selected = ""
	case "configuring":
		m.envKeys = result.EnvKeys
		m.currentField = result.CurrentField
		m.textInput = result.TextInput
		m.cursorPos = result.CursorPos
		m.state = StateConfiguring
	case "installing":
		m.state = StateInstalling
		return m, m.installServer()
	}

	return m, nil
}

// updateConfiguring handles input in the configuring state
func (m Model) updateConfiguring(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	params := states.ConfiguringUpdateParams{
		Msg:          msg,
		Selected:     m.selected,
		EnvKeys:      m.envKeys,
		EnvValues:    m.envValues,
		CurrentField: m.currentField,
		TextInput:    m.textInput,
		CursorPos:    m.cursorPos,
	}

	result := m.configuringHandler.Update(params)

	// Update model state from result
	m.envValues = result.EnvValues
	m.currentField = result.CurrentField
	m.textInput = result.TextInput
	m.cursorPos = result.CursorPos

	if result.ShouldQuit {
		return m, tea.Quit
	}

	switch result.TransitionTo {
	case "details":
		m.state = StateDetails
	case "installing":
		m.state = StateInstalling
		return m, m.installServer()
	}

	return m, nil
}

// updateInstalling handles input in the installing state
func (m Model) updateInstalling(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	params := states.InstallingUpdateParams{
		Msg: msg,
	}

	result := m.installingHandler.UpdateInstalling(params)

	if result.ShouldQuit {
		return m, tea.Quit
	}

	return m, nil
}

// updateComplete handles input in the complete state
func (m Model) updateComplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	params := states.CompleteUpdateParams{
		Msg: msg,
	}

	result := m.installingHandler.UpdateComplete(params)

	if result.ShouldQuit {
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
	params := states.BrowsingViewParams{
		MultiSelectMode: m.multiSelectMode,
		ActiveTab:       m.activeTab,
		Filtering:       m.filtering,
		FilterText:      m.filterText,
		FilteredServers: m.filteredServers,
		FilteredPresets: m.filteredPresets,
		Servers:         m.servers,
		Cursor:          m.cursor,
		PresetCursor:    m.presetCursor,
		MultiSelect:     m.multiSelect,
		Config:          m.config,
		Width:           m.width,
		Height:          m.height,
		AllInstalled:    m.allInstalled,
	}

	return m.browsingHandler.View(params)
}

// viewDetails renders the details state
func (m Model) viewDetails() string {
	params := states.DetailsViewParams{
		Selected: m.selected,
		Config:   m.config,
	}

	return m.detailsHandler.View(params)
}

// viewConfiguring renders the configuring state
func (m Model) viewConfiguring() string {
	params := states.ConfiguringViewParams{
		Selected:     m.selected,
		Config:       m.config,
		EnvKeys:      m.envKeys,
		EnvValues:    m.envValues,
		CurrentField: m.currentField,
		TextInput:    m.textInput,
		CursorPos:    m.cursorPos,
	}

	return m.configuringHandler.View(params)
}

// viewInstalling renders the installing state
func (m Model) viewInstalling() string {
	params := states.InstallingViewParams{
		Selected:    m.selected,
		Scope:       m.scope,
		AdapterName: m.adapter.Name(),
	}

	return m.installingHandler.ViewInstalling(params)
}

// viewComplete renders the complete state
func (m Model) viewComplete() string {
	params := states.CompleteViewParams{
		Selected:   m.selected,
		Scope:      m.scope,
		Err:        m.err,
		InstallMsg: m.installMsg,
	}

	return m.installingHandler.ViewComplete(params)
}

// installServer creates a command that installs the server in the background
func (m Model) installServer() tea.Cmd {
	return func() tea.Msg {
		req := app.ServerInstallRequest{
			ServerName: m.selected,
			Scope:      m.scope,
			EnvValues:  m.envValues,
		}

		resp := m.installer.Install(context.Background(), req)
		return installCompleteMsg{err: resp.Error}
	}
}
