package tui

import (
	"strings"
	"testing"
)

func TestFilterItems_EmptyQuery(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	result := FilterItems(items, "", 2, func(s, lower string) bool {
		return strings.Contains(s, lower)
	})
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
	if result.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", result.Cursor)
	}
}

func TestFilterItems_MatchSome(t *testing.T) {
	items := []string{"alpha", "beta", "alphabet"}
	result := FilterItems(items, "alph", 0, func(s, lower string) bool {
		return strings.Contains(strings.ToLower(s), lower)
	})
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
}

func TestFilterItems_MatchNone(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	result := FilterItems(items, "zzz", 1, func(s, lower string) bool {
		return strings.Contains(strings.ToLower(s), lower)
	})
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
	if result.Cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", result.Cursor)
	}
}

func TestFilterItems_CursorOutOfBounds(t *testing.T) {
	items := []string{"alpha", "beta", "gamma"}
	// cursor=5, only 1 item matches => cursor reset to 0
	result := FilterItems(items, "alph", 5, func(s, lower string) bool {
		return strings.Contains(strings.ToLower(s), lower)
	})
	if result.Cursor != 0 {
		t.Errorf("expected cursor reset to 0, got %d", result.Cursor)
	}
}

func TestFilterItems_CursorWithinBounds(t *testing.T) {
	items := []string{"alpha", "alphabet", "beta"}
	// cursor=1, 2 items match => cursor stays at 1
	result := FilterItems(items, "alph", 1, func(s, lower string) bool {
		return strings.Contains(strings.ToLower(s), lower)
	})
	if result.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", result.Cursor)
	}
}
