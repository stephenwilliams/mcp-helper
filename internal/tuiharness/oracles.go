package tuiharness

import (
	"regexp"
	"strings"
	"unicode"
)

// OracleResult represents the result of an oracle check.
type OracleResult struct {
	Failed      bool
	OracleName  string
	Description string
	Details     string
}

// Oracle is a function that checks for bugs in the screen state.
type Oracle func(screen *Screen, history []*Screen) *OracleResult

// CrashOracle detects if the program has crashed or panicked.
func CrashOracle(exitCode int, stderr string) *OracleResult {
	if exitCode != 0 {
		return &OracleResult{
			Failed:      true,
			OracleName:  "crash",
			Description: "Program exited with non-zero status",
			Details:     stderr,
		}
	}

	// Check for panic in output
	if strings.Contains(stderr, "panic:") || strings.Contains(stderr, "runtime error:") {
		return &OracleResult{
			Failed:      true,
			OracleName:  "panic",
			Description: "Program panicked",
			Details:     stderr,
		}
	}

	return nil
}

// HangOracle detects if the program is hung (no screen change after input).
type HangOracle struct {
	unchangedCount int
	threshold      int
}

// NewHangOracle creates a hang oracle with the given threshold.
func NewHangOracle(threshold int) *HangOracle {
	if threshold <= 0 {
		threshold = 5
	}
	return &HangOracle{threshold: threshold}
}

// Check checks if the screen has been unchanged too long.
func (h *HangOracle) Check(screen *Screen, prevScreen *Screen) *OracleResult {
	if prevScreen == nil {
		h.unchangedCount = 0
		return nil
	}

	if screen.Hash() == prevScreen.Hash() {
		h.unchangedCount++
	} else {
		h.unchangedCount = 0
	}

	if h.unchangedCount >= h.threshold {
		return &OracleResult{
			Failed:      true,
			OracleName:  "hang",
			Description: "Screen unchanged after multiple inputs",
			Details:     "No change after " + string(rune('0'+h.unchangedCount)) + " actions",
		}
	}

	return nil
}

// Reset resets the hang counter.
func (h *HangOracle) Reset() {
	h.unchangedCount = 0
}

// CorruptionOracle detects screen corruption (excessive control characters or broken rendering).
func CorruptionOracle(screen *Screen, history []*Screen) *OracleResult {
	// Skip check on initial/empty screens - give TUI time to render
	trimmed := strings.TrimSpace(screen.GridText)
	if len(trimmed) < 10 {
		return nil // Screen too empty to judge
	}

	// Count control characters and unprintables (excluding common whitespace and null)
	controlCount := 0
	printableCount := 0

	for _, r := range screen.GridText {
		// Skip common whitespace
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' || r == 0 {
			continue
		}

		if unicode.IsPrint(r) {
			printableCount++
		} else if r < 32 || r == 127 {
			// True control characters (excluding already skipped whitespace)
			controlCount++
		}
	}

	// Check for excessive control characters (more than 10% of printable content is suspicious)
	if printableCount > 0 && float64(controlCount)/float64(printableCount) > 0.10 {
		return &OracleResult{
			Failed:      true,
			OracleName:  "corruption",
			Description: "Screen contains excessive control characters",
			Details:     "Control chars: " + intToStr(controlCount) + "/" + intToStr(printableCount),
		}
	}

	// Check for ANSI escape sequence remnants in plain text
	if strings.Contains(screen.GridText, "\x1b[") {
		return &OracleResult{
			Failed:      true,
			OracleName:  "corruption",
			Description: "Unprocessed ANSI escape sequences in output",
			Details:     "Raw escape sequences visible",
		}
	}

	return nil
}

// FocusMarkerOracle detects when focus/selection markers disappear unexpectedly.
type FocusMarkerOracle struct {
	markers       []string
	markerRegexes []*regexp.Regexp
	sawMarker     bool
	missingCount  int
	threshold     int
}

// NewFocusMarkerOracle creates a focus marker oracle.
func NewFocusMarkerOracle() *FocusMarkerOracle {
	return &FocusMarkerOracle{
		markers:   []string{"►", ">", "•", "[x]", "[X]", "[✓]", "[ ]", "→"},
		threshold: 3,
	}
}

// Check checks if focus markers are present when expected.
func (f *FocusMarkerOracle) Check(screen *Screen, _ []*Screen) *OracleResult {
	hasMarker := false
	for _, marker := range f.markers {
		if strings.Contains(screen.GridText, marker) {
			hasMarker = true
			f.sawMarker = true
			break
		}
	}

	// If we've seen markers before but now they're gone, that's suspicious
	if f.sawMarker && !hasMarker {
		f.missingCount++
		if f.missingCount >= f.threshold {
			return &OracleResult{
				Failed:      true,
				OracleName:  "focus_lost",
				Description: "Focus/selection marker disappeared",
				Details:     "Marker missing for " + intToStr(f.missingCount) + " consecutive screens",
			}
		}
	} else {
		f.missingCount = 0
	}

	return nil
}

// Reset resets the oracle state.
func (f *FocusMarkerOracle) Reset() {
	f.sawMarker = false
	f.missingCount = 0
}

// SelectionBoundsOracle detects when selection index appears out of bounds.
type SelectionBoundsOracle struct {
	numberPattern *regexp.Regexp
}

// NewSelectionBoundsOracle creates a selection bounds oracle.
func NewSelectionBoundsOracle() *SelectionBoundsOracle {
	return &SelectionBoundsOracle{
		// Match patterns like "1/10", "5 of 20", "Showing 1-10 of 50"
		numberPattern: regexp.MustCompile(`(\d+)\s*(?:/|of)\s*(\d+)`),
	}
}

// Check checks if selection indices are valid.
func (s *SelectionBoundsOracle) Check(screen *Screen, _ []*Screen) *OracleResult {
	matches := s.numberPattern.FindAllStringSubmatch(screen.GridText, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			current := parseSimpleInt(match[1])
			total := parseSimpleInt(match[2])

			if total > 0 && current > total {
				return &OracleResult{
					Failed:      true,
					OracleName:  "bounds",
					Description: "Selection index out of bounds",
					Details:     match[0] + " - current > total",
				}
			}

			if current < 0 {
				return &OracleResult{
					Failed:      true,
					OracleName:  "bounds",
					Description: "Negative selection index",
					Details:     match[0],
				}
			}
		}
	}

	return nil
}

// EmptyScreenOracle detects completely empty screens (likely rendering failure).
func EmptyScreenOracle(screen *Screen, history []*Screen) *OracleResult {
	// Skip initial screen
	if len(history) == 0 {
		return nil
	}

	trimmed := strings.TrimSpace(screen.GridText)
	if len(trimmed) == 0 {
		return &OracleResult{
			Failed:      true,
			OracleName:  "empty_screen",
			Description: "Screen is completely empty",
			Details:     "No visible content rendered",
		}
	}

	return nil
}

// ErrorMessageOracle detects error messages in the UI.
func ErrorMessageOracle(screen *Screen, _ []*Screen) *OracleResult {
	errorPatterns := []string{
		"error:", "Error:", "ERROR:",
		"panic:", "PANIC:",
		"fatal:", "Fatal:", "FATAL:",
		"failed:", "Failed:", "FAILED:",
		"exception:", "Exception:",
	}

	for _, pattern := range errorPatterns {
		if strings.Contains(screen.GridText, pattern) {
			// Find the line containing the error
			for _, line := range screen.GridLines {
				if strings.Contains(line, pattern) {
					return &OracleResult{
						Failed:      true,
						OracleName:  "error_message",
						Description: "Error message detected in UI",
						Details:     strings.TrimSpace(line),
					}
				}
			}
		}
	}

	return nil
}

// CompositeOracle runs multiple oracles and returns the first failure.
type CompositeOracle struct {
	oracles []Oracle
}

// NewCompositeOracle creates a composite oracle from multiple oracles.
func NewCompositeOracle(oracles ...Oracle) *CompositeOracle {
	return &CompositeOracle{oracles: oracles}
}

// Check runs all oracles and returns the first failure.
func (c *CompositeOracle) Check(screen *Screen, history []*Screen) *OracleResult {
	for _, oracle := range c.oracles {
		if result := oracle(screen, history); result != nil && result.Failed {
			return result
		}
	}
	return nil
}

// DefaultOracles returns the default set of oracles for testing.
func DefaultOracles() *CompositeOracle {
	focusOracle := NewFocusMarkerOracle()
	boundsOracle := NewSelectionBoundsOracle()

	return NewCompositeOracle(
		CorruptionOracle,
		focusOracle.Check,
		boundsOracle.Check,
		EmptyScreenOracle,
		// Note: ErrorMessageOracle is intentionally not included by default
		// as some apps legitimately show error messages
	)
}

// Helper functions

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func parseSimpleInt(s string) int {
	result := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		}
	}
	return result
}
