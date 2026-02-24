package tui

import "strings"

// FilterResult holds the filtered items and adjusted cursor position
type FilterResult[T any] struct {
	Items  []T
	Cursor int
}

// FilterItems filters a slice based on a query string.
// matchFn should return true if the item matches the query.
// cursor is adjusted if it goes out of bounds.
func FilterItems[T any](items []T, query string, cursor int, matchFn func(T, string) bool) FilterResult[T] {
	if query == "" {
		return FilterResult[T]{Items: items, Cursor: cursor}
	}
	lower := strings.ToLower(query)
	var filtered []T
	for _, item := range items {
		if matchFn(item, lower) {
			filtered = append(filtered, item)
		}
	}
	// Adjust cursor if out of bounds
	if cursor >= len(filtered) {
		cursor = 0
	}
	return FilterResult[T]{Items: filtered, Cursor: cursor}
}
