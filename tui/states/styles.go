package states

import "github.com/charmbracelet/lipgloss"

// Styles holds all the lipgloss styles used by state handlers
type Styles struct {
	Title             lipgloss.Style
	Selected          lipgloss.Style
	Normal            lipgloss.Style
	Help              lipgloss.Style
	Error             lipgloss.Style
	Success           lipgloss.Style
	Info              lipgloss.Style
	Label             lipgloss.Style
	Value             lipgloss.Style
	TransportStdio    lipgloss.Style
	TransportHTTP     lipgloss.Style
	TransportUnknown  lipgloss.Style
	CheckboxChecked   lipgloss.Style
	CheckboxUnchecked lipgloss.Style
	SelectedCount     lipgloss.Style
	TabActive         lipgloss.Style
	TabInactive       lipgloss.Style
}
