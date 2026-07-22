package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7c3aed")).
			Padding(0, 1).
			Render

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7c3aed"))
)

var searchBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#7c3aed")).
	Padding(0, 1)

var helpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#6b7280"))

var activeStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#22c55e")).
	Bold(true)

var inactiveStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#6b7280"))

var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#ef4444")).
	Padding(1, 2)

var highlightStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#f59e0b")).
	Bold(true)

var successStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#22c55e"))

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#000")).
		Background(lipgloss.Color("#a78bfa")).
		Bold(false)
	return s
}
