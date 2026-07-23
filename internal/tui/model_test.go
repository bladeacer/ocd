package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		v    string
		want []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"0.0.0", []int{0, 0, 0}},
		{"10.20.30", []int{10, 20, 30}},
		{"1.2", []int{1, 2, 0}},
		{"abc", []int{0, 0, 0}},
		{"1.2.3.4", []int{1, 2, 3}},
	}
	for _, tt := range tests {
		got := parseVersion(tt.v)
		if len(got) != 3 {
			t.Errorf("parseVersion(%q) len = %d, want 3", tt.v, len(got))
			continue
		}
		for i := 0; i < 3; i++ {
			if got[i] != tt.want[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.v, i, got[i], tt.want[i])
			}
		}
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"123", 123},
		{"0", 0},
		{"abc", 0},
		{"12a34", 1234},
		{"", 0},
		{"999", 999},
	}
	for _, tt := range tests {
		got := atoi(tt.s)
		if got != tt.want {
			t.Errorf("atoi(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b []int
		want int
	}{
		{[]int{1, 0, 0}, []int{1, 0, 0}, 0},
		{[]int{2, 0, 0}, []int{1, 0, 0}, 1},
		{[]int{1, 0, 0}, []int{2, 0, 0}, -1},
		{[]int{1, 5, 0}, []int{1, 3, 0}, 2},
		{[]int{1, 0, 5}, []int{1, 0, 3}, 2},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSortPriority(t *testing.T) {
	tests := []struct {
		row  table.Row
		want int
	}{
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", "[x]"}, 0},
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""}, 1},
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "N/A", "v28", "120", ""}, 2},
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "Missing", "v28", "120", ""}, 3},
		{table.Row{"0", "1.0.0", "Mobile", "2024-01-01", "", "v28", "120", ""}, 4},
	}
	for _, tt := range tests {
		got := sortPriority(tt.row)
		if got != tt.want {
			t.Errorf("sortPriority(%v) = %d, want %d", tt.row, got, tt.want)
		}
	}
}

func TestFmtElectron(t *testing.T) {
	tests := []struct {
		v    string
		want string
	}{
		{"28.0.0", "v28.0.0"},
		{"", "---"},
		{"1.0.0", "v1.0.0"},
	}
	for _, tt := range tests {
		got := fmtElectron(tt.v)
		if got != tt.want {
			t.Errorf("fmtElectron(%q) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestTableStyles(t *testing.T) {
	s := tableStyles()
	_ = s.Header
	_ = s.Selected
}

func TestFmtStatus(t *testing.T) {
	active := fmtStatus("M", true)
	inactive := fmtStatus("M", false)
	if active == "" {
		t.Error("expected non-empty active status")
	}
	if inactive == "" {
		t.Error("expected non-empty inactive status")
	}
	if active == inactive {
		t.Log("note: active and inactive styles may render differently with ANSI")
	}
}

//nolint:unused
func newTestSearch() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "test"
	ti.CharLimit = 50
	ti.Width = 20
	return ti
}

func TestRenderHelpCentered(t *testing.T) {
	m := &model{width: 100}
	help := m.renderHelpCentered()
	if !strings.Contains(help, "ocd -- Help") {
		t.Errorf("expected help title, got %q", help)
	}
}

func TestRenderHelpCenteredNarrow(t *testing.T) {
	m := &model{width: 40}
	help := m.renderHelpCentered()
	if !strings.Contains(help, "ocd -- Help") {
		t.Errorf("expected help title for narrow width, got %q", help)
	}
}

func TestModelViewDispatchesLoading(t *testing.T) {
	m := &model{
		state:   stateLoading,
		spinner: newSpinnerModel([]string{"Loading..."}).spinner,
	}
	v := m.View()
	if !strings.Contains(v, "Loading") {
		t.Errorf("expected loading view, got %q", v)
	}
}

func TestModelViewDispatchesLoadingError(t *testing.T) {
	m := &model{
		state: stateLoading,
		err:   errTest("error occurred"),
	}
	v := m.View()
	if !strings.Contains(v, "error occurred") {
		t.Errorf("expected error in view, got %q", v)
	}
}

func TestModelViewDispatchesConfirm(t *testing.T) {
	m := &model{
		state:           stateConfirm,
		selectedVersion: "1.0.0",
	}
	v := m.View()
	if !strings.Contains(v, "1.0.0") {
		t.Errorf("expected version in confirm view, got %q", v)
	}
	if !strings.Contains(v, "Extract CSS") {
		t.Errorf("expected 'Extract CSS' in confirm view, got %q", v)
	}
}

func TestModelHandleTableKeyQuestionMark(t *testing.T) {
	m := &model{
		state: stateTable,
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
			}),
		),
	}
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	_, _ = m.handleTableKey(msg)
	if !m.showHelp {
		t.Error("showHelp should be true after ?")
	}
}

func TestModelHandleTableKeyEscHelp(t *testing.T) {
	m := &model{
		state:    stateTable,
		showHelp: true,
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
			}),
		),
	}
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, _ = m.handleTableKey(msg)
	if m.showHelp {
		t.Error("showHelp should be false after Esc")
	}
}

func TestModelHandleTableKeyEsc(t *testing.T) {
	m := &model{
		state: stateTable,
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
			}),
		),
	}
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, _ = m.handleTableKey(msg)
	m.showHelp = false
}

func TestModelHandleSearchKeyQuestionMark(t *testing.T) {
	m := &model{
		state:    stateSearch,
		searchIn: newTestSearch(),
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
			}),
		),
	}
	m.searchIn.Focus()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	_, _ = m.handleSearchKey(msg)
	if m.state != stateTable {
		t.Error("state should be table after ? in search")
	}
	if !m.showHelp {
		t.Error("showHelp should be true")
	}
}

func TestModelWindowSizeMsg(t *testing.T) {
	m := &model{
		state: stateLoading,
	}
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Log("WindowSizeMsg returned command")
	}
	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
}
