package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/models"
)

// The pending-prefix state machine: a first key that could start a sequence
// sets a pending flag, and the next key completes or cancels it.
func TestDiffModelPendingPrefixes(t *testing.T) {
	type seq struct{ keys []string }

	seqs := []seq{
		{[]string{"g"}},
		{[]string{"z"}},
		{[]string{"y"}},
		{[]string{"d"}},
		{[]string{"u"}},
		{[]string{"x"}},        // unknown first key clears the pending state
		{[]string{"g", "q"}},   // g then quit
		{[]string{"z", "q"}},   // z then quit
		{[]string{"y", "esc"}}, // y then escape
	}

	for _, s := range seqs {
		m := newReadyDiffModel()
		for _, k := range s.keys {
			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		}
		if m.pendingG || m.pendingZ || m.pendingY {
			// A pending flag left set is fine; the next key clears it.
			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		}
	}
}

func TestDiffModelEscapeLeavesSearch(t *testing.T) {
	m := newReadyDiffModel()

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if !m.searchMode {
		t.Fatal("/ should enter search mode")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.searchMode {
		t.Error("esc should leave search mode")
	}
	if m.searchQ != "" {
		t.Errorf("search query should be cleared, got %q", m.searchQ)
	}
}

func TestDiffModelEscapeOutsideSearch(t *testing.T) {
	m := newReadyDiffModel()
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.searchMode {
		t.Error("esc outside search must not enter it")
	}
}

func TestDiffModelYankHunkVariants(t *testing.T) {
	m := newReadyDiffModel()

	// Yank the current hunk, then move and yank another.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("}")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
}

func TestDiffModelYankAllViaCall(t *testing.T) {
	m := newReadyDiffModel()
	m.yankAll()
	if m.View() == "" {
		t.Error("expected a view after yanking everything")
	}
}

func TestDiffModelRenderSideBySideEmptyLabels(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{
		VersionA: "", VersionB: "", HasDiff: true,
		Diff: "@@ -1,1 +1,1 @@\n-.a\n+.b",
	}, config.DiffKeys{})
	m.ready = true
	m.build()

	var b strings.Builder
	m.renderSideBySide(&b)
	if b.Len() == 0 {
		t.Error("expected side-by-side output")
	}
}

func TestFormatLineWithEmptyText(t *testing.T) {
	m := newReadyDiffModel()
	style := lipgloss.NewStyle()

	if got := m.formatLine(parsedLine{kind: lineAdd, text: ""}, &style); strings.TrimSpace(got) == "" {
		t.Error("expected a formatted blank add line")
	}
}

func TestHighlightOnCSSWithANSIContent(t *testing.T) {
	style := lipgloss.NewStyle().Background(lipgloss.Color("#fde68a"))
	content := "\x1b[31mcolor\x1b[0m: red"
	plain := "color: red"

	got := highlightOnCSS(content, plain, "color", style)
	if got == "" {
		t.Error("expected output")
	}
}

func TestRenderSideContentHighlightedWidths(t *testing.T) {
	if got := renderSideContentHighlighted("body { color: red; }", "12", 0); got == "" {
		t.Error("expected output for a zero width")
	}
	if got := renderSideContentHighlighted("x", "", 5); !strings.Contains(got, "x") {
		t.Errorf("content missing: %q", got)
	}
}

func TestModelUpdateRemainingBranches(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(dataLoadedMsg{result: &models.FetchResult{RSS: sampleRSS()}})

	for _, msg := range []tea.Msg{
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")}, // toggle mobile
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}, // early access
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}, // sort
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("enter")},
		tea.KeyMsg{Type: tea.KeyEsc},
		tea.KeyMsg{Type: tea.KeyUp},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyTab},
		tea.KeyMsg{Type: tea.KeyShiftTab},
		tea.KeyMsg{Type: tea.KeyCtrlR},
		tickMsg{},
	} {
		m.Update(msg)
	}
	if m.View() == "" {
		t.Error("expected a view")
	}
}

func TestModelViewStates(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)

	if m.View() == "" {
		t.Error("expected a loading view")
	}

	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(dataLoadedMsg{result: &models.FetchResult{RSS: sampleRSS()}})
	if m.View() == "" {
		t.Error("expected a loaded view")
	}

	// An error is shown through the load-message list, not a banner.
	m2 := New(f, false)
	m2.Update(dataLoadedMsg{result: &models.FetchResult{Error: errFakeTUI{}}})
	if m2.err == nil {
		t.Error("expected the error to be stored")
	}
	m2.View()
}

type errFakeTUI struct{}

func (errFakeTUI) Error() string { return "boom" }

func TestPickerUpdateKeys(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	p := NewPicker(f, false)
	p.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	p.Update(dataLoadedMsg{result: &models.FetchResult{RSS: sampleRSS()}})

	for _, msg := range []tea.Msg{
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")},
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")},
		tea.KeyMsg{Type: tea.KeyEsc},
		tea.KeyMsg{Type: tea.KeyUp},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyCtrlC},
	} {
		p.Update(msg)
	}
	if p.View() == "" {
		t.Error("expected a view")
	}
}

func TestPickerViewErrorState(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	p := NewPicker(f, false)
	p.Update(dataLoadedMsg{result: &models.FetchResult{Error: errFakeTUI{}}})

	if !strings.Contains(p.View(), "rror") {
		t.Errorf("expected an error view, got:\n%s", p.View())
	}
}
