// Package tuiharness provides a headless test harness for Bubble Tea TUI applications.
// It uses teatest for program control and vt for terminal emulation.
package tuiharness

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/charmbracelet/x/vt"
)

// DefaultCols is the default terminal width.
const DefaultCols = 120

// DefaultRows is the default terminal height.
const DefaultRows = 40

// Options configures the test harness.
type Options struct {
	Cols           int           // Terminal width (default 120)
	Rows           int           // Terminal height (default 40)
	WaitTimeout    time.Duration // Max time to wait for screen changes (default 2s)
	PollInterval   time.Duration // Interval for polling screen changes (default 50ms)
	DebugStateFunc func() string // Optional function to get model debug state
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		Cols:         DefaultCols,
		Rows:         DefaultRows,
		WaitTimeout:  2 * time.Second,
		PollInterval: 50 * time.Millisecond,
	}
}

// Screen represents a snapshot of the terminal screen.
type Screen struct {
	Rows       int      // Number of rows
	Cols       int      // Number of columns
	CursorX    int      // Cursor X position (0-indexed)
	CursorY    int      // Cursor Y position (0-indexed)
	GridText   string   // Plain text content of the screen
	GridLines  []string // Individual lines
	RawANSI    string   // Raw ANSI output (if captured)
	DebugState string   // Optional debug state from model
	hash       string   // Cached hash
}

// Hash returns a stable hash of the screen state.
func (s *Screen) Hash() string {
	if s.hash != "" {
		return s.hash
	}
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d:%d:%d:%d:", s.Rows, s.Cols, s.CursorX, s.CursorY)))
	h.Write([]byte(s.GridText))
	if s.DebugState != "" {
		h.Write([]byte(s.DebugState))
	}
	s.hash = hex.EncodeToString(h.Sum(nil))[:16]
	return s.hash
}

// Excerpt returns a short excerpt of the screen for logging.
func (s *Screen) Excerpt(maxLines int) string {
	if maxLines <= 0 {
		maxLines = 5
	}
	lines := s.GridLines
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

// ContainsText checks if the screen contains the given text.
func (s *Screen) ContainsText(text string) bool {
	return strings.Contains(s.GridText, text)
}

// Harness controls a Bubble Tea program for testing.
type Harness struct {
	opts    Options
	tm      *teatest.TestModel
	emu     *vt.Emulator
	mu      sync.Mutex
	stopped bool

	// For capturing raw output
	rawOutput strings.Builder

	// Debug state function (optional)
	debugStateFunc func() string
}

// ModelFactory is a function that creates a new tea.Model for testing.
type ModelFactory func() tea.Model

// Start launches the TUI under test with the given model factory.
func Start(factory ModelFactory, opts ...Options) (*Harness, error) {
	opt := DefaultOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Cols <= 0 {
		opt.Cols = DefaultCols
	}
	if opt.Rows <= 0 {
		opt.Rows = DefaultRows
	}
	if opt.WaitTimeout <= 0 {
		opt.WaitTimeout = 2 * time.Second
	}
	if opt.PollInterval <= 0 {
		opt.PollInterval = 50 * time.Millisecond
	}

	h := &Harness{
		opts:           opt,
		debugStateFunc: opt.DebugStateFunc,
	}

	// Create the virtual terminal emulator
	h.emu = vt.NewEmulator(opt.Cols, opt.Rows)

	// Create the model
	model := factory()

	// Create the test model with teatest
	h.tm = teatest.NewTestModel(
		nil, // no *testing.T needed for our use case
		model,
		teatest.WithInitialTermSize(opt.Cols, opt.Rows),
	)

	return h, nil
}

// Press sends key presses to the TUI.
func (h *Harness) Press(keys ...Key) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return fmt.Errorf("harness already stopped")
	}

	for _, key := range keys {
		h.tm.Send(key.ToTeaKey())
	}

	return nil
}

// Type sends text input to the TUI character by character.
func (h *Harness) Type(text string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return fmt.Errorf("harness already stopped")
	}

	for _, r := range text {
		h.tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	return nil
}

// Resize simulates a terminal resize event.
func (h *Harness) Resize(cols, rows int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return fmt.Errorf("harness already stopped")
	}

	h.opts.Cols = cols
	h.opts.Rows = rows

	// Resize the emulator
	h.emu.Resize(cols, rows)

	// Send resize message
	h.tm.Send(tea.WindowSizeMsg{Width: cols, Height: rows})

	return nil
}

// Screen captures the current screen state.
func (h *Harness) Screen() (*Screen, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return nil, fmt.Errorf("harness already stopped")
	}

	// Get output from the test model
	outputReader := h.tm.Output()

	// Read all available output
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, outputReader)
	output := buf.String()

	// Write to emulator
	_, _ = h.emu.WriteString(output)

	// Get cursor position
	cursorPos := h.emu.CursorPosition()

	// Extract screen content
	screen := &Screen{
		Rows:    h.opts.Rows,
		Cols:    h.opts.Cols,
		CursorX: cursorPos.X,
		CursorY: cursorPos.Y,
		RawANSI: output,
	}

	// Use the emulator's String() method to get the rendered text
	rendered := h.emu.String()

	// Split into lines
	lines := strings.Split(rendered, "\n")
	screen.GridLines = lines
	screen.GridText = rendered

	// Get debug state if available
	if h.debugStateFunc != nil {
		screen.DebugState = h.debugStateFunc()
	}

	return screen, nil
}

// WaitForScreenChange waits for the screen to change from the given hash.
func (h *Harness) WaitForScreenChange(prevHash string) (*Screen, error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.opts.WaitTimeout)
	defer cancel()

	ticker := time.NewTicker(h.opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Return current screen even on timeout
			screen, _ := h.Screen()
			return screen, fmt.Errorf("timeout waiting for screen change")
		case <-ticker.C:
			screen, err := h.Screen()
			if err != nil {
				return nil, err
			}
			if screen.Hash() != prevHash {
				return screen, nil
			}
		}
	}
}

// WaitForText waits for specific text to appear on screen.
func (h *Harness) WaitForText(text string) (*Screen, error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.opts.WaitTimeout)
	defer cancel()

	ticker := time.NewTicker(h.opts.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			screen, _ := h.Screen()
			return screen, fmt.Errorf("timeout waiting for text: %q", text)
		case <-ticker.C:
			screen, err := h.Screen()
			if err != nil {
				return nil, err
			}
			if screen.ContainsText(text) {
				return screen, nil
			}
		}
	}
}

// Stop terminates the TUI and cleans up resources.
func (h *Harness) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return nil
	}

	h.stopped = true

	// Send quit
	h.tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Wait for the program to finish with a timeout
	done := make(chan struct{})
	go func() {
		h.tm.WaitFinished(nil, teatest.WithFinalTimeout(2*time.Second))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		// Force close if needed
	}

	// Close emulator
	if h.emu != nil {
		_ = h.emu.Close()
	}

	return nil
}

// IsRunning returns true if the harness is still running.
func (h *Harness) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return !h.stopped
}

// SendMsg sends a custom tea.Msg to the program.
func (h *Harness) SendMsg(msg tea.Msg) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return fmt.Errorf("harness already stopped")
	}

	h.tm.Send(msg)
	return nil
}

// FinalModel returns the final model state after the program exits.
// This blocks until the program terminates.
func (h *Harness) FinalModel() tea.Model {
	return h.tm.FinalModel(nil, teatest.WithFinalTimeout(5*time.Second))
}

// DrainOutput reads and discards any pending output.
func (h *Harness) DrainOutput() {
	h.mu.Lock()
	defer h.mu.Unlock()
	outputReader := h.tm.Output()
	_, _ = io.Copy(io.Discard, outputReader)
}
