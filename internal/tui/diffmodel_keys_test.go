package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/models"
)

func TestDiffModelEveryKeybind(t *testing.T) {
	// Drive each default keybind on a fresh model so the handler for that
	// action is reached with clean state.
	keys := []string{
		"{", "h", "}", "l", "j", "k", "down", "up",
		"g", "G", "z", "t", "b", "v", "?", "y", "Y",
		"n", "N", "/", "pgdown", "pgup", "home", "end",
		"esc", "enter", "backspace",
	}

	for _, k := range keys {
		m := newReadyDiffModel()
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		if m.View() == "" {
			t.Errorf("key %q produced an empty view", k)
		}
	}
}

func TestDiffModelKeySequences(t *testing.T) {
	sequences := [][]string{
		{"g", "g"}, {"g", "t"}, {"g", "b"}, {"g", "z"}, {"g", "y"},
		{"z", "y"}, {"z", "j"}, {"z", "k"},
		{"y", "y"}, {"Y", "Y"},
		{"d", "d"}, {"u", "u"}, {"G", "g"}, {"g", "G"},
	}

	for _, seq := range sequences {
		m := newReadyDiffModel()
		for _, k := range seq {
			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		}
		if m.View() == "" {
			t.Errorf("sequence %v produced an empty view", seq)
		}
	}
}

func TestDiffModelSpecialKeyTypes(t *testing.T) {
	msgs := []tea.Msg{
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyEsc},
		tea.KeyMsg{Type: tea.KeyBackspace},
		tea.KeyMsg{Type: tea.KeyDelete},
		tea.KeyMsg{Type: tea.KeyLeft},
		tea.KeyMsg{Type: tea.KeyRight},
		tea.KeyMsg{Type: tea.KeyUp},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.KeyMsg{Type: tea.KeyPgUp},
		tea.KeyMsg{Type: tea.KeyPgDown},
		tea.KeyMsg{Type: tea.KeyHome},
		tea.KeyMsg{Type: tea.KeyEnd},
		tea.KeyMsg{Type: tea.KeyTab},
		tea.KeyMsg{Type: tea.KeyCtrlC},
		tea.KeyMsg{Type: tea.KeyCtrlD},
		tea.KeyMsg{Type: tea.KeySpace},
		tea.WindowSizeMsg{Width: 100, Height: 40},
		tickMsg{},
	}
	for _, msg := range msgs {
		m := newReadyDiffModel()
		m.Update(msg)
		if m.View() == "" {
			t.Errorf("message %v produced an empty view", msg)
		}
	}
}

func TestDiffModelHunkNavigationKeys(t *testing.T) {
	m := newReadyDiffModel()
	if len(m.hunkIdx) < 2 {
		t.Skipf("need at least two hunks, have %d", len(m.hunkIdx))
	}

	first := m.hunkIdx[0]
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("}")})
	if m.hunkIdx[m.currentHunk] == first {
		t.Error("next hunk should move away from the first hunk")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("{")})
	if m.hunkIdx[m.currentHunk] != first {
		t.Error("previous hunk should return to the first hunk")
	}
}

func TestDiffModelScrollBounds(t *testing.T) {
	m := newReadyDiffModel()
	for i := 0; i < 40; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}
	if m.vp.YOffset < 0 {
		t.Errorf("offset went negative: %d", m.vp.YOffset)
	}
	for i := 0; i < 40; i++ {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	}
	if m.vp.YOffset < 0 {
		t.Errorf("offset went negative scrolling up: %d", m.vp.YOffset)
	}
}

func TestDiffModelEmptyDiffView(t *testing.T) {
	m := NewDiffModel(&models.DiffResult{
		VersionA: "1.0.0", VersionB: "1.1.0", HasDiff: true,
	}, config.DiffKeys{})
	m.ready = true
	m.build()

	if m.View() == "" {
		t.Error("expected a view for a diff flag with no content")
	}
}

func TestDiffModelComputeTLDRWithoutResult(t *testing.T) {
	m := NewDiffModel(nil, config.DiffKeys{})
	m.computeTLDR()
	if m.tldrResult != nil {
		t.Error("computeTLDR should do nothing without a result")
	}
}

func TestDiffModelRenderWithoutResult(t *testing.T) {
	// A model with no result must still render a header without panicking.
	m := NewDiffModel(nil, config.DiffKeys{})
	m.renderHeader()
	m.View()
}

func TestBuildKeybindsEveryOverride(t *testing.T) {
	kb := buildKeybinds(config.DiffKeys{
		PrevHunk:             []string{"1"},
		NextHunk:             []string{"2"},
		ScrollDown:           []string{"3"},
		ScrollUp:             []string{"4"},
		ToggleSideBySide:     "5",
		Quit:                 []string{"6"},
		Help:                 "7",
		Export:               "8",
		NextSearch:           "9",
		PrevSearch:           "0",
		ScrollToHunk:         "a",
		ScrollToTopOfHunk:    "b",
		ScrollToBottomOfHunk: "c",
	})

	want := map[string]string{
		"prev_hunk": "1", "next_hunk": "2", "scroll_down": "3", "scroll_up": "4",
		"toggle_side": "5", "quit": "6", "help": "7", "export": "8",
		"next_search": "9", "prev_search": "0", "scroll_to_hunk": "a",
		"scroll_to_top": "b", "scroll_to_bottom": "c",
	}
	for name, key := range want {
		got := kb[name]
		if len(got) == 0 || got[0] != key {
			t.Errorf("%s = %v, want %s", name, got, key)
		}
	}
}

func TestBuildKeybindsKeepsDefaultsForEmptyValues(t *testing.T) {
	kb := buildKeybinds(config.DiffKeys{PrevHunk: []string{}, Help: ""})
	if kb["prev_hunk"][0] != "{" {
		t.Errorf("an empty override should keep the default, got %v", kb["prev_hunk"])
	}
	if kb["help"][0] != "?" {
		t.Errorf("an empty help override should keep the default, got %v", kb["help"])
	}
}

func TestRenderHelpAndHeader(t *testing.T) {
	m := newReadyDiffModel()

	if help := m.renderHelp(); !strings.Contains(help, "Help") {
		t.Errorf("unexpected help text:\n%s", help)
	}
	if header := m.renderHeader(); !strings.Contains(header, "1.0.0") {
		t.Errorf("header should name both versions:\n%s", header)
	}
}
