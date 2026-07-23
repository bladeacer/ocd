package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"

	"github.com/bladeacer/ocd/internal/core"
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
	pendingZ bool
	pendingY bool
	count    int

	searchMatchIdxs []int
	searchMatchCurr int
	showHelp        bool

	tldrResult *core.TLDRResult

	summaryStyle lipgloss.Style
	hintStyle    lipgloss.Style
	content      string
}

var (
	cssCommentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280")).Italic(true)
	cssAtRuleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")).Bold(true)
	cssSelectorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#60a5fa"))
	cssPropStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24"))
	cssPunctStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ca3af"))
)

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
		m.content = ""
		return
	}

	raw := strings.Split(m.result.Diff, "\n")
	m.parsed = nil
	m.hunkIdx = nil
	m.searchMatchIdxs = nil
	m.searchMatchCurr = 0
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

func (m *diffModel) renderHeader() string {
	header := fmt.Sprintf("Diff: %s -> %s", m.result.VersionA, m.result.VersionB)
	summary := m.summaryStyle.Render(
		fmt.Sprintf(" 1 file changed, %d insertion(s)(+), %d deletion(s)(-)", m.insertions, m.deletions),
	)
	s := header + "\n" + summary
	if m.sideBySide {
		s += m.renderColumnHeaders()
	}
	return s
}

func (m *diffModel) renderColumnHeaders() string {
	half := (m.vp.Width - 3) / 2
	if half < 20 {
		half = 20
	}
	oldLabel := fmt.Sprintf("Old (%s)", m.result.VersionA)
	newLabel := fmt.Sprintf("New (%s)", m.result.VersionB)
	if m.result.VersionA == "" {
		oldLabel = "Old"
	}
	if m.result.VersionB == "" {
		newLabel = "New"
	}
	leftHdr := lipgloss.NewStyle().Width(half - 2).Bold(true).Foreground(lipgloss.Color("#a78bfa")).Render(oldLabel)
	rightHdr := lipgloss.NewStyle().Width(half - 2).Bold(true).Foreground(lipgloss.Color("#a78bfa")).Render(newLabel)
	div := lipgloss.NewStyle().Width(3).Render(" │ ")
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(
		strings.Repeat("─", half-2) + "─┼─" + strings.Repeat("─", half-2),
	)
	return "\n" + lipgloss.JoinHorizontal(lipgloss.Top, leftHdr, div, rightHdr) + "\n" + sep
}

func (m *diffModel) render() {
	var b strings.Builder
	if m.sideBySide {
		m.renderSideBySide(&b)
	} else {
		m.renderUnified(&b)
	}
	m.content = b.String()
}

func highlightCSSLine(raw string) string {
	line := raw
	trimmed := strings.TrimSpace(line)

	if trimmed == "" {
		return line
	}

	if strings.HasPrefix(trimmed, "/*") {
		return cssCommentStyle.Render(line)
	}

	if strings.HasPrefix(trimmed, "@") {
		parts := strings.SplitN(trimmed, " ", 2)
		return cssAtRuleStyle.Render(parts[0])
	}

	if trimmed == "}" || trimmed == "{" {
		return cssPunctStyle.Render(line)
	}

	colonIdx := strings.Index(trimmed, ":")
	if colonIdx > 0 && !strings.HasPrefix(trimmed, ".") && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "*") {
		prop := trimmed[:colonIdx]

		if !strings.Contains(prop, " ") {
			propIdx := strings.Index(line, prop)
			result := line[:propIdx] + cssPropStyle.Render(prop) + line[propIdx+len(prop):]
			return result
		}
	}

	if strings.HasSuffix(trimmed, "{") || (!strings.Contains(trimmed, ":") && !strings.Contains(trimmed, ";")) {
		return cssSelectorStyle.Render(line)
	}

	return line
}

func highlightSubstring(plain, query string, style lipgloss.Style) string {
	if query == "" {
		return plain
	}
	lower := strings.ToLower(plain)
	q := strings.ToLower(query)
	var out strings.Builder
	start := 0
	for {
		idx := strings.Index(lower[start:], q)
		if idx < 0 {
			out.WriteString(plain[start:])
			break
		}
		abs := start + idx
		out.WriteString(plain[start:abs])
		out.WriteString(style.Render(plain[abs : abs+len(query)]))
		start = abs + len(query)
	}
	return out.String()
}

func (m *diffModel) renderUnified(b *strings.Builder) {
	hl := lipgloss.NewStyle().Background(lipgloss.Color("#854d0e"))
	currMatch := lipgloss.NewStyle().Background(lipgloss.Color("#b91c1c"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	activeLineStyle := lipgloss.NewStyle().Background(lipgloss.Color("#1e3a5f"))
	dimStyle := lipgloss.NewStyle().Faint(true)

	activeStart, activeEnd := m.activeRange()

	for i, pl := range m.parsed {
		line := m.formatLine(pl, &lineNumStyle)

		if pl.kind == lineContext || pl.kind == lineAdd || pl.kind == lineDel {
			line = highlightCSSLine(line)
		}

		plain := stripANSI(line)

		if m.searchQ != "" {
			if strings.Contains(strings.ToLower(plain), strings.ToLower(m.searchQ)) {
				style := hl
				if m.searchMatchCurr >= 0 && m.searchMatchCurr < len(m.searchMatchIdxs) && m.searchMatchIdxs[m.searchMatchCurr] == i {
					style = currMatch
				}
				b.WriteString(highlightSubstring(plain, m.searchQ, style))
				b.WriteString("\n")
				continue
			}
		}

		if i >= activeStart && i < activeEnd {
			if len(m.hunkIdx) > 0 && m.currentHunk < len(m.hunkIdx) && i == m.hunkIdx[m.currentHunk] {
				line = activeLineStyle.Render(line)
			}
		} else if pl.kind != lineEmpty {
			line = dimStyle.Render(line)
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
	currMatch := lipgloss.NewStyle().Background(lipgloss.Color("#b91c1c"))
	activeLineStyle := lipgloss.NewStyle().Background(lipgloss.Color("#1e3a5f"))
	dimStyle := lipgloss.NewStyle().Faint(true)
	divider := lipgloss.NewStyle().Width(3).Render(" │ ")

	groups := m.groupSideBySide()
	activeStart, activeEnd := m.activeRange()

	for i, g := range groups {
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

		switch g.kind {
		case lineHunkHeader:
			styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")).Bold(true).Render(g.oldContent)
			if len(m.hunkIdx) > 0 && m.currentHunk < len(m.hunkIdx) && i == m.hunkIdx[m.currentHunk] && i >= activeStart && i < activeEnd {
				styled = activeLineStyle.Render(styled)
			}
			b.WriteString(styled)
			b.WriteString("\n")
			continue
		case lineFileHeader:
			styled := lipgloss.NewStyle().Foreground(lipgloss.Color("#6366f1")).Render(g.oldContent)
			b.WriteString(styled)
			b.WriteString("\n")
			continue
		case lineEmpty:
			b.WriteString("\n")
			continue
		}

		oldStyled := renderSideContent(oldContent, oldNum, g.kind == lineDel, half-2)
		newStyled := renderSideContent(newContent, newNum, g.kind == lineAdd, half-2)

		left := lipgloss.NewStyle().Width(half - 2).Render(oldStyled)
		right := lipgloss.NewStyle().Width(half - 2).Render(newStyled)

		line := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)

		if m.searchQ != "" {
			plain := stripANSI(line)
			if strings.Contains(strings.ToLower(plain), strings.ToLower(m.searchQ)) {
				style := hl
				if m.searchMatchCurr >= 0 && m.searchMatchCurr < len(m.searchMatchIdxs) && m.searchMatchIdxs[m.searchMatchCurr] == i {
					style = currMatch
				}
				line = highlightSubstring(plain, m.searchQ, style)
			}
		} else if i >= activeStart && i < activeEnd {
			if g.kind != lineEmpty && g.kind != lineHunkHeader && g.kind != lineFileHeader {
				line = activeLineStyle.Render(line)
			}
		} else if g.kind != lineEmpty && g.kind != lineHunkHeader && g.kind != lineFileHeader {
			line = dimStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}
}

func renderSideContent(content, num string, highlighted bool, width int) string {
	if content == "" {
		return strings.Repeat(" ", width)
	}
	var prefix string
	if num != "" {
		prefix = padRight(num, 4)
	} else {
		prefix = "    "
	}
	text := content
	maxContent := width - len([]rune(prefix)) - 1
	if maxContent < 1 {
		maxContent = 1
	}
	if len([]rune(text)) > maxContent {
		var wrapped strings.Builder
		runes := []rune(text)
		for len(runes) > 0 {
			chunk := maxContent
			if chunk > len(runes) {
				chunk = len(runes)
			}
			wrapped.WriteString(prefix + " " + string(runes[:chunk]))
			runes = runes[chunk:]
			if len(runes) > 0 {
				wrapped.WriteString("\n")
			}
			prefix = padRight("", 4)
		}
		styled := wrapped.String()
		styled = highlightCSSLine(styled)
		if highlighted {
			if strings.HasPrefix(content, "+") {
				styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Render(styled)
			} else if strings.HasPrefix(content, "-") {
				styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(styled)
			}
		}
		return styled
	}
	styled := prefix + " " + text
	styled = highlightCSSLine(styled)
	if highlighted {
		if strings.HasPrefix(content, "+") {
			styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Render(styled)
		} else if strings.HasPrefix(content, "-") {
			styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(styled)
		}
	}
	return styled
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
	case lineEmpty:
		return ""
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
	if m.content == "" {
		m.build()
	}
	yOff := m.vp.YOffset
	m.render()
	m.vp.SetContent(m.content)
	if yOff > 0 {
		m.vp.YOffset = yOff
	}
}

func (m *diffModel) viewportHeight(totalHeight int) int {
	h := totalHeight - 5
	if m.sideBySide {
		h -= 2
	}
	if h < 3 {
		h = 3
	}
	return h
}

func (m *diffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.vp.Width = msg.Width - 2
		m.vp.Height = m.viewportHeight(msg.Height)
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
		m.buildSearchMatches()
		if len(m.searchMatchIdxs) > 0 {
			m.searchMatchCurr = 0
		}
		m.render()
		m.vp.SetContent(m.content)
		if len(m.searchMatchIdxs) > 0 {
			m.scrollToMatch()
		}
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

func (m *diffModel) buildSearchMatches() {
	m.searchMatchIdxs = nil
	m.searchMatchCurr = 0
	if m.searchQ == "" {
		return
	}
	q := strings.ToLower(m.searchQ)
	for i, pl := range m.parsed {
		if strings.Contains(strings.ToLower(pl.text), q) {
			m.searchMatchIdxs = append(m.searchMatchIdxs, i)
		}
	}
}

func (m *diffModel) scrollToMatch() {
	if len(m.searchMatchIdxs) == 0 {
		return
	}
	idx := m.searchMatchIdxs[m.searchMatchCurr]
	m.vp.GotoTop()
	for j := 0; j < idx; j++ {
		m.vp.LineDown(1)
	}
}

func (m *diffModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.showHelp {
		m.showHelp = false
		m.count = 0
		return m, nil
	}

	if key == "q" || key == keyCtrlC {
		m.pendingG = false
		m.pendingZ = false
		return m, tea.Quit
	}

	if key == keyEscape {
		m.pendingG = false
		m.pendingZ = false
		if m.searchMode {
			m.searchMode = false
			m.searchQ = ""
			m.searchIn.Blur()
			m.refreshViewport()
		}
		return m, nil
	}

	if m.pendingZ {
		m.pendingZ = false
		m.pendingG = false
		switch key {
		case "z":
			m.vp.GotoTop()
			if len(m.hunkIdx) > 0 {
				idx := m.hunkIdx[m.currentHunk]
				m.vp.GotoTop()
				for i := 0; i < idx; i++ {
					m.vp.LineDown(1)
				}
				half := m.vp.Height / 2
				if idx > half {
					m.vp.YOffset = idx - half
				}
			}
			m.count = 0
			return m, nil
		case "t":
			m.vp.GotoTop()
			if len(m.hunkIdx) > 0 {
				idx := m.hunkIdx[m.currentHunk]
				for i := 0; i < idx; i++ {
					m.vp.LineDown(1)
				}
			}
			m.count = 0
			return m, nil
		case "b":
			if len(m.hunkIdx) > 0 {
				idx := m.hunkIdx[m.currentHunk]
				m.vp.GotoBottom()
				remaining := m.vp.YOffset
				m.vp.GotoTop()
				if idx > remaining {
					m.vp.YOffset = idx - remaining
				} else {
					for i := 0; i < idx; i++ {
						m.vp.LineDown(1)
					}
				}
			}
			m.count = 0
			return m, nil
		}
		m.count = 0
		return m, nil
	}

	switch key {
	case "{":
		m.pendingG = false
		m.count = 0
		m.prevHunk()
		return m, nil
	case "}":
		m.pendingG = false
		m.count = 0
		m.nextHunk()
		return m, nil

	case "n":
		m.pendingG = false
		m.pendingZ = false
		m.pendingY = false
		if m.searchQ != "" && len(m.searchMatchIdxs) > 0 {
			m.searchMatchCurr = (m.searchMatchCurr + 1) % len(m.searchMatchIdxs)
			m.render()
			m.vp.SetContent(m.content)
			m.scrollToMatch()
			m.count = 0
			return m, nil
		}
		n := m.count
		if n < 1 {
			n = 1
		}
		for i := 0; i < n; i++ {
			m.nextHunk()
		}
		m.count = 0
		return m, nil
	case "N":
		m.pendingG = false
		m.pendingZ = false
		m.pendingY = false
		if m.searchQ != "" && len(m.searchMatchIdxs) > 0 {
			m.searchMatchCurr = (m.searchMatchCurr - 1 + len(m.searchMatchIdxs)) % len(m.searchMatchIdxs)
			m.render()
			m.vp.SetContent(m.content)
			m.scrollToMatch()
			m.count = 0
			return m, nil
		}
		n := m.count
		if n < 1 {
			n = 1
		}
		for i := 0; i < n; i++ {
			m.prevHunk()
		}
		m.count = 0
		return m, nil

	case "j", "down":
		m.pendingG = false
		n := m.count
		if n < 1 {
			n = 1
		}
		m.vp.LineDown(n)
		m.count = 0
		return m, nil

	case "k", "up":
		m.pendingG = false
		n := m.count
		if n < 1 {
			n = 1
		}
		m.vp.LineUp(n)
		m.count = 0
		return m, nil

	case "v":
		m.pendingG = false
		m.sideBySide = !m.sideBySide
		yOff := m.vp.YOffset
		m.render()
		m.vp.SetContent(m.content)
		if yOff > 0 && yOff < len(m.parsed) {
			m.vp.YOffset = yOff
		}
		return m, nil

	case "g":
		m.pendingZ = false
		if m.pendingG {
			m.pendingG = false
			m.vp.GotoTop()
		} else {
			m.pendingG = true
		}
		m.count = 0
		return m, nil
	case "G":
		m.pendingG = false
		m.pendingZ = false
		m.vp.GotoBottom()
		m.count = 0
		return m, nil

	case "z":
		m.pendingG = false
		if m.pendingZ {
			m.pendingZ = false
			m.vp.GotoTop()
			if len(m.hunkIdx) > 0 {
				idx := m.hunkIdx[m.currentHunk]
				m.vp.GotoTop()
				for i := 0; i < idx; i++ {
					m.vp.LineDown(1)
				}
				half := m.vp.Height / 2
				if idx > half {
					m.vp.YOffset = idx - half
				}
			}
		} else {
			m.pendingZ = true
		}
		m.count = 0
		return m, nil

	case "/":
		m.pendingG = false
		m.pendingZ = false
		m.searchMode = true
		m.searchIn.Focus()
		m.searchIn.SetValue("")
		m.searchQ = ""
		m.refreshViewport()
		return m, nil

	case "y":
		m.pendingG = false
		m.pendingZ = false
		if m.pendingY {
			m.pendingY = false
			return m, m.yankLineContent()
		}
		m.pendingY = true
		return m, nil

	case "Y":
		m.pendingG = false
		m.pendingZ = false
		m.pendingY = false
		return m, m.yankAll()

	case "o":
		m.pendingG = false
		m.pendingZ = false
		m.pendingY = false
		return m, m.openInEditor()

	case "e":
		m.pendingG = false
		m.pendingZ = false
		m.pendingY = false
		m.count = 0
		return m, m.runTLDR()

	case "?":
		m.pendingG = false
		m.pendingZ = false
		m.pendingY = false
		m.showHelp = !m.showHelp
		m.count = 0
		return m, nil

	case "CtrlU":
		m.pendingG = false
		m.pendingZ = false
		m.vp.HalfViewUp()
		return m, nil

	case "CtrlD":
		m.pendingG = false
		m.pendingZ = false
		m.vp.HalfViewDown()
		return m, nil
	}

	if m.pendingY {
		m.pendingY = false
		return m, m.yankHunk()
	}
	m.pendingG = false
	m.pendingZ = false
	m.pendingY = false

	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 {
		r := msg.Runes[0]
		if unicode.IsDigit(r) {
			m.count = m.count*10 + int(r-'0')
			return m, nil
		}
	}

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

func (m *diffModel) yankLineContent() tea.Cmd {
	return func() tea.Msg {
		if len(m.hunkIdx) == 0 || m.currentHunk >= len(m.hunkIdx) {
			return nil
		}
		start := m.hunkIdx[m.currentHunk]
		if start >= 0 && start < len(m.parsed) {
			text := m.parsed[start].text
			if strings.HasPrefix(text, "@@") {
				if parts := strings.Split(text, "@@"); len(parts) >= 3 {
					text = strings.TrimSpace(parts[2])
				}
			}
			if err := clipboard.WriteAll(text); err != nil {
				return nil
			}
		}
		return nil
	}
}

func (m *diffModel) runTLDR() tea.Cmd {
	return func() tea.Msg {
		if m.result == nil {
			return nil
		}
		if m.tldrResult == nil {
			m.tldrResult = core.AnalyzeDiff(m.result.Diff)
			m.tldrResult.VersionA = m.result.VersionA
			m.tldrResult.VersionB = m.result.VersionB
		}
		fmt.Print(m.tldrResult.String())
		fname := fmt.Sprintf("ocd-tldr-%s-%s.toml", m.result.VersionA, m.result.VersionB)
		fname = strings.ReplaceAll(fname, ".", "_")
		exportTLDR(m.tldrResult, fname, "toml")
		fmt.Printf("\nTLDR exported to %s\n", fname)
		return nil
	}
}

func exportTLDR(t *core.TLDRResult, path, format string) {
	var data []byte
	switch format {
	case "json":
		data, _ = json.MarshalIndent(t, "", "  ")
	case "yaml":
		data, _ = yaml.Marshal(t)
	default:
		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)
		encoder.Encode(t)
		data = buf.Bytes()
	}
	os.WriteFile(path, data, 0644)
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

	editor := os.Getenv("OCD_DIFF_PAGER")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		if _, err := exec.LookPath("delta"); err == nil {
			editor = "delta"
		} else {
			editor = "less"
		}
	}
	if editor == "less" {
		cmd := exec.Command("less", "-R", tmpPath)
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			os.Remove(tmpPath)
			return nil
		})
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
	m.render()
	m.vp.SetContent(m.content)
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
	m.render()
	m.vp.SetContent(m.content)
}

func (m *diffModel) activeRange() (int, int) {
	if len(m.hunkIdx) == 0 {
		return 0, 0
	}
	start := m.hunkIdx[m.currentHunk]
	end := len(m.parsed)
	if m.currentHunk+1 < len(m.hunkIdx) {
		end = m.hunkIdx[m.currentHunk+1]
	}
	return start, end
}

func (m *diffModel) View() string {
	if !m.ready {
		return "\n  Loading diff view..."
	}
	if m.content == "" && m.result.HasDiff && m.result.Error == nil {
		m.build()
		m.vp.SetContent(m.content)
	} else if !m.result.HasDiff && m.result.Error == nil {
		return fmt.Sprintf("Diff: %s -> %s\n\nNo differences found.", m.result.VersionA, m.result.VersionB)
	} else if m.result.Error != nil {
		return fmt.Sprintf("Error: %v", m.result.Error)
	}

	if m.showHelp {
		return m.renderHelp()
	}

	header := m.renderHeader()
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

	footer := m.hintStyle.Render(
		fmt.Sprintf("\n%s  {}  j/k  /  o  q  ? help",
			hunkInfo,
		),
	)

	return header + "\n\n" + m.vp.View() + searchBar + footer
}

func (m *diffModel) renderHelp() string {
	helpContent := []string{
		"  Diff Viewer Help",
		"",
		"  {}        Jump prev/next hunk",
		"  j/k       Scroll up/down",
		"  n/N       Next/prev (hunk or search match)",
		"  gg/G      Top/bottom of diff",
		"  zz/zt/zb  Center/top/bottom current hunk",
		"  e         Export TLDR analysis",
		"  v         Toggle side-by-side",
		"  /         Search within diff",
		"  y         Yank current hunk to clipboard",
		"  Y         Yank entire diff to clipboard",
		"  yy        Yank current hunk header line content",
		"  o         Open diff viewer ($OCD_DIFF_PAGER / $EDITOR / delta / less)",
		"  q / Esc   Quit / Close help",
	}
	helpText := strings.Join(helpContent, "\n")
	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#a78bfa")).
		Padding(1, 2).
		Foreground(lipgloss.Color("#e5e7eb"))
	box := helpStyle.Render(helpText)
	width := m.vp.Width + 2
	if width < 80 {
		width = 80
	}
	pad := (width - lipgloss.Width(box)) / 2
	if pad < 2 {
		pad = 2
	}
	return lipgloss.NewStyle().PaddingLeft(pad).Render("\n\n  " + box + "\n\n  Press ? or Esc to close help")
}

func RunDiffViewer(result *models.DiffResult) error {
	m := NewDiffModel(result)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
