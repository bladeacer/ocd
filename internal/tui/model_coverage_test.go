//nolint:goconst
package tui

import (
	"testing"

	"github.com/bladeacer/ocd/internal/models"
	"github.com/charmbracelet/bubbles/table"
)

func makeTestResult() *models.FetchResult {
	return &models.FetchResult{
		RSS: []models.RSSVersion{
			{Version: "1.2.0", Type: models.Desktop, Date: "2024-03-15", Electron: "28", IsEarly: false},
			{Version: "1.1.0", Type: models.Desktop, Date: "2024-02-01", Electron: "27", IsEarly: true},
			{Version: "1.0.0", Type: models.Mobile, Date: "2024-01-10", Electron: "", IsEarly: false},
			{Version: "1.2.0", Type: models.Mobile, Date: "2024-03-20", Electron: "28", IsEarly: false},
		},
		Docker: []models.DockerTag{
			{Version: "1.2.0", Tag: "latest"},
		},
		Electron: models.ElectronMap{
			"28": "120",
			"27": "119",
		},
	}
}

func TestBuildRows(t *testing.T) {
	m := &model{result: makeTestResult()}
	m.buildRows()
	if len(m.rows) == 0 {
		t.Fatal("expected rows to be populated")
	}
	found := false
	for _, row := range m.rows {
		if row[1] == "1.2.0" && row[2] == "Desktop" {
			found = true
			if row[4] != "Found" {
				t.Errorf("expected Docker status Found for Desktop 1.2.0, got %s", row[4])
			}
			if row[5] != "v28" {
				t.Errorf("expected Electron v28, got %s", row[5])
			}
			if row[6] != "120" {
				t.Errorf("expected Chromium 120, got %s", row[6])
			}
		}
	}
	if !found {
		t.Error("expected Desktop 1.2.0 row in buildRows output")
	}
}

func TestBuildRowsMobileDedup(t *testing.T) {
	m := &model{result: makeTestResult()}
	m.buildRows()
	for _, row := range m.rows {
		if row[1] == "1.0.0" && row[2] != "Mobile" {
			t.Errorf("expected 1.0.0 to be Mobile type, got %s", row[2])
		}
	}
}

func TestUpdateTableDataSortByPriority(t *testing.T) {
	cols := []table.Column{
		{Title: "ID", Width: 5}, {Title: "Version", Width: 14},
		{Title: "Type", Width: 8}, {Title: "Date", Width: 12},
		{Title: "Docker", Width: 8}, {Title: "Electron", Width: 10},
		{Title: "Chromium", Width: 10}, {Title: "CSS", Width: 6},
	}
	m := &model{
		rows: []table.Row{
			{"0", "1.0.0", "Desktop", "2024-01-01", "Missing", "", "", ""},
			{"1", "1.1.0", "Desktop", "2024-02-01", "Found", "v28", "120", ""},
			{"2", "1.2.0", "Desktop", "2024-03-01", "Found", "v28", "120", "[x]"},
		},
		sortByPriority: true,
		tbl:            table.New(table.WithColumns(cols)),
	}
	m.updateTableData()
	rows := m.tbl.Rows()
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0][1] != "1.2.0" {
		t.Errorf("expected highest priority (CSS) row 1.2.0 first, got %s", rows[0][1])
	}
}

func TestUpdateTableDataSortNoPriority(t *testing.T) {
	m := &model{
		rows: []table.Row{
			{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""},
			{"1", "1.2.0", "Desktop", "2024-03-01", "Found", "v28", "120", ""},
			{"2", "1.1.0", "Desktop", "2024-02-01", "Found", "v28", "120", ""},
		},
		sortByPriority: false,
		tbl: table.New(table.WithColumns([]table.Column{
			{Title: "ID", Width: 5}, {Title: "Version", Width: 14},
			{Title: "Type", Width: 8}, {Title: "Date", Width: 12},
			{Title: "Docker", Width: 8}, {Title: "Electron", Width: 10},
			{Title: "Chromium", Width: 10}, {Title: "CSS", Width: 6},
		})),
	}
	m.updateTableData()
	rows := m.tbl.Rows()
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0][1] != "1.2.0" {
		t.Errorf("expected highest version 1.2.0 first, got %s", rows[0][1])
	}
	if rows[2][1] != "1.0.0" {
		t.Errorf("expected lowest version 1.0.0 last, got %s", rows[2][1])
	}
}

func TestUpdateTableDataFilteredIsNil(t *testing.T) {
	m := &model{
		rows: []table.Row{
			{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""},
		},
		sortByPriority: true,
		tbl: table.New(table.WithColumns([]table.Column{
			{Title: "ID", Width: 5}, {Title: "Version", Width: 14},
			{Title: "Type", Width: 8}, {Title: "Date", Width: 12},
			{Title: "Docker", Width: 8}, {Title: "Electron", Width: 10},
			{Title: "Chromium", Width: 10}, {Title: "CSS", Width: 6},
		})),
	}
	m.filtered = nil
	m.updateTableData()
	rows := m.tbl.Rows()
	if len(rows) != 1 {
		t.Fatalf("expected 1 row when filtered is nil, got %d", len(rows))
	}
	if rows[0][1] != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", rows[0][1])
	}
}

func TestSelectRow(t *testing.T) {
	m := &model{
		state: stateTable,
		tbl: table.New(
			table.WithColumns([]table.Column{{Title: "ID", Width: 5}, {Title: "Version", Width: 14},
				{Title: "Type", Width: 8}, {Title: "Date", Width: 12}, {Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10}, {Title: "Chromium", Width: 10}, {Title: "CSS", Width: 6}}),
			table.WithRows([]table.Row{
				{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""},
				{"1", "1.1.0", "Desktop", "2024-02-01", "Found", "v28", "120", ""},
			}),
			table.WithFocused(true),
		),
	}
	_, cmd := m.selectRow()
	if m.selectedVersion != "1.0.0" {
		t.Errorf("expected selectedVersion 1.0.0, got %s", m.selectedVersion)
	}
	if m.state != stateConfirm {
		t.Errorf("expected state stateConfirm, got %v", m.state)
	}
	if cmd != nil {
		t.Log("selectRow returned no command")
	}
}

func TestSelectRowSecondRow(t *testing.T) {
	m := &model{
		state: stateTable,
		tbl: table.New(
			table.WithColumns([]table.Column{{Title: "ID", Width: 5}, {Title: "Version", Width: 14},
				{Title: "Type", Width: 8}, {Title: "Date", Width: 12}, {Title: "Docker", Width: 8},
				{Title: "Electron", Width: 10}, {Title: "Chromium", Width: 10}, {Title: "CSS", Width: 6}}),
			table.WithRows([]table.Row{
				{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""},
				{"1", "1.1.0", "Desktop", "2024-02-01", "Found", "v28", "120", ""},
			}),
			table.WithFocused(true),
		),
	}
	m.tbl.MoveDown(1)
	_, cmd := m.selectRow()
	if m.selectedVersion != "1.1.0" {
		t.Errorf("expected selectedVersion 1.1.0 after MoveDown, got %s", m.selectedVersion)
	}
	if m.state != stateConfirm {
		t.Errorf("expected state stateConfirm, got %v", m.state)
	}
	if cmd != nil {
		t.Log("selectRow second row returned no command")
	}
}

func TestLoadTicker(t *testing.T) {
	m := &model{}
	msg := m.loadTicker()
	if _, ok := msg.(tickMsg); !ok {
		t.Error("expected loadTicker to return tickMsg")
	}
}

func TestApplyFiltersHideEarlyAccess(t *testing.T) {
	m := &model{
		result: &models.FetchResult{
			RSS: []models.RSSVersion{
				{Version: "1.0.0", IsEarly: false},
				{Version: "1.1.0", IsEarly: true},
			},
		},
		rows: []table.Row{
			{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""},
			{"1", "1.1.0", "Desktop", "2024-02-01", "Found", "v28", "120", ""},
		},
		showEarlyAccess: false,
	}
	m.applyFilters()
	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered row (early access hidden), got %d", len(m.filtered))
	}
	if m.filtered[0][1] != "1.0.0" {
		t.Errorf("expected 1.0.0 to be kept, got %s", m.filtered[0][1])
	}
}
