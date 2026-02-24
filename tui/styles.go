package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Color scheme
	colorCyan    = lipgloss.Color("86")
	colorGray    = lipgloss.Color("240")
	colorWhite   = lipgloss.Color("255")
	colorGreen   = lipgloss.Color("42")
	colorRed     = lipgloss.Color("196")
	colorYellow  = lipgloss.Color("226")

	// Title style
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan).
			MarginBottom(1)

	// Selected item style - uses background highlight to distinguish from transport badges
	selectedStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true).
			Background(lipgloss.Color("236")).
			PaddingLeft(2)

	// Normal item style
	normalStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			PaddingLeft(4)

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(colorGray).
			MarginTop(1)

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true).
			MarginTop(1)

	// Success style
	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true).
			MarginTop(1)

	// Info style
	infoStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			MarginTop(1)

	// Detail label style
	labelStyle = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	// Detail value style
	valueStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	// Transport badge styles
	transportStdioStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	transportHTTPStyle = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	transportUnknownStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	// Description styles
	descriptionStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				PaddingLeft(4)

	descriptionDimStyle = lipgloss.NewStyle().
				Foreground(colorGray).
				PaddingLeft(4)

	// Checkbox styles for multi-select
	checkboxCheckedStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	checkboxUncheckedStyle = lipgloss.NewStyle().
				Foreground(colorGray)

	selectedCountStyle = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	bulkProgressStyle = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	resultSuccessStyle = lipgloss.NewStyle().
				Foreground(colorGreen)

	resultFailureStyle = lipgloss.NewStyle().
				Foreground(colorRed)

	// Tab styles for preset/server tabs
	tabActiveStyle = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true).
				Background(lipgloss.Color("236"))

	tabInactiveStyle = lipgloss.NewStyle().
				Foreground(colorGray)
)
