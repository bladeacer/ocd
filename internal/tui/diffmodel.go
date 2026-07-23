package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bladeacer/ocd/internal/models"
)

type lineKind int

const (
	lineContext lineKind = iota
	lineAdd
	lineDel
	lineHunkHeader
	lineFileHeader
	lineEmpty
)

type parsedLine struct {
	text       string
	kind       lineKind
	oldLineNum int
	newLineNum int
}

type diffModel struct {
	result *models.DiffResult
	vp     viewport.Model
	ready  bool

	parsed  []parsedLine
	hunkIdx []int

	currentHunk int
	insertions  int
	deletions   int

	sideBySide bool

	searchMode bool
	searchIn   textinput.Model
	searchQ    string

	pendingG bool

	summaryStyle lipgloss.Style
	hintStyle    lipgloss.Style
	content      string
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

	if !m.result.HasDiff {
		m.content = fmt.Sprintf("Diff: %s -> %s\n\nNo differences found.", m.result.VersionA, m.result.VersionB)
		return
	}

	raw := strings.Split(m.result.Diff, "\n")
	m.parsed = nil
	m.hunkIdx = nil
	m.insertions = 0
	m.deletions = 0
	oldStart, newStart := 0, 0
	inHunk := false

	for _, line := range raw {
		if strings.HasPrefix(line, "@@") {
			oldStart, newStart = parseHunkHeader(line)
			inHunk = true
			m.hunkIdx = append(m.hunkIdx, len(m.parsed))
			m.parsed = append(m.parsed, parsedLine{text: line, kind: lineHunkHeader})
			continue
		}

		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			m.parsed = append(m.parsed, parsedLine{text: line, kind: lineFileHeader})
			continue
		}

		if !inHunk || line == "" {
			m.parsed = append(m.parsed, parsedLine{text: line, kind: lineEmpty})
			continue
		}

		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			m.insertions++
			m.parsed = append(m.parsed, parsedLine{
				text: line, kind: lineAdd,
				oldLineNum: 0, newLineNum: newStart,
			})
			newStart++

		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			m.deletions++
			m.parsed = append(m.parsed, parsedLine{
				text: line, kind: lineDel,
				oldLineNum: oldStart, newLineNum: 0,
			})
			oldStart++

		default:
			m.parsed = append(m.parsed, parsedLine{
				text: line, kind: lineContext,
				oldLineNum: oldStart, newLineNum: newStart,
			})
			oldStart++
			newStart++
		}
	}

	m.render()
}

func parseHunkHeader(h string) (oldStart, newStart int) {
	oldStart, newStart = 1, 1
	rest := h
	if idx := strings.Index(rest, "@@"); idx >= 0 {
		rest = rest[idx+2:]
	}
	if idx := strings.Index(rest, "@@"); idx >= 0 {
		rest = strings.TrimSpace(rest[:idx])
	}
	parts := strings.Fields(rest)
	for _, p := range parts {
		if strings.HasPrefix(p, "-") {
			v := parseLeadingNum(p[1:])
			if v > 0 {
				oldStart = v
			}
		}
		if strings.HasPrefix(p, "+") {
			v := parseLeadingNum(p[1:])
			if v > 0 {
				newStart = v
			}
		}
	}
	return
}

func parseLeadingNum(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

func (m *diffModel) render() {
	header := fmt.Sprintf("Diff: %s -> %s", m.result.VersionA, m.result.VersionB)
	summary := m.summaryStyle.Render(
		fmt.Sprintf(" 1 file changed, %d insertion(s)(+), %d deletion(s)(-)", m.insertions, m.deletions),
	)

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(summary)
	b.WriteString("\n\n")

	if m.sideBySide {
		m.renderSideBySide(&b)
	} else {
		m.renderUnified(&b)
	}

	m.content = b.String()
}

func (m *diffModel) renderUnified(b *strings.Builder) {
	hl := lipgloss.NewStyle().Background(lipgloss.Color("#854d0e"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))

	for _, pl := range m.parsed {
		line := m.formatLine(pl, &lineNumStyle)
		if m.searchQ != "" {
			plain := stripANSI(line)
			if strings.Contains(strings.ToLower(plain), strings.ToLower(m.searchQ)) {
				b.WriteString(hl.Render(plain))
				b.WriteString("\n")
				continue
			}
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func (m *diffModel) renderSideBySide(b *strings.Builder) {
	half := (m.vp.Width - 3) / 2
	if half < 20 {
		half = 20
	}
	hl := lipgloss.NewStyle().Background(lipgloss.Color("#854d0e"))

	groups := m.groupSideBySide()

	for _, g := range groups {
		oldContent := g.oldContent
		newContent := g.newContent
		oldNum := ""
		newNum := ""
		if g.oldLineNum > 0 {
			oldNum = strconv.Itoa(g.oldLineNum)
		}
		if g.newLineNum > 0 {
			newNum = strconv.Itoa(g.newLineNum)
		}

		var oldStyled, newStyled string

		switch g.kind {
		case lineHunkHeader:
			styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")).Bold(true).Render(g.oldContent)
			b.WriteString(styled)
			b.WriteString("\n")
			continue
		case lineFileHeader:
			styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1")).Render(g.oldContent)
			b.WriteString(styled)
			b.WriteString("\n")
			continue
		case lineDel:
			oldStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(
				truncate(oldContent, half*2),
			)
		case lineAdd:
			newStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Render(
				truncate(newContent, half*2),
			)
		default:
			oldStyled = truncate(oldContent, half*2)
			newStyled = truncate(newContent, half*2)
		}

		left := fmt.Sprintf("%s %s", padRight(oldNum, 4), oldStyled)
		right := fmt.Sprintf("%s %s", padRight(newNum, 4), newStyled)

		left = truncate(left, half)
		right = truncate(right, half)

		line := left + " | " + right

		if m.searchQ != "" {
			plain := stripANSI(line)
			if strings.Contains(strings.ToLower(plain), strings.ToLower(m.searchQ)) {
				line = hl.Render(plain)
			}
		}

		b.WriteString(line)
		b.WriteString("\n")
	}
}

type sbGroup struct {
	oldContent string
	newContent string
	oldLineNum int
	newLineNum int
	kind       lineKind
}

func (m *diffModel) groupSideBySide() []sbGroup {
	var groups []sbGroup
	for _, pl := range m.parsed {
		switch pl.kind {
		case lineContext:
			groups = append(groups, sbGroup{
				oldContent: pl.text,
				newContent: pl.text,
				oldLineNum: pl.oldLineNum,
				newLineNum: pl.newLineNum,
				kind:       lineContext,
			})
		case lineDel:
			groups = append(groups, sbGroup{
				oldContent: pl.text,
				newContent: "",
				oldLineNum: pl.oldLineNum,
				newLineNum: 0,
				kind:       lineDel,
			})
		case lineAdd:
			groups = append(groups, sbGroup{
				oldContent: "",
				newContent: pl.text,
				oldLineNum: 0,
				newLineNum: pl.newLineNum,
				kind:       lineAdd,
			})
		case lineHunkHeader:
			groups = append(groups, sbGroup{
				oldContent: pl.text,
				newContent: pl.text,
				kind:       lineHunkHeader,
			})
		case lineFileHeader:
			groups = append(groups, sbGroup{
				oldContent: pl.text,
				newContent: pl.text,
				kind:       lineFileHeader,
			})
		default:
			groups = append(groups, sbGroup{
				oldContent: pl.text,
				newContent: pl.text,
				kind:       lineEmpty,
			})
		}
	}
	return groups
}

func (m *diffModel) formatLine(pl parsedLine, numStyle *lipgloss.Style) string {
	oldNum := ""
	newNum := ""
	if pl.oldLineNum > 0 {
		oldNum = fmt.Sprintf("%d", pl.oldLineNum)
	}
	if pl.newLineNum > 0 {
		newNum = fmt.Sprintf("%d", pl.newLineNum)
	}
	nums := fmt.Sprintf("%4s %4s", oldNum, newNum)

	switch pl.kind {
	case lineHunkHeader:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")).Bold(true).Render(
			fmt.Sprintf("  %s", pl.text),
		)
	case lineAdd:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Render(
			fmt.Sprintf("%s |%s", nums, pl.text),
		)
	case lineDel:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(
			fmt.Sprintf("%s |%s", nums, pl.text),
		)
	case lineFileHeader:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1")).Render(
			fmt.Sprintf("  %s", pl.text),
		)
	default:
		return fmt.Sprintf("%s | %s", numStyle.Render(nums), pl.text)
	}
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n > 3 {
		return string(runes[:n-3]) + "..."
	}
	return string(runes[:n])
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
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
	m.render()
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

const (
	keyEscape    = "esc"
	keyEnter     = "enter"
	keyCtrlC     = "ctrl+c"
	keyTab       = "tab"
	keyBackspace = "backspace"
)

func (m *diffModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEscape, keyCtrlC:
		m.searchMode = false
		m.searchQ = ""
		m.searchIn.Blur()
		m.refreshViewport()
		return m, nil

	case keyEnter, keyTab:
		m.searchMode = false
		m.searchQ = m.searchIn.Value()
		m.searchIn.Blur()
		m.scrollToMatch()
		m.refreshViewport()
		return m, nil

	case keyBackspace:
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
	if m.searchQ == "" || len(m.parsed) == 0 {
		return
	}
	q := strings.ToLower(m.searchQ)
	m.vp.GotoTop()
	for i, pl := range m.parsed {
		if strings.Contains(strings.ToLower(pl.text), q) {
			for j := 0; j < i; j++ {
				m.vp.LineDown(1)
			}
			return
		}
	}
}

func (m *diffModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", keyCtrlC, keyEscape:
		return m, tea.Quit

	case "n":
		m.nextHunk()
		return m, nil
	case "N":
		m.prevHunk()
		return m, nil

	case "v":
		m.sideBySide = !m.sideBySide
		m.render()
		m.vp.SetContent(m.content)
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

func (m *diffModel) yankText(start, end int) string {
	var lines []string
	for i := start; i < end && i < len(m.parsed); i++ {
		lines = append(lines, m.parsed[i].text)
	}
	return strings.Join(lines, "\n")
}

func (m *diffModel) yankHunk() tea.Cmd {
	return func() tea.Msg {
		if len(m.hunkIdx) == 0 {
			return nil
		}
		start := m.hunkIdx[m.currentHunk]
		end := len(m.parsed)
		if m.currentHunk+1 < len(m.hunkIdx) {
			end = m.hunkIdx[m.currentHunk+1]
		}
		text := m.yankText(start, end)
		if err := clipboard.WriteAll(text); err != nil {
			return nil
		}
		return nil
	}
}

func (m *diffModel) yankAll() tea.Cmd {
	return func() tea.Msg {
		text := m.yankText(0, len(m.parsed))
		if err := clipboard.WriteAll(text); err != nil {
			return nil
		}
		return nil
	}
}

func (m *diffModel) openInEditor() tea.Cmd {
	f, err := os.CreateTemp("", "ocd-diff-*.txt")
	if err != nil {
		return nil
	}
	tmpPath := f.Name()
	if _, err := f.WriteString(m.result.Diff); err != nil {
		f.Close()
		return nil
	}
	f.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, tmpPath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		os.Remove(tmpPath)
		return nil
	})
}

func (m *diffModel) nextHunk() {
	if len(m.hunkIdx) == 0 {
		return
	}
	m.currentHunk = (m.currentHunk + 1) % len(m.hunkIdx)
	m.vp.GotoTop()
	for i := 0; i < m.hunkIdx[m.currentHunk]; i++ {
		m.vp.LineDown(1)
	}
}

func (m *diffModel) prevHunk() {
	if len(m.hunkIdx) == 0 {
		return
	}
	m.currentHunk--
	if m.currentHunk < 0 {
		m.currentHunk = len(m.hunkIdx) - 1
	}
	m.vp.GotoTop()
	for i := 0; i < m.hunkIdx[m.currentHunk]; i++ {
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
	if len(m.hunkIdx) > 0 {
		hunkInfo = m.hintStyle.Render(
			fmt.Sprintf("hunk %d/%d", m.currentHunk+1, len(m.hunkIdx)),
		)
	}

	mode := "unified"
	if m.sideBySide {
		mode = "side-by-side"
	}

	footer := m.hintStyle.Render(
		fmt.Sprintf("\n%s  n/N hunk  v %s  gg/G top/bot  / search  y hunk  Y all  o edit  q quit",
			hunkInfo, mode,
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
