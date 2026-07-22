package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bladeacer/ocd/internal/models"
	"github.com/bladeacer/ocd/internal/sources"
)

type Selection struct {
	Version string
}

type viewState int

const (
	stateLoading viewState = iota
	stateTable
	stateSearch
	stateConfirm
)

type model struct {
	fetcher  *sources.Fetcher
	result   *models.FetchResult
	force    bool
	state    viewState
	spinner  spinner.Model
	tbl      table.Model
	searchIn textinput.Model

	startTime    time.Time
	loadMessages []string
	loadIndex    int

	rows     []table.Row
	filtered []table.Row

	showMobile      bool
	showEarlyAccess bool
	foundOnly       bool
	sortByPriority  bool
	searchQuery     string

	selectedVersion string

	err   error
	width int
}

type dataLoadedMsg struct {
	result *models.FetchResult
}

type tickMsg struct{}

func New(f *sources.Fetcher, force bool) *model {
	s := spinner.New()
	s.Style = spinnerStyle
	s.Spinner = spinner.Pulse

	columns := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Version", Width: 14},
		{Title: "Type", Width: 8},
		{Title: "Date", Width: 12},
		{Title: "Docker", Width: 8},
		{Title: "Electron", Width: 10},
		{Title: "Chromium", Width: 10},
		{Title: "CSS", Width: 6},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	t.SetStyles(tableStyles())

	ti := textinput.New()
	ti.Placeholder = "Type to filter versions..."
	ti.CharLimit = 50
	ti.Width = 40

	m := &model{
		fetcher:         f,
		force:           force,
		state:           stateLoading,
		spinner:         s,
		tbl:             t,
		searchIn:        ti,
		startTime:       time.Now(),
		showMobile:      true,
		showEarlyAccess: true,
		sortByPriority:  true,
		loadMessages: []string{
			"Fetching RSS changelog...",
			"Checking Docker Hub tags...",
			"Loading Electron-Chromium map...",
			"Compiling version data...",
			"Almost done...",
		},
	}

	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchData, m.loadTicker)
}

func (m *model) loadTicker() tea.Msg {
	time.Sleep(2 * time.Second)
	return tickMsg{}
}

func (m *model) fetchData() tea.Msg {
	result := m.fetcher.FetchAll(m.force)
	return dataLoadedMsg{result: result}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.updateTableDimensions()
		return m, nil

	case dataLoadedMsg:
		m.result = msg.result
		if msg.result.Error != nil {
			m.err = msg.result.Error
			return m, nil
		}
		m.buildRows()
		m.applyFilters()
		m.state = stateTable
		return m, nil

	case tickMsg:
		if m.state == stateLoading {
			m.loadIndex++
			if m.loadIndex >= len(m.loadMessages) {
				m.loadIndex = len(m.loadMessages) - 1
			}
			return m, m.loadTicker
		}
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
	case stateLoading:
		return m.handleLoadingKey(msg)
	case stateTable:
		return m.handleTableKey(msg)
	case stateSearch:
		return m.handleSearchKey(msg)
	case stateConfirm:
		return m.handleConfirmKey(msg)
	}
	return m, nil
}

func (m *model) handleLoadingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "q" || msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) handleTableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "enter":
		return m.selectRow()

	case "/":
		m.state = stateSearch
		m.searchIn.Focus()
		m.searchIn.SetValue("")
		return m, nil

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

func (m *model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "escape", "ctrl+c":
		m.state = stateTable
		m.searchQuery = ""
		m.searchIn.SetValue("")
		m.searchIn.Blur()
		m.applyFilters()
		return m, nil

	case "enter", "tab":
		m.state = stateTable
		m.searchQuery = m.searchIn.Value()
		m.searchIn.Blur()
		m.applyFilters()
		return m, nil

	case "backspace":
		val := m.searchIn.Value()
		if len(val) > 0 {
			m.searchIn.SetValue(val[:len(val)-1])
		}
		m.searchQuery = m.searchIn.Value()
		m.applyFilters()
		return m, nil
	}

	if len(msg.String()) == 1 {
		m.searchIn, _ = m.searchIn.Update(msg)
		m.searchQuery = m.searchIn.Value()
		m.applyFilters()
		return m, nil
	}

	return m, nil
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
	m.selectedVersion = row[1]
	m.state = stateConfirm
	return m, nil
}

func (m *model) applyFilters() {
	var filtered []table.Row

	for _, row := range m.rows {
		version := row[1]
		vType := row[2]
		status := row[4]

		if !m.showMobile && vType == "Mobile" {
			continue
		}
		if !m.showEarlyAccess {
			var isEarly bool
			for _, v := range m.result.RSS {
				if v.Version == version {
					isEarly = v.IsEarly
					break
				}
			}
			if isEarly {
				continue
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

func (m *model) cssDirFor(version string) string {
	return filepath.Join(".obsidian_cache", "css", version, "app.css")
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

		cssStatus := ""
		if _, err := os.Stat(m.cssDirFor(v)); err == nil {
			cssStatus = "\u2713"
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
			cssStatus,
		})
	}

	m.rows = rows
	m.filtered = rows
}

func (m *model) updateTableDimensions() {
	availW := m.width - 6
	if availW < 60 {
		availW = 60
	}

	m.tbl.SetColumns([]table.Column{
		{Title: "ID", Width: 5},
		{Title: "Version", Width: 14},
		{Title: "Type", Width: 8},
		{Title: "Date", Width: 12},
		{Title: "Docker", Width: 8},
		{Title: "Electron", Width: 10},
		{Title: "Chromium", Width: 10},
		{Title: "CSS", Width: 6},
	})
}

func (m *model) Run() (Selection, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return Selection{}, err
	}
	m2, ok := finalModel.(*model)
	if !ok {
		return Selection{}, nil
	}
	if m2.state == stateConfirm {
		return Selection{Version: m2.selectedVersion}, nil
	}
	return Selection{}, nil
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
	if len(row) > 7 && row[7] == "\u2713" {
		return 0
	}
	switch row[4] {
	case statusFoundStr:
		return 1
	case statusNAStr:
		return 2
	case statusMissingStr:
		return 3
	}
	return 4
}

func fmtElectron(v string) string {
	if v == "" {
		return "---"
	}
	return "v" + v
}

const (
	statusFoundStr   = "Found"
	statusMissingStr = "Missing"
	statusNAStr      = "N/A"
)
