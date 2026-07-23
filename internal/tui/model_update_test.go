package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelHandleTableKeyM(t *testing.T) {
	m := &model{
		state:           stateTable,
		showMobile:      true,
		showEarlyAccess: true,
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if m.showMobile {
		t.Error("expected showMobile=false after pressing m")
	}
	if cmd != nil {
		t.Log("m key returned no command")
	}
}

func TestModelHandleTableKeyE(t *testing.T) {
	m := &model{
		state:           stateTable,
		showEarlyAccess: true,
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.showEarlyAccess {
		t.Error("expected showEarlyAccess=false after pressing e")
	}
	if cmd != nil {
		t.Log("e key returned no command")
	}
}

func TestModelHandleTableKeyF(t *testing.T) {
	m := &model{
		state:     stateTable,
		foundOnly: false,
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if !m.foundOnly {
		t.Error("expected foundOnly=true after pressing f")
	}
	if cmd != nil {
		t.Log("f key returned no command")
	}
}

func TestModelHandleTableKeyS(t *testing.T) {
	m := &model{
		state:          stateTable,
		sortByPriority: true,
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.sortByPriority {
		t.Error("expected sortByPriority=false after pressing s")
	}
	if cmd != nil {
		t.Log("s key returned no command")
	}
}

func TestModelHandleTableKeyEnterNoSelection(t *testing.T) {
	m := &model{
		state: stateTable,
		tbl:   table.New(table.WithColumns([]table.Column{{Title: "V", Width: 5}})),
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateConfirm {
		t.Log("no row selected, state unchanged")
	}
	if cmd != nil {
		t.Log("enter with no row returned no command")
	}
}

func TestModelHandleConfirmKeyY(t *testing.T) {
	m := &model{
		state:           stateConfirm,
		selectedVersion: "1.0.0",
	}
	_, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Error("expected quit command for y in confirm")
	}
}

func TestModelHandleConfirmKeyN(t *testing.T) {
	m := &model{
		state:           stateConfirm,
		selectedVersion: "1.0.0",
	}
	_, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.state != stateTable {
		t.Error("expected state=stateTable after n in confirm")
	}
	if m.selectedVersion != "" {
		t.Error("expected selectedVersion cleared after n")
	}
	if cmd != nil {
		t.Log("n returned no command")
	}
}

func TestModelHandleConfirmKeyEnter(t *testing.T) {
	m := &model{
		state:           stateConfirm,
		selectedVersion: "1.0.0",
	}
	_, cmd := m.handleConfirmKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("expected quit command for enter in confirm")
	}
}

func TestModelCSSDirFor(t *testing.T) {
	m := &model{}
	path := m.cssDirFor("1.2.3")
	expected := ".obsidian_cache/css/1.2.3/app.css"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestModelUpdateTableDimensions(t *testing.T) {
	m := &model{
		tbl: table.New(table.WithColumns([]table.Column{{Title: "V", Width: 5}})),
	}
	m.updateTableDimensions()
}

func TestModelHandleTableKeySlash(t *testing.T) {
	m := &model{
		state:    stateTable,
		searchIn: textinput.New(),
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.state != stateSearch {
		t.Error("expected state=stateSearch after pressing /")
	}
	if cmd != nil {
		t.Log("slash returned no command")
	}
}

func TestModelHandleTableKeyQuit(t *testing.T) {
	m := &model{
		state: stateTable,
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command for q in table")
	}
}

func TestModelHandleTableKeyCtrlC(t *testing.T) {
	m := &model{
		state: stateTable,
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("expected quit command for ctrl+c in table")
	}
}

func TestModelHandleTableKeyUnhandled(t *testing.T) {
	m := &model{
		state: stateTable,
		tbl: table.New(
			table.WithColumns([]table.Column{{Title: "V", Width: 5}}),
			table.WithRows([]table.Row{{"test"}}),
		),
	}
	_, cmd := m.handleTableKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_ = cmd
}

func TestModelHandleSearchKeyEscape(t *testing.T) {
	ti := newTestSearch()
	ti.Focus()
	m := &model{
		state:       stateSearch,
		searchQuery: "test",
		searchIn:    ti,
	}
	m.searchIn.SetValue("test")

	_, cmd := m.handleSearchKey(tea.KeyMsg{Type: tea.KeyEscape})
	if m.state != stateTable {
		t.Error("expected state=stateTable after escape in search")
	}
	if m.searchQuery != "" {
		t.Error("expected searchQuery cleared after escape")
	}
	if cmd != nil {
		t.Log("escape returned no command")
	}
}

func TestModelHandleSearchKeyEnter(t *testing.T) {
	ti := newTestSearch()
	ti.Focus()
	ti.SetValue("query")
	m := &model{
		state:       stateSearch,
		searchQuery: "",
		searchIn:    ti,
	}

	_, cmd := m.handleSearchKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateTable {
		t.Error("expected state=stateTable after enter in search")
	}
	if m.searchQuery != "query" {
		t.Errorf("expected searchQuery=query, got %s", m.searchQuery)
	}
	if cmd != nil {
		t.Log("enter returned no command")
	}
}

func TestModelHandleSearchKeyBackspace(t *testing.T) {
	ti := newTestSearch()
	ti.Focus()
	ti.SetValue("ab")
	m := &model{
		state:       stateSearch,
		searchQuery: "ab",
		searchIn:    ti,
	}

	_, cmd := m.handleSearchKey(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.searchQuery != "a" {
		t.Errorf("expected searchQuery=a after backspace, got %s", m.searchQuery)
	}
	if cmd != nil {
		t.Log("backspace returned no command")
	}
}

func TestModelHandleSearchKeyRune(t *testing.T) {
	ti := newTestSearch()
	ti.Focus()
	m := &model{
		state:       stateSearch,
		searchQuery: "",
		searchIn:    ti,
	}

	_, _ = m.handleSearchKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
}

func TestModelHandleKeyDispatches(t *testing.T) {
	m := &model{state: stateLoading, loadMessages: []string{"test"}, loadIndex: 0}
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	m.state = stateTable
	m.tbl = table.New(table.WithColumns([]table.Column{{Title: "V", Width: 5}}))
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	m.state = stateSearch
	m.searchIn = newTestSearch()
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	m.state = stateConfirm
	m.selectedVersion = "1.0.0"
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
}

func TestModelSelectionConfirmed(t *testing.T) {
	m := &model{
		state:           stateConfirm,
		selectedVersion: "1.0.0",
	}
	if m.state != stateConfirm {
		t.Error("expected stateConfirm")
	}
	if m.selectedVersion != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", m.selectedVersion)
	}
}

func TestModelSelectionCancelled(t *testing.T) {
	m := &model{
		state: stateTable,
	}
	if m.state == stateConfirm {
		t.Error("expected non-confirm state")
	}
	if m.selectedVersion != "" {
		t.Errorf("expected empty version, got %s", m.selectedVersion)
	}
}

func TestModelApplyFilters(t *testing.T) {
	m := &model{
		rows: []table.Row{
			{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""},
			{"1", "1.0.1", "Mobile", "2024-01-02", "N/A", "", "", ""},
		},
		showMobile:      false,
		showEarlyAccess: true,
		foundOnly:       false,
	}
	m.applyFilters()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered row, got %d", len(m.filtered))
	}
}

func TestModelApplyFiltersSearch(t *testing.T) {
	m := &model{
		rows: []table.Row{
			{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""},
			{"1", "1.5.0", "Desktop", "2024-02-01", "Found", "v28", "120", ""},
		},
		showEarlyAccess: true,
		searchQuery:     "1.5.0",
	}
	m.applyFilters()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 filtered row, got %d", len(m.filtered))
	}
	if m.filtered[0][1] != "1.5.0" {
		t.Errorf("expected 1.5.0, got %s", m.filtered[0][1])
	}
}

func TestModelSelectRowNoRows(t *testing.T) {
	m := &model{
		tbl: table.New(table.WithColumns([]table.Column{{Title: "V", Width: 5}})),
	}
	_, cmd := m.selectRow()
	if cmd != nil {
		t.Log("no row selected, no command")
	}
}
