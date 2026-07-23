package tui

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bladeacer/ocd/internal/models"
)

func TestMain(m *testing.M) {
	os.Setenv("CLICOLOR_FORCE", "1")
	os.Exit(m.Run())
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		hdr              string
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
	if m.content != "" {
		t.Errorf("expected empty content for no-diff, got %q", m.content)
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
	if !strings.Contains(m.content, "a") {
		t.Errorf("expected diff content, got: %s", m.content)
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

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !m.sideBySide {
		t.Error("expected sideBySide=true after pressing v")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
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

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.searchMode {
		t.Error("expected searchMode=true after pressing /")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
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

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.currentHunk != 1 {
		t.Errorf("expected hunk 1 after n, got %d", m.currentHunk)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	if m.currentHunk != 0 {
		t.Errorf("expected hunk 0 after N, got %d", m.currentHunk)
	}
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

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if !m.pendingG {
		t.Error("expected pendingG=true after first g")
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.pendingG {
		t.Error("expected pendingG=false after second g")
	}

	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.pendingG {
		t.Error("expected pendingG=false after G")
	}
}

func TestDiffModelOpenInEditor(t *testing.T) {
	t.Setenv("EDITOR", "cat")
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

func TestDiffModelInit(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{})
	cmd := m.Init()
	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestDiffModelHandleSearchKeyEscape(t *testing.T) {
	r := &models.DiffResult{VersionA: "a", VersionB: "b"}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.searchMode = true
	m.searchIn.Focus()
	m.searchIn.SetValue("test")
	m.searchQ = "test"
	m.handleSearchKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.searchMode {
		t.Error("expected searchMode false after escape")
	}
	if m.searchQ != "" {
		t.Errorf("expected empty searchQ after escape, got %q", m.searchQ)
	}
}

func TestDiffModelHandleSearchKeyEnter(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1 +1 @@\n-old\n+new\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.searchMode = true
	m.searchIn.Focus()
	m.searchIn.SetValue("old")
	m.handleSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.searchMode {
		t.Error("expected searchMode false after enter")
	}
	if m.searchQ != "old" {
		t.Errorf("expected searchQ='old', got %q", m.searchQ)
	}
}

func TestDiffModelHandleSearchKeyTab(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1 +1 @@\n-old\n+new\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.searchMode = true
	m.searchIn.Focus()
	m.searchIn.SetValue("new")
	m.handleSearchKey(tea.KeyMsg{Type: tea.KeyTab})
	if m.searchMode {
		t.Error("expected searchMode false after tab")
	}
	if m.searchQ != "new" {
		t.Errorf("expected searchQ='new', got %q", m.searchQ)
	}
}

func TestDiffModelHandleSearchKeyBackspace(t *testing.T) {
	r := &models.DiffResult{VersionA: "a", VersionB: "b"}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.searchMode = true
	m.searchIn.Focus()
	m.searchIn.SetValue("abc")
	m.searchQ = "abc"
	m.handleSearchKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchQ != "ab" {
		t.Errorf("expected searchQ='ab' after backspace, got %q", m.searchQ)
	}
}

func TestDiffModelHandleSearchKeyCharacter(t *testing.T) {
	r := &models.DiffResult{VersionA: "a", VersionB: "b"}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.searchMode = true
	m.searchIn.Focus()
	m.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.searchQ != "x" {
		t.Errorf("expected searchQ='x', got %q", m.searchQ)
	}
}

func TestDiffModelHandleSearchKeyNonCharacter(t *testing.T) {
	r := &models.DiffResult{VersionA: "a", VersionB: "b"}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.searchMode = true
	m.searchIn.Focus()
	_, cmd := m.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}})
	if cmd != nil {
		t.Error("expected nil cmd for non-character key sequence")
	}
}

func TestDiffModelScrollToMatch(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: "first"},
		{text: "second"},
		{text: "target"},
	}
	m.vp.Width = 100
	m.vp.Height = 3
	m.vp.SetContent("line0\nline1\nline2\nline3\nline4\n")
	m.searchQ = "target"
	m.scrollToMatch()
	if m.vp.YOffset != 2 {
		t.Errorf("expected YOffset=2, got %d", m.vp.YOffset)
	}
}

func TestDiffModelScrollToMatchNoMatch(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: "first"},
		{text: "second"},
	}
	m.vp.Width = 100
	m.vp.Height = 50
	m.searchQ = "nonexistent"
	m.scrollToMatch()
	if m.vp.YOffset != 0 {
		t.Errorf("expected YOffset=0 after GotoTop with no match, got %d", m.vp.YOffset)
	}
}

func TestDiffModelRenderUnified(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: "--- a/file.go", kind: lineFileHeader},
		{text: "+++ b/file.go", kind: lineFileHeader},
		{text: "@@ -1,3 +1,4 @@", kind: lineHunkHeader},
		{text: " context", kind: lineContext, oldLineNum: 1, newLineNum: 1},
		{text: "-old", kind: lineDel, oldLineNum: 2, newLineNum: 0},
		{text: "+new", kind: lineAdd, oldLineNum: 0, newLineNum: 2},
		{text: "", kind: lineEmpty},
	}
	var b strings.Builder
	m.renderUnified(&b)
	result := stripANSI(b.String())
	if !strings.Contains(result, "file.go") {
		t.Error("expected file header in output")
	}
	if !strings.Contains(result, "@@") {
		t.Error("expected hunk header in output")
	}
	if !strings.Contains(result, "context") {
		t.Error("expected context line in output")
	}
	if !strings.Contains(result, "-old") {
		t.Error("expected deletion line in output")
	}
	if !strings.Contains(result, "+new") {
		t.Error("expected addition line in output")
	}
}

func TestDiffModelRenderUnifiedWithSearch(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: " hello", kind: lineContext, oldLineNum: 1, newLineNum: 1},
		{text: "-world", kind: lineDel, oldLineNum: 2, newLineNum: 0},
		{text: "+world", kind: lineAdd, oldLineNum: 0, newLineNum: 2},
	}
	m.searchQ = "hello"
	var b strings.Builder
	m.renderUnified(&b)
	result := stripANSI(b.String())
	if !strings.Contains(result, "hello") {
		t.Error("expected hello in output with search highlight")
	}
}

func TestDiffModelRenderSideBySideDefaultWithSearch(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: "--- a", kind: lineFileHeader},
		{text: "+++ b", kind: lineFileHeader},
		{text: "@@ -1,3 +1,4 @@", kind: lineHunkHeader},
		{text: " context", kind: lineContext, oldLineNum: 1, newLineNum: 1},
		{text: "-old", kind: lineDel, oldLineNum: 2, newLineNum: 0},
		{text: "+new", kind: lineAdd, oldLineNum: 0, newLineNum: 2},
	}
	m.vp.Width = 100
	m.searchQ = "context"
	var b strings.Builder
	m.renderSideBySide(&b)
	result := stripANSI(b.String())
	if !strings.Contains(result, "context") {
		t.Error("expected context in output")
	}
}

func TestDiffModelOpenInEditorWithEmptyDiff(t *testing.T) {
	r := &models.DiffResult{VersionA: "a", VersionB: "b", Diff: "", HasDiff: false}
	m := NewDiffModel(r)
	cmd := m.openInEditor()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
}

func TestDiffModelYankHunkNoHunks(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	cmd := m.yankHunk()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	msg := cmd()
	if msg != nil {
		t.Error("expected nil msg from yankHunk with no hunks")
	}
}

func TestDiffModelYankAllNoContent(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b", HasDiff: false})
	cmd := m.yankAll()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	msg := cmd()
	if msg != nil {
		t.Error("expected nil msg from yankAll with no content")
	}
}

func TestDiffModelPrevHunkAtFirst(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n@@ -10,3 +10,3 @@\n x\n-y\n+z\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.currentHunk = 0
	m.prevHunk()
	if m.currentHunk != 1 {
		t.Errorf("expected currentHunk=1 after prevHunk from 0, got %d", m.currentHunk)
	}
}

func TestDiffModelHandleNormalKeyY(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.build()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	_, cmd := m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Error("expected non-nil command from y key")
	}
}

func TestDiffModelYankHunkCmd(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.build()
	cmd := m.yankHunk()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	msg := cmd()
	if msg != nil {
		t.Error("expected nil msg from yankHunk with hunks")
	}
}

func TestDiffModelYankAllCmd(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.build()
	cmd := m.yankAll()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	msg := cmd()
	if msg != nil {
		t.Error("expected nil msg from yankAll with content")
	}
}

func TestDiffModelHandleNormalKeyCtrlU(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n" + strings.Repeat("@@ -1,2 +1,2 @@\n context\n-old\n+new\n", 30),
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 10})

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	yAfterDown := m.vp.YOffset

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	yAfterUp := m.vp.YOffset

	if yAfterUp >= yAfterDown {
		t.Errorf("expected YOffset to decrease after Ctrl+U, before=%d after=%d", yAfterDown, yAfterUp)
	}
}

func TestDiffModelHandleNormalKeyCtrlD(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n" + strings.Repeat("@@ -1,2 +1,2 @@\n context\n-old\n+new\n", 30),
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 10})

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.vp.YOffset <= 0 {
		t.Errorf("expected YOffset > 0 after Ctrl+D, got %d", m.vp.YOffset)
	}
}

func TestDiffModelHandleNormalKeyGG(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n" + strings.Repeat("@@ -1,2 +1,2 @@\n context\n-old\n+new\n", 30),
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 10})

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.vp.YOffset == 0 {
		t.Skip("could not scroll down, skipping")
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if !m.pendingG {
		t.Error("expected pendingG=true after first g")
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.pendingG {
		t.Error("expected pendingG=false after second g")
	}
	if m.vp.YOffset != 0 {
		t.Errorf("expected YOffset=0 after gg, got %d", m.vp.YOffset)
	}
}

func TestDiffModelHandleNormalKeyG(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n" + strings.Repeat("@@ -1,2 +1,2 @@\n context\n-old\n+new\n", 30),
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 10})

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.pendingG {
		t.Error("expected pendingG=false after G")
	}
	if m.vp.YOffset <= 0 {
		t.Errorf("expected YOffset > 0 after G, got %d", m.vp.YOffset)
	}
}

func TestDiffModelHandleNormalKeyNWithSearch(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n@@ -10,3 +10,3 @@\n x\n-y\n+z\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.searchQ = "something"

	if m.currentHunk != 0 {
		t.Fatalf("expected hunk 0, got %d", m.currentHunk)
	}
	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.currentHunk != 1 {
		t.Errorf("expected hunk 1 after n with searchQ set, got %d", m.currentHunk)
	}
}

func TestDiffModelHandleNormalKeyO(t *testing.T) {
	t.Setenv("EDITOR", "cat")
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.build()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	_, cmd := m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if cmd == nil {
		t.Error("expected non-nil command from o key")
	}
}

func TestDiffModelHandleNormalKeySlash(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	if m.searchMode {
		t.Error("expected searchMode=false initially")
	}
	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.searchMode {
		t.Error("expected searchMode=true after pressing / in normal mode")
	}
}

func TestDiffModelHandleNormalKeyV(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	if m.sideBySide {
		t.Error("expected sideBySide=false initially")
	}
	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !m.sideBySide {
		t.Error("expected sideBySide=true after pressing v via handleNormalKey")
	}
}

func TestDiffModelHandleNormalKeyYUpper(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.build()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	_, cmd := m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	if cmd == nil {
		t.Error("expected non-nil command from Y key")
	}
}

func TestDiffModelViewEmpty(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a",
		VersionB: "b",
		HasDiff:  false,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	v := m.View()
	if !strings.Contains(v, "No differences found.") {
		t.Errorf("expected 'No differences found.' in view, got %q", v)
	}
}

func TestDiffModelRenderUnifiedFileHeader(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: "--- a/foo.go", kind: lineFileHeader},
		{text: "+++ b/foo.go", kind: lineFileHeader},
	}
	var b strings.Builder
	m.renderUnified(&b)
	result := stripANSI(b.String())
	if !strings.Contains(result, "foo.go") {
		t.Error("expected file header text in unified output")
	}
}

func TestDiffModelScrollToMatchAtEnd(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: "first"},
		{text: "second"},
		{text: "third"},
		{text: "target"},
	}
	m.vp.Width = 100
	m.vp.Height = 3
	m.vp.SetContent("line0\nline1\nline2\nline3\nline4\nline5\n")
	m.searchQ = "target"
	m.scrollToMatch()
	if m.vp.YOffset != 3 {
		t.Errorf("expected YOffset=3 for match at end, got %d", m.vp.YOffset)
	}
}

func TestDiffModelScrollToMatchEmptyQuery(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.parsed = []parsedLine{
		{text: "first"},
	}
	m.searchQ = ""
	m.scrollToMatch()
	if m.vp.YOffset != 0 {
		t.Errorf("expected YOffset=0 for empty query, got %d", m.vp.YOffset)
	}
}

func TestDiffModelScrollToMatchEmptyParsed(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.searchQ = "something"
	m.scrollToMatch()
	if m.vp.YOffset != 0 {
		t.Errorf("expected YOffset=0 for empty parsed, got %d", m.vp.YOffset)
	}
}

func TestRenderColumnHeaders(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "v1",
		VersionB: "v2",
		Diff:     "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff:  true,
	}
	m := NewDiffModel(r)
	m.sideBySide = true
	m.vp.Width = 100
	result := m.renderHeader()
	if !strings.Contains(result, "v1") {
		t.Errorf("expected v1 in header, got: %s", result)
	}
	if !strings.Contains(result, "v2") {
		t.Errorf("expected v2 in header, got: %s", result)
	}
	if !strings.Contains(result, "Old") {
		t.Errorf("expected Old in column headers, got: %s", result)
	}
	if !strings.Contains(result, "New") {
		t.Errorf("expected New in column headers, got: %s", result)
	}
}

func TestHighlightCSSLineComment(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	input := "/* comment */"
	result := highlightCSSLine(input)
	if result == input {
		t.Error("expected CSS comment to be styled")
	}
}

func TestHighlightCSSLineAtRule(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	input := "@media screen"
	result := highlightCSSLine(input)
	if result == input {
		t.Errorf("expected at-rule to be styled, got: %q", result)
	}
}

func TestHighlightCSSLinePropValue(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	input := "  color: red;"
	result := highlightCSSLine(input)
	if result == input {
		t.Error("expected property:value to be styled")
	}
	plain := stripANSI(result)
	if !strings.Contains(plain, "color") {
		t.Errorf("expected color in output, got: %s", plain)
	}
	if !strings.Contains(plain, "red") {
		t.Errorf("expected red in output, got: %s", plain)
	}
}

func TestHighlightCSSLineSelector(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	input := ".class {"
	result := highlightCSSLine(input)
	if result == input {
		t.Error("expected selector to be styled")
	}
}

func TestHighlightCSSLinePunct(t *testing.T) {
	t.Setenv("CLICOLOR_FORCE", "1")
	input := "}"
	result := highlightCSSLine(input)
	if result == input {
		t.Error("expected punctuation to be styled")
	}
}

func TestDiffModelViewWithSideBySide(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	v := m.View()
	if !strings.Contains(v, "side-by-side") {
		t.Errorf("expected 'side-by-side' in view, got: %s", v)
	}
}

func TestDiffModelHandleNormalKeyCurlyBrace(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n@@ -10,3 +10,3 @@\n x\n-y\n+z\n@@ -20,3 +20,3 @@\n p\n-q\n+r\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	if m.currentHunk != 0 {
		t.Fatalf("expected hunk 0, got %d", m.currentHunk)
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}})
	if m.currentHunk != 1 {
		t.Errorf("expected hunk 1 after }, got %d", m.currentHunk)
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}})
	if m.currentHunk != 0 {
		t.Errorf("expected hunk 0 after {, got %d", m.currentHunk)
	}
}

func TestDiffModelHandleNormalKeyCountMotion(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n" + strings.Repeat("@@ -1,2 +1,2 @@\n context\n-old\n+new\n", 30),
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 10})

	y0 := m.vp.YOffset

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if m.count != 3 {
		t.Errorf("expected count=3, got %d", m.count)
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.vp.YOffset <= y0 {
		t.Errorf("expected YOffset to increase after 3j, before=%d after=%d", y0, m.vp.YOffset)
	}
	if m.count != 0 {
		t.Errorf("expected count reset to 0, got %d", m.count)
	}
}

func TestDiffModelHandleNormalKeyCountHunk(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n@@ -10,3 +10,3 @@\n x\n-y\n+z\n@@ -20,3 +20,3 @@\n p\n-q\n+r\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	if m.currentHunk != 0 {
		t.Fatalf("expected hunk 0, got %d", m.currentHunk)
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if m.count != 2 {
		t.Errorf("expected count=2, got %d", m.count)
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.currentHunk != 2 {
		t.Errorf("expected hunk 2 after 2n, got %d", m.currentHunk)
	}
	if m.count != 0 {
		t.Errorf("expected count reset to 0, got %d", m.count)
	}
}

func TestDiffModelHandleNormalKeyZZ(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n@@ -10,3 +10,3 @@\n x\n-y\n+z\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if !m.pendingZ {
		t.Error("expected pendingZ=true after first z")
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if m.pendingZ {
		t.Error("expected pendingZ=false after second z")
	}
}

func TestDiffModelHandleNormalKeyZT(t *testing.T) {
	r := &models.DiffResult{
		VersionA: "a", VersionB: "b",
		Diff:    "--- a\n+++ b\n@@ -1,3 +1,3 @@\n a\n-b\n+c\n@@ -10,3 +10,3 @@\n x\n-y\n+z\n",
		HasDiff: true,
	}
	m := NewDiffModel(r)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if !m.pendingZ {
		t.Error("expected pendingZ=true after first z")
	}

	m.handleNormalKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if m.pendingZ {
		t.Error("expected pendingZ=false after zt")
	}
}

func TestDiffModelViewportHeightSideBySide(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{VersionA: "a", VersionB: "b"})
	m.sideBySide = true
	h := m.viewportHeight(50)
	want := 50 - 5 - 2
	if h != want {
		t.Errorf("expected %d, got %d", want, h)
	}
}
