package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
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
	spinner *spinnerModel

	firstVer        string
	secondVer       string
	err             error
	done            bool
	searchMode      bool
	searchQ         string
	showMobile      bool
	showEarlyAccess bool
	showHelp        bool
	rows            []table.Row
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
		spinner:    newSpinnerModel([]string{"Loading version data..."}),
		showMobile: false,
	}
}

func (m *pickerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), m.loadData)
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

	case tickMsg, spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.err != nil || m.result == nil {
			if msg.String() == "q" || msg.String() == keyCtrlC {
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

	key := msg.String()

	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	switch key {
	case "q", keyCtrlC:
		m.done = true
		return m, tea.Quit
	case keyEscape:
		m.showHelp = false
		return m, nil
	case "/":
		m.searchMode = true
		m.search.Focus()
		m.search.SetValue(m.searchQ)
		return m, nil
	case keyEnter:
		return m.selectCurrent()
	case "m":
		m.showMobile = !m.showMobile
		m.applyFilter()
	case "e":
		m.showEarlyAccess = !m.showEarlyAccess
		m.applyFilter()
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	}

	var cmd tea.Cmd
	m.tbl, cmd = m.tbl.Update(msg)
	return m, cmd
}

func (m *pickerModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEscape, keyCtrlC:
		m.searchMode = false
		m.searchQ = ""
		m.search.Blur()
		m.search.SetValue("")
		m.applyFilter()
		return m, nil
	case keyEnter, keyTab:
		m.searchMode = false
		m.searchQ = m.search.Value()
		m.search.Blur()
		m.applyFilter()
		return m, nil
	case keyBackspace:
		val := m.search.Value()
		if len(val) > 0 {
			m.search.SetValue(val[:len(val)-1])
		}
		m.searchQ = m.search.Value()
		m.applyFilter()
		return m, nil
	case "?":
		m.searchMode = false
		m.searchQ = ""
		m.search.Blur()
		m.search.SetValue("")
		m.applyFilter()
		m.showHelp = !m.showHelp
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
			elV = naStr
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

func (m *pickerModel) cssExtracted(version string) bool {
	_, err := os.Stat(filepath.Join(".obsidian_cache", "css", version, "app.css"))
	return err == nil
}

func (m *pickerModel) applyFilter() {
	var filtered []table.Row

	for _, row := range m.rows {
		if !m.showMobile && row[1] == mobileStr {
			continue
		}
		if !m.showEarlyAccess && m.result != nil {
			var isEarly bool
			for _, v := range m.result.RSS {
				if v.Version == row[0] {
					isEarly = v.IsEarly
					break
				}
			}
			if isEarly {
				continue
			}
		}
		if m.searchQ != "" {
			haystack := strings.ToLower(strings.Join(row, " "))
			if !strings.Contains(haystack, strings.ToLower(m.searchQ)) {
				continue
			}
		}
		filtered = append(filtered, row)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		cssI := m.cssExtracted(filtered[i][0])
		cssJ := m.cssExtracted(filtered[j][0])
		if cssI != cssJ {
			return cssI
		}
		return compareVersions(
			parseVersion(filtered[i][0]),
			parseVersion(filtered[j][0]),
		) > 0
	})

	m.tbl.SetRows(filtered)
}

func (m *pickerModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error loading data:\n\n%v\n\nPress q to quit.", m.err)
	}
	if m.result == nil {
		return m.spinner.View()
	}

	if m.showHelp {
		helpContent := []string{
			"  Version Picker Help",
			"",
			"  ↑ ↓      Navigate rows",
			"  enter    Select version",
			"  /        Search/filter versions",
			"  m        Toggle mobile versions",
			"  e        Toggle early access / insider versions",
			"  q        Quit",
			"  ? / Esc  Close this help",
		}
		helpText := strings.Join(helpContent, "\n")
		box := helpBorderStyle.Render(helpText)
		return "\n\n\n" + box
	}

	prompt := "Select the first version:"
	if m.step == stepPickSecond {
		prompt = fmt.Sprintf("Select the second version (first: %s):", m.firstVer)
	}

	var searchBar string
	if m.searchMode {
		searchBar = "\n" + searchBoxStyle.Render(m.search.View())
	}

	filters := fmt.Sprintf("[%s %s]",
		fmtStatus("M", m.showMobile),
		fmtStatus("E", m.showEarlyAccess),
	)

	keys := helpStyle.Render("/ search  m toggle mobile  e toggle early  enter select  q quit  ? help")

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
