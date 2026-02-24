package tuiharness

import tea "github.com/charmbracelet/bubbletea"

// Key represents a key press that can be sent to the TUI.
type Key struct {
	Type  tea.KeyType
	Runes []rune
	Alt   bool
}

// ToTeaKey converts the Key to a tea.KeyMsg.
func (k Key) ToTeaKey() tea.KeyMsg {
	return tea.KeyMsg{
		Type:  k.Type,
		Runes: k.Runes,
		Alt:   k.Alt,
	}
}

// String returns a human-readable representation of the key.
func (k Key) String() string {
	switch k.Type {
	case tea.KeyUp:
		return "Up"
	case tea.KeyDown:
		return "Down"
	case tea.KeyLeft:
		return "Left"
	case tea.KeyRight:
		return "Right"
	case tea.KeyEnter:
		return "Enter"
	case tea.KeyEsc:
		return "Esc"
	case tea.KeyTab:
		return "Tab"
	case tea.KeyShiftTab:
		return "Shift+Tab"
	case tea.KeyBackspace:
		return "Backspace"
	case tea.KeySpace:
		return "Space"
	case tea.KeyCtrlC:
		return "Ctrl+C"
	case tea.KeyCtrlR:
		return "Ctrl+R"
	case tea.KeyCtrlA:
		return "Ctrl+A"
	case tea.KeyCtrlE:
		return "Ctrl+E"
	case tea.KeyPgUp:
		return "PgUp"
	case tea.KeyPgDown:
		return "PgDn"
	case tea.KeyHome:
		return "Home"
	case tea.KeyEnd:
		return "End"
	case tea.KeyRunes:
		if len(k.Runes) > 0 {
			return string(k.Runes)
		}
		return "Runes"
	default:
		return k.Type.String()
	}
}

// Common key definitions for easy use.
var (
	KeyUp        = Key{Type: tea.KeyUp}
	KeyDown      = Key{Type: tea.KeyDown}
	KeyLeft      = Key{Type: tea.KeyLeft}
	KeyRight     = Key{Type: tea.KeyRight}
	KeyEnter     = Key{Type: tea.KeyEnter}
	KeyEsc       = Key{Type: tea.KeyEsc}
	KeyTab       = Key{Type: tea.KeyTab}
	KeyShiftTab  = Key{Type: tea.KeyShiftTab}
	KeyBackspace = Key{Type: tea.KeyBackspace}
	KeySpace     = Key{Type: tea.KeySpace}
	KeyCtrlC     = Key{Type: tea.KeyCtrlC}
	KeyCtrlR     = Key{Type: tea.KeyCtrlR}
	KeyCtrlA     = Key{Type: tea.KeyCtrlA}
	KeyCtrlE     = Key{Type: tea.KeyCtrlE}
	KeyPgUp      = Key{Type: tea.KeyPgUp}
	KeyPgDown    = Key{Type: tea.KeyPgDown}
	KeyHome      = Key{Type: tea.KeyHome}
	KeyEnd       = Key{Type: tea.KeyEnd}
)

// Rune creates a Key for a single character.
func Rune(r rune) Key {
	return Key{Type: tea.KeyRunes, Runes: []rune{r}}
}

// Runes creates a Key for multiple characters.
func Runes(s string) Key {
	return Key{Type: tea.KeyRunes, Runes: []rune(s)}
}

// ActionType represents the type of action in the exploration space.
type ActionType int

const (
	ActionKeyPress ActionType = iota
	ActionType_
	ActionResize
)

// Action represents an action that can be performed on the TUI.
type Action struct {
	Type   ActionType
	Key    Key
	Text   string
	Width  int
	Height int
}

// String returns a human-readable representation of the action.
func (a Action) String() string {
	switch a.Type {
	case ActionKeyPress:
		return "Press(" + a.Key.String() + ")"
	case ActionType_:
		return "Type(" + a.Text + ")"
	case ActionResize:
		return "Resize(" + string(rune('0'+a.Width/100)) + string(rune('0'+(a.Width/10)%10)) + string(rune('0'+a.Width%10)) + "x" + string(rune('0'+a.Height/10)) + string(rune('0'+a.Height%10)) + ")"
	default:
		return "Unknown"
	}
}

// ToJSON returns a JSON-serializable representation.
func (a Action) ToJSON() map[string]any {
	m := map[string]any{
		"type": a.Type,
	}
	switch a.Type {
	case ActionKeyPress:
		m["key"] = a.Key.String()
	case ActionType_:
		m["text"] = a.Text
	case ActionResize:
		m["width"] = a.Width
		m["height"] = a.Height
	}
	return m
}

// PressAction creates a key press action.
func PressAction(key Key) Action {
	return Action{Type: ActionKeyPress, Key: key}
}

// TypeAction creates a text typing action.
func TypeAction(text string) Action {
	return Action{Type: ActionType_, Text: text}
}

// ResizeAction creates a resize action.
func ResizeAction(width, height int) Action {
	return Action{Type: ActionResize, Width: width, Height: height}
}

// DefaultActionSpace returns the default set of actions for exploration.
func DefaultActionSpace() []Action {
	return []Action{
		// Navigation keys
		PressAction(KeyUp),
		PressAction(KeyDown),
		PressAction(KeyLeft),
		PressAction(KeyRight),
		PressAction(KeyPgUp),
		PressAction(KeyPgDown),
		PressAction(KeyHome),
		PressAction(KeyEnd),
		// Tab navigation
		PressAction(KeyTab),
		PressAction(KeyShiftTab),
		// Selection/confirmation
		PressAction(KeyEnter),
		PressAction(KeySpace),
		// Cancel/escape
		PressAction(KeyEsc),
		PressAction(KeyBackspace),
		// Control keys
		PressAction(KeyCtrlC),
		PressAction(KeyCtrlR),
		// Text input (search/filter)
		PressAction(Rune('/')),
		// Common resize events
		ResizeAction(80, 24),
		ResizeAction(100, 30),
		ResizeAction(120, 40),
	}
}

// RandomTextActions returns a set of text typing actions for exploration.
func RandomTextActions() []Action {
	texts := []string{
		"a", "test", "search", "abc", "123",
	}
	actions := make([]Action, len(texts))
	for i, t := range texts {
		actions[i] = TypeAction(t)
	}
	return actions
}
