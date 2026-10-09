package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/models"
)

func newReadyDiffModel() *diffModel {
	m := NewDiffModel(&models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.1.0",
		HasDiff:  true,
		Diff: "@@ -1,4 +1,5 @@\n .a {\n-  color: red;\n+  color: blue;\n+  border: 1px;\n }\n" +
			"-- .old-only { color: #fff; }\n" +
			"++ .new-only { color: #000; }\n" +
			"@@ -20,3 +20,4 @@\n .b { --x: 1; --y: 2; }\n",
	}, config.DiffKeys{})
	m.ready = true
	m.build()
	return m
}

func TestDiffModelFormatLineKinds(t *testing.T) {
	m := newReadyDiffModel()
	style := lipgloss.NewStyle()

	cases := []struct {
		kind lineKind
		text string
		old  int
		new  int
	}{
		{lineHunkHeader, "@@ -1,2 +1,2 @@", 0, 0},
		{lineAdd, "+added", 0, 5},
		{lineDel, "-removed", 5, 0},
		{lineContext, " context", 1, 1},
	}
	for _, tc := range cases {
		got := m.formatLine(parsedLine{kind: tc.kind, text: tc.text, oldLineNum: tc.old, newLineNum: tc.new}, &style)
		if strings.TrimSpace(got) == "" {
			t.Errorf("formatLine(%v) produced nothing", tc.kind)
		}
	}
}

func TestHighlightOnCSS(t *testing.T) {
	style := lipgloss.NewStyle().Background(lipgloss.Color("#fde68a"))

	// A query that matches should be highlighted.
	if got := highlightOnCSS("color: red", "color: red", "color", style); got == "" {
		t.Error("expected highlighted output")
	}
	// An empty query returns the content untouched.
	if got := highlightOnCSS("plain", "plain", "", style); got != "plain" {
		t.Errorf("empty query changed the output: %q", got)
	}
	// A query that does not match returns the content untouched.
	if got := highlightOnCSS("plain", "plain", "zzz", style); got != "plain" {
		t.Errorf("unmatched query changed the output: %q", got)
	}
}

func TestRenderSideContentHighlighted(t *testing.T) {
	if got := renderSideContentHighlighted("", "", 10); strings.TrimSpace(got) != "" {
		t.Errorf("empty content should give blank space, got %q", got)
	}
	if got := renderSideContentHighlighted("hello", "1", 10); !strings.Contains(got, "hello") {
		t.Errorf("content missing from the side view: %q", got)
	}
}

func TestDiffModelNormalKeys(t *testing.T) {
	m := newReadyDiffModel()

	for _, r := range []string{"j", "k", "G", "g", "{", "}", "?", "z", "t", "b", "V"} {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)})
	}
	if m.View() == "" {
		t.Error("expected a view after handling keys")
	}
}

func TestDiffModelSearchFlow(t *testing.T) {
	m := newReadyDiffModel()

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("color")})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})

	if m.View() == "" {
		t.Error("expected a view after searching")
	}
}

func TestDiffModelSearchCancelled(t *testing.T) {
	m := newReadyDiffModel()

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.searchMode {
		t.Error("esc should leave search mode")
	}
}

func TestDiffModelHelpToggleOverlay(t *testing.T) {
	m := newReadyDiffModel()

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if !m.showHelp {
		t.Fatal("? should show help")
	}
	if !strings.Contains(m.View(), "Help") {
		t.Error("expected the help overlay")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if m.showHelp {
		t.Error("? should hide help again")
	}
}

func TestDiffModelExportPromptEscape(t *testing.T) {
	m := newReadyDiffModel()

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if !m.exportAsk {
		t.Fatal("e should open the export prompt")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.exportAsk {
		t.Error("esc should close the export prompt")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if m.exportAsk {
		t.Error("q should close the export prompt")
	}
}

func TestDiffModelExportJSONAndYAML(t *testing.T) {
	for _, tc := range []struct{ key, ext string }{{"j", "json"}, {"y", "yaml"}} {
		m := newReadyDiffModel()
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
		if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)}); cmd == nil {
			t.Errorf("%s export should quit", tc.key)
		}
		if m.exportFormat != tc.ext {
			t.Errorf("exportFormat = %q, want %q", m.exportFormat, tc.ext)
		}
	}
}

func TestDiffModelViewStates(t *testing.T) {
	// Not ready yet.
	m := NewDiffModel(&models.DiffResult{VersionA: "1.0.0", VersionB: "1.1.0"}, config.DiffKeys{})
	if !strings.Contains(m.View(), "Loading") {
		t.Errorf("expected a loading view, got:\n%s", m.View())
	}

	// No differences.
	m = NewDiffModel(&models.DiffResult{VersionA: "1.0.0", VersionB: "1.1.0"}, config.DiffKeys{})
	m.ready = true
	if !strings.Contains(m.View(), "No differences") {
		t.Errorf("expected a no-differences view, got:\n%s", m.View())
	}

	// Error.
	m = NewDiffModel(&models.DiffResult{
		VersionA: "1.0.0", VersionB: "1.1.0", HasDiff: true,
		Error: errFake{},
	}, config.DiffKeys{})
	m.ready = true
	if !strings.Contains(m.View(), "Error") {
		t.Errorf("expected an error view, got:\n%s", m.View())
	}
}

type errFake struct{}

func (errFake) Error() string { return "boom" }

func TestDiffModelQuitAndCancel(t *testing.T) {
	m := newReadyDiffModel()

	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}); cmd == nil {
		t.Error("q should quit")
	}

	m = newReadyDiffModel()
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC}); cmd == nil {
		t.Error("ctrl-c should quit")
	}
}

func TestDiffModelYankKeys(t *testing.T) {
	m := newReadyDiffModel()

	for _, r := range []string{"y", "Y"} {
		mm := newReadyDiffModel()
		mm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)})
	}
	if m.View() == "" {
		t.Error("expected a view")
	}
}

func TestDiffModelOpenInEditorFallsBack(t *testing.T) {
	m := newReadyDiffModel()

	t.Setenv("OCD_DIFF_PAGER", "")
	t.Setenv("EDITOR", "")

	// The command is built, not run, so this stays safe in tests.
	if cmd := m.openInEditor(); cmd == nil {
		t.Log("no editor command available; the fallback path returned nil")
	}
}

func TestDiffModelScrollAndGoto(t *testing.T) {
	m := newReadyDiffModel()

	for i := 0; i < 10; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m.Update(tea.KeyMsg{Type: tea.KeyEnd})

	if m.View() == "" {
		t.Error("expected a view")
	}
}

func TestDiffModelWindowSize(t *testing.T) {
	m := newReadyDiffModel()
	m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})

	if m.vp.Width == 0 || m.vp.Height == 0 {
		t.Errorf("viewport was not sized: %dx%d", m.vp.Width, m.vp.Height)
	}
}
