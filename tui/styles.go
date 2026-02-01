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

	// Selected item style
	selectedStyle = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true).
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
)
