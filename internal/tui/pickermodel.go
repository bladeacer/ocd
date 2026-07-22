package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bladeacer/ocd/internal/models"
	"github.com/bladeacer/ocd/internal/sources"
)

type pickStep int

const (
	stepPickFirst pickStep = iota
	stepPickSecond
)

type pickerModel struct {
	fetcher *sources.Fetcher
	force   bool
	step    pickStep
	result  *models.FetchResult
	tbl     table.Model
	search  textinput.Model

	firstVer   string
	secondVer  string
	err        error
	done       bool
	searchMode bool
	searchQ    string
	showMobile bool
	rows       []table.Row
}

func NewPicker(f *sources.Fetcher, force bool) *pickerModel {
	columns := []table.Column{
		{Title: "Version", Width: 14},
		{Title: "Type", Width: 10},
		{Title: "Date", Width: 12},
		{Title: "Docker", Width: 8},
		{Title: "Electron", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)
	t.SetStyles(tableStyles())

	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 50
	ti.Width = 30

	return &pickerModel{
		fetcher:    f,
		force:      force,
		step:       stepPickFirst,
		tbl:        t,
		search:     ti,
		showMobile: true,
	}
}

func (m *pickerModel) Init() tea.Cmd {
	return m.loadData
}

func (m *pickerModel) loadData() tea.Msg {
	r := m.fetcher.FetchAll(m.force)
	return dataLoadedMsg{result: r}
}

func (m *pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dataLoadedMsg:
		m.result = msg.result
		if msg.result.Error != nil {
			m.err = msg.result.Error
			return m, nil
		}
		m.buildRows()
		m.applyFilter()
		return m, nil

	case tea.KeyMsg:
		if m.err != nil || m.result == nil {
			if msg.String() == "q" || msg.String() == "ctrl+c" {
				m.done = true
				return m, tea.Quit
			}
			return m, nil
		}

		return m.handleKey(msg)
	}

	return m, nil
}

func (m *pickerModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchMode {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		m.done = true
		return m, tea.Quit
	case "/":
		m.searchMode = true
		m.search.Focus()
		m.search.SetValue(m.searchQ)
		return m, nil
	case "enter":
		return m.selectCurrent()
	case "m":
		m.showMobile = !m.showMobile
		m.applyFilter()
	}

	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	return m, cmd
}

func (m *pickerModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "escape", "ctrl+c":
		m.searchMode = false
		m.searchQ = ""
		m.search.Blur()
		m.search.SetValue("")
		m.applyFilter()
		return m, nil
	case "enter", "tab":
		m.searchMode = false
		m.searchQ = m.search.Value()
		m.search.Blur()
		m.applyFilter()
		return m, nil
	case "backspace":
		val := m.search.Value()
		if len(val) > 0 {
			m.search.SetValue(val[:len(val)-1])
		}
		m.searchQ = m.search.Value()
		m.applyFilter()
		return m, nil
	}

	if len(msg.String()) == 1 {
		m.search, _ = m.search.Update(msg)
		m.searchQ = m.search.Value()
		m.applyFilter()
		return m, nil
	}

	return m, nil
}

func (m *pickerModel) selectCurrent() (tea.Model, tea.Cmd) {
	row := m.tbl.SelectedRow()
	if row == nil {
		return m, nil
	}
	v := row[0]
	if m.step == stepPickFirst {
		m.firstVer = v
		m.step = stepPickSecond
		m.searchQ = ""
		m.buildRows()
		m.applyFilter()
	} else {
		m.secondVer = v
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *pickerModel) buildRows() {
	rss := m.result.RSS
	sort.Slice(rss, func(i, j int) bool {
		return compareVersions(
			parseVersion(rss[i].Version),
			parseVersion(rss[j].Version),
		) > 0
	})

	var rows []table.Row
	dockerMap := make(map[string]bool)
	for _, dt := range m.result.Docker {
		dockerMap[dt.Version] = true
	}

	for _, v := range rss {
		if m.step == stepPickSecond && v.Version == m.firstVer {
			continue
		}
		date := v.Date
		if len(date) > 10 {
			date = date[:10]
		}

		stat := "N/A"
		if v.Type == models.Desktop {
			if dockerMap[v.Version] {
				stat = "Found"
			} else {
				stat = "Missing"
			}
		}

		elV := v.Electron
		if elV == "" {
			elV = "---"
		} else {
			elV = "v" + elV
		}

		rows = append(rows, table.Row{
			v.Version,
			string(v.Type),
			date,
			stat,
			elV,
		})
	}
	m.rows = rows
}

func (m *pickerModel) applyFilter() {
	var filtered []table.Row

	for _, row := range m.rows {
		if !m.showMobile && row[1] == "Mobile" {
			continue
		}
		if m.searchQ != "" {
			haystack := strings.ToLower(strings.Join(row, " "))
			if !strings.Contains(haystack, strings.ToLower(m.searchQ)) {
				continue
			}
		}
		filtered = append(filtered, row)
	}
	m.tbl.SetRows(filtered)
}

func (m *pickerModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading data:\n\n%v\n\nPress q to quit.", m.err)
	}
	if m.result == nil {
		return "\n  Loading version data..."
	}

	prompt := "Select the first version:"
	if m.step == stepPickSecond {
		prompt = fmt.Sprintf("Select the second version (first: %s):", m.firstVer)
	}

	var searchBar string
	if m.searchMode {
		searchBar = "\n" + searchBoxStyle.Render(m.search.View())
	}

	filters := fmt.Sprintf("[%s]",
		fmtStatus("M", m.showMobile),
	)

	keys := helpStyle.Render("/ search  m toggle mobile  enter select  q quit")

	return fmt.Sprintf("%s%s\n\n%s\n\n%s  %s", prompt, searchBar, m.tbl.View(), filters, keys)
}

func PickVersions(f *sources.Fetcher, force bool) (string, string, error) {
	m := NewPicker(f, force)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", "", err
	}
	picked, ok := final.(*pickerModel)
	if !ok || !picked.done {
		return "", "", nil
	}
	return picked.firstVer, picked.secondVer, nil
}
