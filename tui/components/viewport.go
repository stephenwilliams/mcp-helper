package components

// ViewportManager handles scrolling/viewport for lists
type ViewportManager struct {
	Start    int // First visible row
	Height   int // Terminal height
	Reserved int // Lines reserved for header/footer
	MinHeight int // Minimum viewport height
}

// NewViewportManager creates a new ViewportManager with default settings
func NewViewportManager(reserved, minHeight int) *ViewportManager {
	return &ViewportManager{
		Start:     0,
		Reserved:  reserved,
		MinHeight: minHeight,
	}
}

// GetViewportHeight returns available rows for content
func (v *ViewportManager) GetViewportHeight() int {
	if v.Height <= v.Reserved {
		return v.MinHeight
	}
	return v.Height - v.Reserved
}

// AdjustForCursor ensures cursor is visible, returns new start
func (v *ViewportManager) AdjustForCursor(cursor, totalItems int) int {
	viewportHeight := v.GetViewportHeight()

	// Scroll up if cursor is above viewport
	if cursor < v.Start {
		v.Start = cursor
	}

	// Scroll down if cursor is below viewport
	if cursor >= v.Start+viewportHeight {
		v.Start = cursor - viewportHeight + 1
	}

	// Ensure start is not negative
	if v.Start < 0 {
		v.Start = 0
	}

	return v.Start
}

// GetVisibleRange returns start/end indices for rendering
func (v *ViewportManager) GetVisibleRange(cursor, totalItems int) (start, end int) {
	viewportHeight := v.GetViewportHeight()
	v.AdjustForCursor(cursor, totalItems)

	start = v.Start
	end = v.Start + viewportHeight

	// Clamp to total items
	if end > totalItems {
		end = totalItems
	}

	return start, end
}

// SetTerminalHeight updates height from WindowSizeMsg
func (v *ViewportManager) SetTerminalHeight(height int) {
	v.Height = height
}
