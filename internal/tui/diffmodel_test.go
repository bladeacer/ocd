package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bladeacer/ocd/internal/models"
)

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		hdr           string
		wantOld, wantNew int
	}{
		{"@@ -1,5 +1,6 @@", 1, 1},
		{"@@ -10,5 +20,6 @@", 10, 20},
		{"@@ -1 +1 @@", 1, 1},
		{"@@ -100 +200 @@ section header", 100, 200},
		{"@@ -5,3 +5,4 @@ func foo()", 5, 5},
	}

	for _, tt := range tests {
		gotOld, gotNew := parseHunkHeader(tt.hdr)
		if gotOld != tt.wantOld {
			t.Errorf("parseHunkHeader(%q) old = %d, want %d", tt.hdr, gotOld, tt.wantOld)
		}
		if gotNew != tt.wantNew {
			t.Errorf("parseHunkHeader(%q) new = %d, want %d", tt.hdr, gotNew, tt.wantNew)
		}
	}
}

func TestParseLeadingNum(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"1", 1},
		{"123", 123},
		{"1,5", 1},
		{"0abc", 0},
		{"abc", 0},
		{"", 0},
		{"10,20", 10},
	}
	for _, tt := range tests {
		got := parseLeadingNum(tt.s)
		if got != tt.want {
			t.Errorf("parseLeadingNum(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 5, "hello"},
		{"hello", 10, "hello"},
		{"hello world", 5, "he..."},
		{"", 5, ""},
		{"hi", 0, ""},
		{"abcdef", 3, "abc"},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hi", 4, "hi  "},
		{"hello", 3, "hello"},
		{"", 2, "  "},
		{"a", 1, "a"},
	}
	for _, tt := range tests {
		got := padRight(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("padRight(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"plain text", "plain text"},
		{"", ""},
		{"\x1b[1m\x1b[32mgreen bold\x1b[0m", "green bold"},
	}
	for _, tt := range tests {
		got := stripANSI(tt.input)
		if got != tt.expected {
			t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDiffModelBuildNoDiff(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "",
		HasDiff:  false,
	}
	m := NewDiffModel(r)
	m.build()
	if !strings.Contains(m.content, "No differences found.") {
		t.Errorf("expected 'No differences found.' in content, got %q", m.content)
	}
}

func TestDiffModelBuildError(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Error:    errTest("file not found"),
	}
	m := NewDiffModel(r)
	m.build()
	if !strings.Contains(m.content, "Error: file not found") {
		t.Errorf("expected error in content, got %q", m.content)
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

func TestDiffModelBuildWithDiff(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- v1.0.0\n+++ v1.0.1\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.build()
	if m.content == "" {
		t.Fatal("expected non-empty content")
	}
	if !strings.Contains(m.content, "Diff: 1.0.0 -> 1.0.1") {
		t.Errorf("expected header in content, got: %s", m.content)
	}
	if m.insertions != 1 {
		t.Errorf("expected 1 insertion, got %d", m.insertions)
	}
	if m.deletions != 1 {
		t.Errorf("expected 1 deletion, got %d", m.deletions)
	}
}

func TestFormatLine(t *testing.T) {
	pl := parsedLine{text: " hello", kind: lineContext, oldLineNum: 5, newLineNum: 5}
	m := NewDiffModel(&models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1 +1 @@\n-old\n+new\n",
		HasDiff:  true,
	})
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	line := m.formatLine(pl, &numStyle)
	if !strings.Contains(line, "hello") {
		t.Errorf("expected line to contain 'hello', got %s", line)
	}
}

func TestDiffModelYankText(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1 +1 @@\n-old\n+new\n",
		HasDiff:  true,
	})
	m.build()
	text := m.yankText(0, len(m.parsed))
	if !strings.Contains(text, "old") || !strings.Contains(text, "new") {
		t.Errorf("yankText missing content, got: %s", text)
	}
}

func TestGroupSideBySide(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.build()
	groups := m.groupSideBySide()
	if len(groups) == 0 {
		t.Fatal("expected non-empty groups")
	}
	hasDel := false
	hasAdd := false
	for _, g := range groups {
		if g.kind == lineDel {
			hasDel = true
		}
		if g.kind == lineAdd {
			hasAdd = true
		}
	}
	if !hasDel {
		t.Error("expected deletion in side-by-side groups")
	}
	if !hasAdd {
		t.Error("expected addition in side-by-side groups")
	}
}

func TestDiffModelWindowSizeMsg(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	_, cmd := m.Update(msg)
	if !m.ready {
		t.Error("expected ready=true after WindowSizeMsg")
	}
	if cmd != nil {
		t.Log("WindowSizeMsg returned a command")
	}
}

func TestDiffModelToggleSideBySide(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	if m.sideBySide {
		t.Error("expected sideBySide=false initially")
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !m.sideBySide {
		t.Error("expected sideBySide=true after pressing v")
	}
	if cmd != nil {
		t.Log("toggle key returned a command")
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if m.sideBySide {
		t.Error("expected sideBySide=false after pressing v again")
	}
}

func TestDiffModelQuitKey(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected non-nil command for quit key")
	}
}

func TestDiffModelSearchMode(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.searchMode {
		t.Error("expected searchMode=true after pressing /")
	}
	if cmd != nil {
		t.Log("search returned a command")
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.searchMode {
		t.Error("expected searchMode=false after escape")
	}
}

func TestDiffModelViewBeforeReady(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	v := m.View()
	if !strings.Contains(v, "Loading diff view...") {
		t.Errorf("expected 'Loading diff view...' in view, got %q", v)
	}
}

func TestDiffModelViewAfterReady(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	v := m.View()
	if !strings.Contains(v, "Diff: 1.0.0 -> 1.0.1") {
		t.Errorf("expected header in view, got: %s", v)
	}
}

func TestDiffModelHunkNavigation(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n@@ -10,3 +10,3 @@\n x\n-y\n+z\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	if m.currentHunk != 0 {
		t.Errorf("expected initial hunk 0, got %d", m.currentHunk)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.currentHunk != 1 {
		t.Errorf("expected hunk 1 after n, got %d", m.currentHunk)
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	if m.currentHunk != 0 {
		t.Errorf("expected hunk 0 after N, got %d", m.currentHunk)
	}
	_ = cmd
}

func TestDiffModelGotoTop(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if !m.pendingG {
		t.Error("expected pendingG=true after first g")
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.pendingG {
		t.Error("expected pendingG=false after second g")
	}

	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.pendingG {
		t.Error("expected pendingG=false after G")
	}
	_ = cmd
}

func TestDiffModelOpenInEditor(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "test diff",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	cmd := m.openInEditor()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

func TestDiffModelYankHunk(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.build()
	cmd := m.yankHunk()
	if cmd == nil {
		t.Fatal("expected non-nil command for yankHunk")
	}
}

func TestDiffModelYankAll(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	cmd := m.yankAll()
	if cmd == nil {
		t.Fatal("expected non-nil command for yankAll")
	}
}

func TestDiffModelNextPrevHunkEmpty(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "no hunks here",
		HasDiff:  false,
	}
	m := NewDiffModel(r)
	m.build()
	m.nextHunk()
	m.prevHunk()
}

func TestDiffModelRefreshViewport(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.0.1",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n d\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.vp.Width = 100
	m.vp.Height = 50
	m.ready = true

	m.refreshViewport()
	if m.content == "" {
		t.Error("expected non-empty content after refreshViewport")
	}
}
