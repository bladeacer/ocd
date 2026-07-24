package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var helpBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#a78bfa"}).
	Padding(1, 2).
	Foreground(lipgloss.AdaptiveColor{Light: "#1f2937", Dark: "#e5e7eb"})

func (m *model) View() string {
	switch m.state {
	case stateLoading:
		return m.loadingView()
	case stateTable, stateSearch:
		return m.tableContentView()
	case stateConfirm:
		return m.confirmView()
	}
	return ""
}

func (m *model) loadingView() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error loading data:\n\n%s", m.err))
	}

	elapsed := time.Since(m.startTime).Round(time.Second)
	msg := m.loadMessages[m.loadIndex]

	return fmt.Sprintf("\n  %s %s\n\n  elapsed: %s\n", m.spinner.View(), msg, elapsed)
}

func (m *model) tableContentView() string {
	if m.showHelp {
		return m.renderHelpCentered()
	}

	var b strings.Builder

	b.WriteString(titleStyle("ocd -- Obsidian CSS Diff"))

	if m.state == stateSearch {
		b.WriteString("\n" + searchBoxStyle.Render(m.searchIn.View()))
	}

	b.WriteString("\n\n")
	b.WriteString(m.tbl.View())

	b.WriteString(m.footerView())

	return b.String()
}

func (m *model) renderHelpCentered() string {
	helpContent := []string{
		"  ocd -- Help",
		"",
		"  ↑ ↓      Navigate rows",
		"  ← →      Scroll columns",
		"  enter    Select version (extract CSS)",
		"  /        Search/filter versions",
		"  m        Toggle mobile versions",
		"  e        Toggle early access / insider versions",
		"  f        Show only versions with Docker images",
		"  s        Toggle sort priority",
		"  q        Quit",
		"  ? / Esc  Close this help",
	}
	helpText := strings.Join(helpContent, "\n")
	box := helpBorderStyle.Render(helpText)
	width := m.width
	if width < 80 {
		width = 80
	}
	pad := (width - lipgloss.Width(box)) / 2
	if pad < 2 {
		pad = 2
	}
	padded := lipgloss.NewStyle().PaddingLeft(pad).Render(box)
	return "\n\n\n" + padded
}

func (m *model) footerView() string {
	parts := []string{
		fmtStatus("M", m.showMobile),
		fmtStatus("E", m.showEarlyAccess),
		fmtStatus("F", m.foundOnly),
		fmtStatus("S", m.sortByPriority),
	}

	keys := helpStyle.Render("up/down/left/right nav  / search  enter select  m toggle mobile  e toggle early  f toggle docker  s toggle sort  q quit  ? help")

	info := fmt.Sprintf("[%s]", strings.Join(parts, " "))
	return "\n\n" + info + "\n" + keys
}

func fmtStatus(label string, active bool) string {
	if active {
		return activeStyle.Render(strings.ToUpper(label))
	}
	return inactiveStyle.Render(strings.ToUpper(label))
}

func (m *model) confirmView() string {
	v := m.selectedVersion
	msg := fmt.Sprintf(
		"Extract CSS for version %s?\n\n  This downloads and extracts app.css from the Obsidian %s release.\n\n  [y] Yes   [n] No",
		highlightStyle.Render(v),
		v,
	)
	return lipgloss.NewStyle().Padding(1, 2).Render(msg)
}
