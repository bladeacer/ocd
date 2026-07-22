package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bladeacer/obsi-css-diff/internal/models"
)

type Selection struct {
	Version string
}

type viewState int

const (
	stateLoading viewState = iota
	stateTable
	stateConfirm
)

type model struct {
	result  *models.FetchResult
	state   viewState
	spinner spinner.Model
	tbl     table.Model

	rows     []table.Row
	filtered []table.Row

	showMobile      bool
	showEarlyAccess bool
	foundOnly       bool
	sortByPriority  bool
	searchQuery     string

	selectedVersion string

	ready bool
	err   error

	width  int
	height int
}

func New(result *models.FetchResult) *model {
	s := spinner.New()
	s.Style = spinnerStyle

	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Version", Width: 14},
		{Title: "Type", Width: 10},
		{Title: "Date", Width: 12},
		{Title: "Docker", Width: 10},
		{Title: "Electron", Width: 12},
		{Title: "Chromium", Width: 12},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(tableStyles())

	m := &model{
		result:          result,
		state:           stateTable,
		spinner:         s,
		tbl:             t,
		showMobile:      true,
		showEarlyAccess: true,
		sortByPriority:  true,
		ready:           true,
	}

	m.buildRows()
	m.applyFilters()
	return m
}

func (m *model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateTableDimensions()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateTable:
		return m.handleTableKey(msg)
	case stateConfirm:
		return m.handleConfirmKey(msg)
	}
	return m, nil
}

func (m *model) handleTableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "enter":
		return m.selectRow()

	case "m":
		m.showMobile = !m.showMobile
		m.applyFilters()

	case "e":
		m.showEarlyAccess = !m.showEarlyAccess
		m.applyFilters()

	case "f":
		m.foundOnly = !m.foundOnly
		m.applyFilters()

	case "s":
		m.sortByPriority = !m.sortByPriority
		m.applyFilters()
	}

	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	return m, cmd
}

func (m *model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, tea.Quit
	case "n", "escape", "q":
		m.selectedVersion = ""
		m.state = stateTable
		return m, nil
	}
	return m, nil
}

func (m *model) selectRow() (tea.Model, tea.Cmd) {
	row := m.tbl.SelectedRow()
	if row == nil {
		return m, nil
	}

	version := row[1]
	m.selectedVersion = version
	m.state = stateConfirm
	return m, nil
}

func (m *model) applyFilters() {
	var filtered []table.Row

	dockerMap := make(map[string]bool)
	for _, dt := range m.result.Docker {
		dockerMap[dt.Version] = true
	}

	for _, row := range m.rows {
		version := row[1]
		vType := row[2]
		status := row[4]

		if !m.showMobile && vType == "Mobile" {
			continue
		}
		if !m.showEarlyAccess {
			for _, v := range m.result.RSS {
				if v.Version == version && v.IsEarly {
					continue
				}
			}
		}
		if m.foundOnly && status != statusFoundStr {
			continue
		}
		if m.searchQuery != "" {
			searchable := strings.ToLower(strings.Join(row, " "))
			if !strings.Contains(searchable, strings.ToLower(m.searchQuery)) {
				continue
			}
		}

		filtered = append(filtered, row)
	}

	m.filtered = filtered
	m.updateTableData()
}

func (m *model) updateTableData() {
	rows := m.filtered
	if rows == nil {
		rows = m.rows
	}

	sorted := make([]table.Row, len(rows))
	copy(sorted, rows)

	sort.Slice(sorted, func(i, j int) bool {
		vI := parseVersion(sorted[i][1])
		vJ := parseVersion(sorted[j][1])

		if m.sortByPriority {
			prioI := sortPriority(sorted[i])
			prioJ := sortPriority(sorted[j])
			if prioI != prioJ {
				return prioI < prioJ
			}
		}
		return compareVersions(vI, vJ) > 0
	})

	m.tbl.SetRows(sorted)
}

func parseVersion(v string) []int {
	parts := strings.Split(v, ".")
	ints := make([]int, 3)
	for i, p := range parts {
		if i < 3 {
			ints[i] = atoi(p)
		}
	}
	return ints
}

func atoi(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func compareVersions(a, b []int) int {
	for i := 0; i < 3; i++ {
		va := 0
		vb := 0
		if i < len(a) {
			va = a[i]
		}
		if i < len(b) {
			vb = b[i]
		}
		if va != vb {
			return va - vb
		}
	}
	return 0
}

func sortPriority(row table.Row) int {
	switch row[4] {
	case statusFoundStr:
		return 0
	case statusNAStr:
		return 1
	case statusMissingStr:
		return 2
	}
	return 3
}

const (
	statusFoundStr   = "Found"
	statusMissingStr = "Missing"
	statusNAStr      = "N/A"
)

func (m *model) updateTableDimensions() {
	availW := m.width - 6
	if availW < 60 {
		availW = 60
	}

	idW := 5
	verW := 14
	typeW := 10
	dateW := 12
	statW := 10
	elW := 12
	crW := 12

	totalMin := idW + verW + typeW + dateW + statW + elW + crW + 2
	if availW < totalMin {
		statW = 8
		elW = 10
		crW = 10
	}

	m.tbl.SetColumns([]table.Column{
		{Title: "ID", Width: idW},
		{Title: "Version", Width: verW},
		{Title: "Type", Width: typeW},
		{Title: "Date", Width: dateW},
		{Title: "Docker", Width: statW},
		{Title: "Electron", Width: elW},
		{Title: "Chromium", Width: crW},
	})
}

func (m *model) buildRows() {
	var rows []table.Row

	dockerMap := make(map[string]bool)
	for _, dt := range m.result.Docker {
		dockerMap[dt.Version] = true
	}

	electronMap := m.result.Electron

	rssVersions := m.result.RSS
	sort.Slice(rssVersions, func(i, j int) bool {
		return compareVersions(
			parseVersion(rssVersions[i].Version),
			parseVersion(rssVersions[j].Version),
		) > 0
	})

	seen := make(map[string]bool)

	for idx, rss := range rssVersions {
		v := rss.Version
		key := v + "||" + string(rss.Type)
		if seen[key] {
			continue
		}
		seen[key] = true

		elV := rss.Electron
		crV := "---"
		if elV != "" {
			if ch, ok := electronMap[elV]; ok {
				crV = ch
			}
		}

		hasDocker := dockerMap[v]
		stat := statusNAStr
		if rss.Type == models.Desktop {
			if hasDocker {
				stat = statusFoundStr
			} else {
				stat = statusMissingStr
			}
		}

		date := rss.Date
		if len(date) > 10 {
			date = date[:10]
		}

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", idx),
			v,
			string(rss.Type),
			date,
			stat,
			fmtElectron(elV),
			crV,
		})
	}

	m.rows = rows
	m.filtered = rows
}

func fmtElectron(v string) string {
	if v == "" {
		return "---"
	}
	return "v" + v
}
