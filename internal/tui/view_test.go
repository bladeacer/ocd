package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelTableContentView(t *testing.T) {
	m := &model{
		state:           stateTable,
		showMobile:      true,
		showEarlyAccess: true,
	}
	v := m.tableContentView()
	if !strings.Contains(v, "ocd -- Obsidian CSS Diff") {
		t.Errorf("expected title in table content view, got %q", v)
	}
}

func TestModelConfirmView(t *testing.T) {
	m := &model{
		state:           stateConfirm,
		selectedVersion: "1.0.0",
	}
	v := m.confirmView()
	if !strings.Contains(v, "1.0.0") {
		t.Errorf("expected version in confirm view, got %q", v)
	}
	if !strings.Contains(v, "Extract CSS") {
		t.Errorf("expected 'Extract CSS' in confirm view, got %q", v)
	}
}

func TestModelLoadingView(t *testing.T) {
	m := &model{
		state:        stateLoading,
		loadMessages: []string{"Loading test..."},
		loadIndex:    0,
	}
	v := m.loadingView()
	if !strings.Contains(v, "elapsed:") {
		t.Errorf("expected 'elapsed:' in loading view, got %q", v)
	}
}

func TestModelLoadingViewError(t *testing.T) {
	m := &model{
		state: stateLoading,
		err:   errTest("something broke"),
	}
	v := m.loadingView()
	if !strings.Contains(v, "something broke") {
		t.Errorf("expected error in loading view, got %q", v)
	}
}

func TestModelViewDispatch(t *testing.T) {
	m := &model{state: stateLoading, loadMessages: []string{"test"}, loadIndex: 0}
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view")
	}

	m.state = stateTable
	v = m.View()
	if v == "" {
		t.Error("expected non-empty table view")
	}
}

func TestModelHandleLoadingKey(t *testing.T) {
	m := &model{state: stateLoading}
	_, cmd := m.handleLoadingKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command for q during loading")
	}

	_, cmd = m.handleLoadingKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected no command for unrelated key during loading")
	}
}

func TestModelFooter(t *testing.T) {
	m := &model{
		showMobile:      true,
		showEarlyAccess: false,
		foundOnly:       true,
		sortByPriority:  false,
	}
	f := m.footerView()
	plain := stripANSI(f)
	if !strings.Contains(plain, "M") {
		t.Errorf("expected M in footer, got %q", plain)
	}
	if !strings.Contains(plain, "E") {
		t.Errorf("expected E in footer, got %q", plain)
	}
	if !strings.Contains(plain, "up/down") {
		t.Errorf("expected key hints in footer, got %q", plain)
	}
	if !strings.Contains(plain, "\n") {
		t.Errorf("expected newline in footer for line break, got %q", plain)
	}
}

func TestModelFilterToggleDefaults(t *testing.T) {
	m := &model{
		showMobile:      false,
		showEarlyAccess: false,
		foundOnly:       false,
		state:           stateTable,
		loadMessages:    []string{"test"},
		loadIndex:       0,
	}
	v := m.View()
	if v == "" {
		t.Error("expected non-empty view")
	}
}

func TestModelSearchState(t *testing.T) {
	m := &model{
		state:      stateSearch,
		showMobile: true,
		searchIn:   newTestSearch(),
	}
	v := m.View()
	if v == "" {
		t.Error("expected non-empty search view")
	}
}
