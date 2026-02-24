package components

import (
	"strings"

	"github.com/stephenwilliams/mcp-helper/internal/env"
)

// RenderInputWithCursor renders a text input with visible cursor.
// If the key indicates a secret field, the text is masked with asterisks.
func RenderInputWithCursor(text string, cursorPos int, key string, cursorStyle func(...string) string) string {
	if env.IsSecret(key) {
		masked := strings.Repeat("*", len(text))
		if cursorPos < len(masked) {
			return masked[:cursorPos] + cursorStyle("_") + masked[cursorPos:]
		}
		return masked + cursorStyle("_")
	}
	if cursorPos < len(text) {
		return text[:cursorPos] + cursorStyle(string(text[cursorPos])) + text[cursorPos+1:]
	}
	return text + cursorStyle("_")
}
