<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-01 | Updated: 2026-02-01 -->

# tui

## Purpose

Interactive terminal user interface for browsing and installing MCP servers. Built with Bubbletea (Elm architecture) and styled with Lipgloss. Provides a state-machine driven UI with browsing, details, configuration, and installation states.

## Key Files

| File | Description |
|------|-------------|
| `app.go` | Entry point `Run()` function that starts the Bubbletea program |
| `model.go` | TUI model with state machine, Update/View methods, and all UI states |
| `styles.go` | Lipgloss style definitions for colors, badges, and text formatting |

## For AI Agents

### Working In This Directory

- Follow Bubbletea's Elm architecture: Model, Update, View
- State is managed via `State` enum (StateBrowsing, StateDetails, etc.)
- Each state has its own `update*` and `view*` methods
- Messages are typed (e.g., `installCompleteMsg`)

### Bubbletea Pattern

```go
// Model holds all UI state
type Model struct {
    state    State
    cursor   int
    // ...
}

// Update handles messages and returns new model + optional command
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle key presses based on current state
    case tea.WindowSizeMsg:
        // Handle terminal resize
    case installCompleteMsg:
        // Handle async operation completion
    }
    return m, nil
}

// View renders current state to string
func (m Model) View() string {
    switch m.state {
    case StateBrowsing:
        return m.viewBrowsing()
    // ...
    }
}
```

### State Machine

```
StateBrowsing → StateDetails → StateConfiguring → StateInstalling → StateComplete
      ↑              ↓
      └──── (esc) ───┘
```

### Testing Requirements

- Test state transitions with mock models
- Test view rendering for each state
- Test key handling in each state

### Common Patterns

- Use `tea.Cmd` for async operations (installation)
- Styles defined as package-level variables in `styles.go`
- Secret fields detected by name patterns (TOKEN, KEY, SECRET, etc.)
- Scrolling for long server lists

## Dependencies

### Internal

- `internal/config` - Server configuration types
- `internal/adapter` - Adapter interface for installation

### External

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
