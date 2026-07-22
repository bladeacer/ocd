package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err))
	}

	switch m.state {
	case stateLoading:
		return m.loadingView()
	case stateTable:
		return m.tableView()
	case stateConfirm:
		return m.confirmView()
	}
	return ""
}

func (m *model) loadingView() string {
	s := fmt.Sprintf("\n%s Loading Obsidian version data...\n", m.spinner.View())
	return appStyle.Render(s)
}

func (m *model) tableView() string {
	if !m.ready {
		return m.loadingView()
	}

	var b strings.Builder

	b.WriteString(headerStyle.Render("obsi-css-diff"))
	b.WriteString("\n")

	b.WriteString(filterStyle.Render(
		fmt.Sprintf("[m] Mobile:%v [e] Early:%v [f] Docker:%v [s] Priority:%v  /search  enter:select q:quit",
			formatBool(m.showMobile),
			formatBool(m.showEarlyAccess),
			formatBool(m.foundOnly),
			formatBool(m.sortByPriority),
		),
	))
	b.WriteString("\n")

	t := m.tbl

	rendered := lipgloss.NewStyle().MaxWidth(m.width - 4).Render(t.View())
	b.WriteString(rendered)
	b.WriteString("\n")

	b.WriteString(helpStyle.Render("m: toggle mobile | e: toggle early access | f: toggle docker-found | s: toggle sort | q: quit"))

	if m.selectedVersion != "" {
		b.WriteString(fmt.Sprintf("\n\n%s Selected: v%s", successStyle.Render(">"), m.selectedVersion))
	}

	return appStyle.Render(b.String())
}

func (m *model) confirmView() string {
	content := fmt.Sprintf(
		"%s\n\nVersion: %s\n\n%s   %s",
		labelStyle.Render("Extract app.css for this version?"),
		versionStyle.Render(m.selectedVersion),
		successStyle.Render("[Y] Yes"),
		errorStyle.Render("[N] No"),
	)
	return appStyle.Render(confirmBoxStyle.Render(content))
}

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("205"))

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("60"))

	return s
}

func formatBool(v bool) string {
	if v {
		return greenDot
	}
	return dimDot
}

const (
	greenDot = "\033[32m●\033[0m"
	dimDot   = "\033[2m○\033[0m"
)

func (m *model) Run() (Selection, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return Selection{}, err
	}

	m2, ok := finalModel.(*model)
	if !ok {
		return Selection{}, nil
	}

	if m2.state == stateConfirm {
		return Selection{
			Version: m2.selectedVersion,
		}, nil
	}

	return Selection{}, nil
}
