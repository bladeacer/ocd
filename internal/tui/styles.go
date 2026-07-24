package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#6d28d9", Dark: "#7c3aed"}).
			Padding(0, 1).
			Render

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6d28d9", Dark: "#7c3aed"})
)

var searchBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#6d28d9", Dark: "#7c3aed"}).
	Padding(0, 1)

var helpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#9ca3af", Dark: "#6b7280"})

var activeStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#15803d", Dark: "#22c55e"}).
	Bold(true)

var inactiveStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#9ca3af", Dark: "#6b7280"})

var errorStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#ef4444"}).
	Padding(1, 2)

var highlightStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#f59e0b"}).
	Bold(true)

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#9ca3af", Dark: "#374151"}).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#000")).
		Background(lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#a78bfa"}).
		Bold(false)
	return s
}
