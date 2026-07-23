package tui

import (
	"testing"

	"github.com/bladeacer/ocd/internal/models"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPickerSelectCurrentFirstStep(t *testing.T) {
	m := &pickerModel{
		step: stepPickFirst,
		result: &models.FetchResult{
			RSS:    []models.RSSVersion{},
			Docker: []models.DockerTag{},
		},
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
			table.WithRows([]table.Row{
				{"1.0.0", "Desktop", "2024-01-01", "Found", "v25"},
			}),
			table.WithFocused(true),
		),
	}
	_, _ = m.selectCurrent()
	if m.firstVer != "1.0.0" {
		t.Errorf("firstVer = %q, want %q", m.firstVer, "1.0.0")
	}
	if m.step != stepPickSecond {
		t.Errorf("step = %d, want %d", m.step, stepPickSecond)
	}
}

func TestPickerSelectCurrentSecondStep(t *testing.T) {
	m := &pickerModel{
		step:     stepPickSecond,
		firstVer: "1.0.0",
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
			table.WithRows([]table.Row{
				{"1.0.0", "Desktop", "2024-01-01", "Found", "v25"},
				{"1.1.0", "Desktop", "2024-02-01", "Found", "v26"},
			}),
			table.WithFocused(true),
		),
	}
	_, _ = m.selectCurrent()
	if m.secondVer != "1.0.0" {
		t.Errorf("secondVer = %q, want %q", m.secondVer, "1.0.0")
	}
	if !m.done {
		t.Error("done should be true")
	}
}

func TestPickerSelectCurrentNoRow(t *testing.T) {
	m := &pickerModel{
		step: stepPickFirst,
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
	}
	_, _ = m.selectCurrent()
	if m.firstVer != "" {
		t.Errorf("firstVer = %q, want empty", m.firstVer)
	}
}

func TestPickerBuildRows(t *testing.T) {
	m := &pickerModel{
		step: stepPickFirst,
		result: &models.FetchResult{
			RSS: []models.RSSVersion{
				{Version: "1.0.0", Type: models.Desktop, Date: "2024-01-01", Electron: "25"},
				{Version: "1.1.0", Type: models.Mobile, Date: "2024-02-01", Electron: "26"},
			},
			Docker: []models.DockerTag{
				{Version: "1.0.0"},
			},
		},
	}
	m.buildRows()
	if len(m.rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(m.rows))
	}
	if m.rows[0][0] != "1.1.0" {
		t.Errorf("rows[0][0] = %q, want %q", m.rows[0][0], "1.1.0")
	}
	if m.rows[0][3] != "N/A" {
		t.Errorf("rows[0][3] = %q, want %q", m.rows[0][3], "N/A")
	}
}

func TestPickerBuildRowsFiltersFirst(t *testing.T) {
	m := &pickerModel{
		step:     stepPickSecond,
		firstVer: "1.0.0",
		result: &models.FetchResult{
			RSS: []models.RSSVersion{
				{Version: "1.0.0", Type: models.Desktop, Date: "2024-01-01"},
				{Version: "1.1.0", Type: models.Desktop, Date: "2024-02-01"},
			},
			Docker: []models.DockerTag{},
		},
	}
	m.buildRows()
	if len(m.rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(m.rows))
	}
	if m.rows[0][0] != "1.1.0" {
		t.Errorf("rows[0][0] = %q, want %q", m.rows[0][0], "1.1.0")
	}
}

func TestPickerApplyFilterHideMobile(t *testing.T) {
	m := &pickerModel{
		showMobile: false,
		rows: []table.Row{
			{"1.0.0", "Desktop", "2024-01-01", "Found", "v25"},
			{"1.1.0", "Mobile", "2024-02-01", "N/A", "v26"},
			{"1.2.0", "Desktop", "2024-03-01", "Found", "v27"},
		},
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
	}
	m.applyFilter()
	if len(m.tbl.Rows()) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(m.tbl.Rows()))
	}
	for _, r := range m.tbl.Rows() {
		if r[1] == "Mobile" {
			t.Error("Mobile row should be filtered out")
		}
	}
}

func TestPickerApplyFilterSearch(t *testing.T) {
	m := &pickerModel{
		searchQ: "1.0",
		search:  textinput.New(),
		rows: []table.Row{
			{"1.0.0", "Desktop", "2024-01-01", "Found", "v25"},
			{"1.1.0", "Desktop", "2024-02-01", "Found", "v26"},
			{"2.0.0", "Mobile", "2024-03-01", "N/A", "v27"},
		},
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
	}
	m.applyFilter()
	if len(m.tbl.Rows()) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(m.tbl.Rows()))
	}
}

func TestPickerHandleKeySlash(t *testing.T) {
	m := &pickerModel{
		search: textinput.New(),
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
		showMobile: true,
	}
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	_, _ = m.handleKey(msg)
	if !m.searchMode {
		t.Error("searchMode should be true after pressing /")
	}
	if !m.search.Focused() {
		t.Error("search input should be focused")
	}
}

func TestPickerHandleKeyM(t *testing.T) {
	m := &pickerModel{
		search: textinput.New(),
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
		showMobile: false,
		rows: []table.Row{
			{"1.0.0", "Desktop", "2024-01-01", "Found", "v25"},
		},
	}
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	_, _ = m.handleKey(msg)
	if !m.showMobile {
		t.Error("showMobile should be true after pressing m")
	}
}

func TestPickerHandleKeyEnter(t *testing.T) {
	m := &pickerModel{
		step:   stepPickFirst,
		search: textinput.New(),
		result: &models.FetchResult{
			RSS:    []models.RSSVersion{},
			Docker: []models.DockerTag{},
		},
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
			table.WithRows([]table.Row{
				{"1.0.0", "Desktop", "2024-01-01", "Found", "v25"},
			}),
			table.WithFocused(true),
		),
	}
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = m.handleKey(msg)
	if m.firstVer != "1.0.0" {
		t.Errorf("firstVer = %q, want %q", m.firstVer, "1.0.0")
	}
}

func TestPickerHandleSearchKeyEscape(t *testing.T) {
	m := &pickerModel{
		searchMode: true,
		searchQ:    "test",
		search:     textinput.New(),
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
	}
	m.search.SetValue("test")
	msg := tea.KeyMsg{Type: tea.KeyEscape}
	_, _ = m.handleSearchKey(msg)
	if m.searchMode {
		t.Error("searchMode should be false")
	}
	if m.searchQ != "" {
		t.Errorf("searchQ = %q, want empty", m.searchQ)
	}
}

func TestPickerHandleSearchKeyEnter(t *testing.T) {
	m := &pickerModel{
		searchMode: true,
		search:     textinput.New(),
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
	}
	m.search.SetValue("1.0")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, _ = m.handleSearchKey(msg)
	if m.searchMode {
		t.Error("searchMode should be false")
	}
	if m.searchQ != "1.0" {
		t.Errorf("searchQ = %q, want %q", m.searchQ, "1.0")
	}
}

func TestPickerHandleSearchKeyBackspace(t *testing.T) {
	m := &pickerModel{
		searchMode: true,
		search:     textinput.New(),
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
	}
	m.search.SetValue("ab")
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	_, _ = m.handleSearchKey(msg)
	if m.search.Value() != "a" {
		t.Errorf("search value = %q, want %q", m.search.Value(), "a")
	}
	if m.searchQ != "a" {
		t.Errorf("searchQ = %q, want %q", m.searchQ, "a")
	}
}

func TestPickerHandleSearchKeyCharacter(t *testing.T) {
	m := &pickerModel{
		searchMode: true,
		search:     textinput.New(),
		tbl: table.New(
			table.WithColumns([]table.Column{
				{Title: "V", Width: 14},
				{Title: "Type", Width: 10},
				{Title: "Date", Width: 12},
				{Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10},
			}),
		),
	}
	m.search.Focus()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	_, _ = m.handleSearchKey(msg)
	if m.searchQ != "x" {
		t.Errorf("searchQ = %q, want %q", m.searchQ, "x")
	}
}

func TestPickerCSSExtracted(t *testing.T) {
	m := &pickerModel{}
	result := m.cssExtracted("nonexistent")
	if result {
		t.Error("cssExtracted should return false for non-existent file")
	}
}
