package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bladeacer/ocd/internal/models"
)

type diffModel struct {
	result *models.DiffResult
	vp     viewport.Model
	ready  bool

	rawLines    []string
	styledLines []string
	content     string

	hunkLines   []int
	currentHunk int

	insertions int
	deletions  int

	searchMode bool
	searchIn   textinput.Model
	searchQ    string

	pendingG bool

	summaryStyle lipgloss.Style
	hintStyle    lipgloss.Style
}

func NewDiffModel(result *models.DiffResult) *diffModel {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.CharLimit = 80
	ti.Width = 40

	return &diffModel{
		result:       result,
		searchIn:     ti,
		summaryStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")),
		hintStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")),
	}
}

func (m *diffModel) Init() tea.Cmd {
	return nil
}

func (m *diffModel) build() {
	if m.content != "" {
		return
	}

	if m.result.Error != nil {
		m.content = fmt.Sprintf("Error: %v", m.result.Error)
		return
	}

	header := fmt.Sprintf("Diff: %s \u2192 %s", m.result.VersionA, m.result.VersionB)
	diff := m.result.Diff

	if !m.result.HasDiff {
		m.content = header + "\n\nNo differences found."
		return
	}

	m.rawLines = strings.Split(diff, "\n")
	m.styledLines = make([]string, len(m.rawLines))
	m.insertions = 0
	m.deletions = 0

	for i, line := range m.rawLines {
		var styled string
		switch {
		case strings.HasPrefix(line, "@@"):
			m.hunkLines = append(m.hunkLines, i)
			styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")).Bold(true).Render(
				fmt.Sprintf("\u2502 %s", line),
			)
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			m.insertions++
			styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Render(
				fmt.Sprintf("\u2502 %s", line),
			)
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			m.deletions++
			styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(
				fmt.Sprintf("\u2502 %s", line),
			)
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
			styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1")).Render(line)
		default:
			styled = fmt.Sprintf("  %s", line)
		}
		m.styledLines[i] = styled
	}

	m.renderContent()
}

func (m *diffModel) renderContent() {
	header := fmt.Sprintf("Diff: %s \u2192 %s", m.result.VersionA, m.result.VersionB)
	summary := m.summaryStyle.Render(
		fmt.Sprintf("+%d -%d", m.insertions, m.deletions),
	)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")

	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("#854d0e"))

	for _, line := range m.styledLines {
		if m.searchQ != "" {
			plain := stripANSI(line)
			if strings.Contains(strings.ToLower(plain), strings.ToLower(m.searchQ)) {
				b.WriteString(highlightStyle.Render(plain))
				b.WriteString("\n")
				continue
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(summary)

	m.content = b.String()
}

func stripANSI(s string) string {
	var out strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}

func (m *diffModel) refreshViewport() {
	m.renderContent()
	m.vp.SetContent(m.content)
}

func (m *diffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.vp.Width = msg.Width - 2
		m.vp.Height = msg.Height - 6
		m.ready = true
		m.build()
		m.vp.SetContent(m.content)
		return m, nil

	case tea.KeyMsg:
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		return m.handleNormalKey(msg)
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *diffModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "escape", "ctrl+c":
		m.searchMode = false
		m.searchQ = ""
		m.searchIn.Blur()
		m.refreshViewport()
		return m, nil

	case "enter", "tab":
		m.searchMode = false
		m.searchQ = m.searchIn.Value()
		m.searchIn.Blur()
		m.scrollToMatch()
		m.refreshViewport()
		return m, nil

	case "backspace":
		val := m.searchIn.Value()
		if len(val) > 0 {
			m.searchIn.SetValue(val[:len(val)-1])
		}
		m.searchQ = m.searchIn.Value()
		m.refreshViewport()
		return m, nil
	}

	if len(msg.String()) == 1 {
		m.searchIn, _ = m.searchIn.Update(msg)
		m.searchQ = m.searchIn.Value()
		m.refreshViewport()
		return m, nil
	}

	return m, nil
}

func (m *diffModel) scrollToMatch() {
	if m.searchQ == "" || len(m.rawLines) == 0 {
		return
	}
	q := strings.ToLower(m.searchQ)
	m.vp.GotoTop()
	for i, line := range m.rawLines {
		if strings.Contains(strings.ToLower(line), q) {
			for j := 0; j < i; j++ {
				m.vp.LineDown(1)
			}
			return
		}
	}
}

func (m *diffModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "escape":
		return m, tea.Quit

	case "n":
		m.nextHunk()
		return m, nil
	case "N":
		m.prevHunk()
		return m, nil

	case "g":
		if m.pendingG {
			m.pendingG = false
			m.vp.GotoTop()
		} else {
			m.pendingG = true
		}
		return m, nil
	case "G":
		m.pendingG = false
		m.vp.GotoBottom()
		return m, nil

	case "/":
		m.searchMode = true
		m.searchIn.Focus()
		m.searchIn.SetValue("")
		m.searchQ = ""
		m.refreshViewport()
		return m, nil

	case "y":
		m.pendingG = false
		return m, m.yankHunk()

	case "Y":
		m.pendingG = false
		return m, m.yankAll()

	case "o":
		m.pendingG = false
		return m, m.openInEditor()
	}

	m.pendingG = false

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *diffModel) yankHunk() tea.Cmd {
	return func() tea.Msg {
		if len(m.hunkLines) == 0 {
			return nil
		}
		start := m.hunkLines[m.currentHunk]
		end := len(m.rawLines)
		if m.currentHunk+1 < len(m.hunkLines) {
			end = m.hunkLines[m.currentHunk+1]
		}
		text := strings.Join(m.rawLines[start:end], "\n")
		if err := clipboard.WriteAll(text); err != nil {
			return nil
		}
		return nil
	}
}

func (m *diffModel) yankAll() tea.Cmd {
	return func() tea.Msg {
		text := strings.Join(m.rawLines, "\n")
		if err := clipboard.WriteAll(text); err != nil {
			return nil
		}
		return nil
	}
}

func (m *diffModel) openInEditor() tea.Cmd {
	return func() tea.Msg {
		f, err := os.CreateTemp("", "ocd-diff-*.txt")
		if err != nil {
			return nil
		}
		tmpPath := f.Name()
		f.WriteString(m.result.Diff)
		f.Close()
		defer os.Remove(tmpPath)

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command(editor, tmpPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
		return nil
	}
}

func (m *diffModel) nextHunk() {
	if len(m.hunkLines) == 0 {
		return
	}
	m.currentHunk = (m.currentHunk + 1) % len(m.hunkLines)
	m.vp.GotoTop()
	for i := 0; i < m.hunkLines[m.currentHunk]; i++ {
		m.vp.LineDown(1)
	}
}

func (m *diffModel) prevHunk() {
	if len(m.hunkLines) == 0 {
		return
	}
	m.currentHunk--
	if m.currentHunk < 0 {
		m.currentHunk = len(m.hunkLines) - 1
	}
	m.vp.GotoTop()
	for i := 0; i < m.hunkLines[m.currentHunk]; i++ {
		m.vp.LineDown(1)
	}
}

func (m *diffModel) View() string {
	if !m.ready {
		return "\n  Loading diff view..."
	}
	if m.content == "" {
		m.build()
		m.vp.SetContent(m.content)
	}

	var searchBar string
	if m.searchMode {
		searchBar = "\n" + searchBoxStyle.Render(m.searchIn.View())
	}

	hunkInfo := ""
	if len(m.hunkLines) > 0 {
		hunkInfo = m.hintStyle.Render(
			fmt.Sprintf("hunk %d/%d", m.currentHunk+1, len(m.hunkLines)),
		)
	}

	footer := m.hintStyle.Render(
		fmt.Sprintf("\n%s  n/N hunk  gg/G top/bot  / search  y hunk  Y all  o edit  q quit",
			hunkInfo,
		),
	)

	return m.vp.View() + searchBar + footer
}

func RunDiffViewer(result *models.DiffResult) error {
	m := NewDiffModel(result)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
