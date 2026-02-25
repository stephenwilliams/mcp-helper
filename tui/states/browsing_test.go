package states

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stephenwilliams/mcp-helper/internal/config"
)

// testStyles returns a minimal Styles struct for testing
func testStyles() Styles {
	return Styles{
		Title:             lipgloss.NewStyle(),
		Selected:          lipgloss.NewStyle(),
		Normal:            lipgloss.NewStyle(),
		Help:              lipgloss.NewStyle(),
		Error:             lipgloss.NewStyle(),
		Success:           lipgloss.NewStyle(),
		Info:              lipgloss.NewStyle(),
		Label:             lipgloss.NewStyle(),
		Value:             lipgloss.NewStyle(),
		TransportStdio:    lipgloss.NewStyle(),
		TransportHTTP:     lipgloss.NewStyle(),
		TransportUnknown:  lipgloss.NewStyle(),
		CheckboxChecked:   lipgloss.NewStyle(),
		CheckboxUnchecked: lipgloss.NewStyle(),
		SelectedCount:     lipgloss.NewStyle(),
		TabActive:         lipgloss.NewStyle(),
		TabInactive:       lipgloss.NewStyle(),
	}
}

// TestBrowsingView_ViewportCalculation tests that the viewport correctly calculates
// visible items based on terminal height, accounting for 2 lines per item (name + description).
func TestBrowsingView_ViewportCalculation(t *testing.T) {
	handler := NewBrowsingHandler(testStyles())

	tests := []struct {
		name            string
		height          int
		serverCount     int
		multiSelectMode bool
		cursor          int
		wantStart       int
		wantEnd         int
	}{
		{
			name:            "small window single-select shows limited items",
			height:          20,
			serverCount:     20,
			multiSelectMode: false,
			cursor:          0,
			// 20 - 9 reserved = 11 available lines, 11/2 = 5 visible items
			wantStart: 0,
			wantEnd:   5,
		},
		{
			name:            "small window multi-select shows fewer items due to extra reserved lines",
			height:          20,
			serverCount:     20,
			multiSelectMode: true,
			cursor:          0,
			// 20 - 11 reserved = 9 available lines, 9/2 = 4 visible items
			wantStart: 0,
			wantEnd:   4,
		},
		{
			name:            "cursor centering in middle of list",
			height:          30,
			serverCount:     20,
			multiSelectMode: false,
			cursor:          10,
			// 30 - 9 = 21 available, 21/2 = 10 visible items
			// start = cursor - visibleItems/2 = 10 - 5 = 5
			wantStart: 5,
			wantEnd:   15,
		},
		{
			name:            "cursor near end adjusts start",
			height:          30,
			serverCount:     20,
			multiSelectMode: false,
			cursor:          18,
			// 30 - 9 = 21 available, 21/2 = 10 visible items
			// start would be 18 - 5 = 13, end = 23, but list only has 20
			// so end = 20, start = 20 - 10 = 10
			wantStart: 10,
			wantEnd:   20,
		},
		{
			name:            "very small window uses minimum",
			height:          5,
			serverCount:     20,
			multiSelectMode: false,
			cursor:          0,
			// 5 - 9 = -4, clamped to minimum 3 available lines, 3/2 = 1 visible item
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:            "fewer items than visible space shows all",
			height:          50,
			serverCount:     3,
			multiSelectMode: false,
			cursor:          1,
			// All 3 items fit
			wantStart: 0,
			wantEnd:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test servers
			servers := make(map[string]*config.Server)
			serverList := make([]string, tt.serverCount)
			for i := 0; i < tt.serverCount; i++ {
				name := string(rune('a'+i%26)) + string(rune('0'+i/26))
				serverList[i] = name
				servers[name] = &config.Server{
					Description: "Description for " + name,
					Transport:   "stdio",
				}
			}

			params := BrowsingViewParams{
				Config:          &config.Config{Servers: servers},
				Servers:         serverList,
				FilteredServers: serverList,
				Cursor:          tt.cursor,
				MultiSelectMode: tt.multiSelectMode,
				MultiSelect:     make(map[string]bool),
				Width:           80,
				Height:          tt.height,
			}

			output := handler.View(params)

			// Count how many server entries are rendered
			// Each server line contains the transport badge [stdio]
			renderedCount := strings.Count(output, "[stdio]")

			expectedCount := tt.wantEnd - tt.wantStart
			if renderedCount != expectedCount {
				t.Errorf("expected %d items rendered (start=%d, end=%d), got %d",
					expectedCount, tt.wantStart, tt.wantEnd, renderedCount)
			}

			// Verify scroll indicator appears when list is scrollable
			if tt.serverCount > expectedCount {
				if !strings.Contains(output, "Showing") {
					t.Error("expected scroll indicator 'Showing X-Y of Z' when list is scrollable")
				}
			}
		})
	}
}

// TestBrowsingView_SmallWindowKeepsHeaderVisible verifies that the header
// remains visible even when the terminal is very small.
func TestBrowsingView_SmallWindowKeepsHeaderVisible(t *testing.T) {
	handler := NewBrowsingHandler(testStyles())

	servers := map[string]*config.Server{
		"server-1": {Description: "First server", Transport: "stdio"},
		"server-2": {Description: "Second server", Transport: "stdio"},
		"server-3": {Description: "Third server", Transport: "stdio"},
	}

	params := BrowsingViewParams{
		Config:          &config.Config{Servers: servers},
		Servers:         []string{"server-1", "server-2", "server-3"},
		FilteredServers: []string{"server-1", "server-2", "server-3"},
		Cursor:          0,
		MultiSelectMode: true,
		MultiSelect:     make(map[string]bool),
		Width:           80,
		Height:          15, // Very small window
	}

	output := handler.View(params)
	lines := strings.Split(output, "\n")

	// First line should be the title
	if len(lines) == 0 || !strings.Contains(lines[0], "Select MCP Servers") {
		t.Error("expected title 'Select MCP Servers' to be visible at top")
	}
}

// TestBrowsingView_DescriptionsTakeSpace verifies that descriptions are
// properly accounted for in viewport calculation.
func TestBrowsingView_DescriptionsTakeSpace(t *testing.T) {
	handler := NewBrowsingHandler(testStyles())

	// Create 10 servers with descriptions
	servers := make(map[string]*config.Server)
	serverList := make([]string, 10)
	for i := 0; i < 10; i++ {
		name := string(rune('a' + i))
		serverList[i] = name
		servers[name] = &config.Server{
			Description: "Description for server " + name,
			Transport:   "stdio",
		}
	}

	params := BrowsingViewParams{
		Config:          &config.Config{Servers: servers},
		Servers:         serverList,
		FilteredServers: serverList,
		Cursor:          0,
		MultiSelectMode: false,
		MultiSelect:     make(map[string]bool),
		Width:           80,
		Height:          25, // Limited height
	}

	output := handler.View(params)

	// Count server names and descriptions
	serverLines := strings.Count(output, "[stdio]")
	descriptionLines := strings.Count(output, "Description for server")

	// Each visible server should have a corresponding description
	if serverLines != descriptionLines {
		t.Errorf("expected equal server names (%d) and descriptions (%d)",
			serverLines, descriptionLines)
	}

	// With height 25, reserved 9, available 16, visibleItems = 8
	// So we should see at most 8 servers
	if serverLines > 8 {
		t.Errorf("expected at most 8 servers visible, got %d", serverLines)
	}
}

// TestBrowsingView_MultiSelectReservesMoreSpace verifies that multi-select mode
// correctly reserves more space for additional UI elements.
func TestBrowsingView_MultiSelectReservesMoreSpace(t *testing.T) {
	handler := NewBrowsingHandler(testStyles())

	servers := make(map[string]*config.Server)
	serverList := make([]string, 20)
	for i := 0; i < 20; i++ {
		name := string(rune('a'+i%26)) + string(rune('0'+i/26))
		serverList[i] = name
		servers[name] = &config.Server{
			Description: "Description",
			Transport:   "stdio",
		}
	}

	cfg := &config.Config{Servers: servers}

	// Single-select mode
	singleParams := BrowsingViewParams{
		Config:          cfg,
		Servers:         serverList,
		FilteredServers: serverList,
		Cursor:          0,
		MultiSelectMode: false,
		MultiSelect:     make(map[string]bool),
		Width:           80,
		Height:          30,
	}

	// Multi-select mode
	multiParams := BrowsingViewParams{
		Config:          cfg,
		Servers:         serverList,
		FilteredServers: serverList,
		Cursor:          0,
		MultiSelectMode: true,
		MultiSelect:     make(map[string]bool),
		Width:           80,
		Height:          30,
	}

	singleOutput := handler.View(singleParams)
	multiOutput := handler.View(multiParams)

	singleCount := strings.Count(singleOutput, "[stdio]")
	multiCount := strings.Count(multiOutput, "[stdio]")

	// Multi-select should show fewer items due to additional reserved lines
	if multiCount >= singleCount {
		t.Errorf("multi-select mode should show fewer items than single-select: multi=%d, single=%d",
			multiCount, singleCount)
	}
}
