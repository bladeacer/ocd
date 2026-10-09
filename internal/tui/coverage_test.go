package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/models"
	"github.com/bladeacer/ocd/internal/sources"
)

// fakeProgram replaces the Bubble Tea program so a model can be driven
// without a terminal.
type fakeProgram struct {
	model tea.Model
	send  []tea.Msg
	err   error
	ran   bool
}

func (f *fakeProgram) Run() (tea.Model, error) {
	f.ran = true
	if f.err != nil {
		return nil, f.err
	}
	for _, msg := range f.send {
		next, cmd := f.model.Update(msg)
		if next != nil {
			f.model = next
		}
		if cmd != nil {
			// Run any command the update produced, ignoring its result.
			_ = cmd()
		}
	}
	return f.model, nil
}

// withFakeProgram installs a fake program for the duration of a test.
func withFakeProgram(t *testing.T, send []tea.Msg, err error) *fakeProgram {
	t.Helper()
	fp := &fakeProgram{send: send, err: err}
	orig := newProgram
	newProgram = func(m tea.Model) program {
		fp.model = m
		return fp
	}
	t.Cleanup(func() { newProgram = orig })
	return fp
}

// seededFetcher returns a fetcher backed by a cache that already holds every
// source, so no network call is made.
func seededFetcher(t *testing.T, rss []models.RSSVersion) *sources.Fetcher {
	t.Helper()
	dir := t.TempDir()
	orig := cache.CacheDir
	cache.CacheDir = dir
	t.Cleanup(func() { cache.CacheDir = orig })

	c, err := cache.New(0)
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	if err := c.Set("rss_versions", rss); err != nil {
		t.Fatalf("set rss: %v", err)
	}
	if err := c.Set("docker_versions", []models.DockerTag{{Version: "1.0.0", Tag: "latest"}}); err != nil {
		t.Fatalf("set docker: %v", err)
	}
	if err := c.Set("electron_versions", models.ElectronMap{"1.0.0": "120"}); err != nil {
		t.Fatalf("set electron: %v", err)
	}
	return sources.NewFetcher(c)
}

func sampleRSS() []models.RSSVersion {
	return []models.RSSVersion{
		{Version: "1.0.0", Type: models.Desktop, Date: "2023-01-01", Electron: "28"},
		{Version: "1.1.0", Type: models.Desktop, Date: "2023-02-01", Electron: "28"},
		{Version: "1.0.0", Type: models.Mobile, Date: "2023-01-02", Electron: "28"},
		{Version: "0.9.0", Type: models.Desktop, Date: "2022-12-01", IsEarly: true, Electron: "27"},
	}
}

func TestModelNewAndInit(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)

	if m.state != stateLoading {
		t.Errorf("state = %v, want stateLoading", m.state)
	}
	if len(m.loadMessages) == 0 {
		t.Error("expected loading messages")
	}
	if cmd := m.Init(); cmd == nil {
		t.Error("Init must return a command")
	}
}

func TestModelFetchDataUsesCache(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)

	msg := m.fetchData()
	loaded, ok := msg.(dataLoadedMsg)
	if !ok {
		t.Fatalf("fetchData returned %T, want dataLoadedMsg", msg)
	}
	if loaded.result == nil || len(loaded.result.RSS) != len(sampleRSS()) {
		t.Fatalf("unexpected result: %+v", loaded.result)
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if next != m {
		t.Fatal("Update should return the same model")
	}
	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
}

func TestModelUpdateDataLoaded(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)

	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(dataLoadedMsg{result: &models.FetchResult{RSS: sampleRSS()}})

	if len(m.rows) == 0 {
		t.Fatal("expected rows to be built")
	}
	view := m.View()
	if !strings.Contains(view, "Version") {
		t.Errorf("expected a table header in the view, got:\n%s", view)
	}
}

func TestModelUpdateDataError(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)

	m.Update(dataLoadedMsg{result: &models.FetchResult{Error: errors.New("boom")}})
	if m.err == nil {
		t.Error("expected the error to be stored")
	}
}

func TestModelUpdateKeys(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	m := New(f, false)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(dataLoadedMsg{result: &models.FetchResult{RSS: sampleRSS()}})

	keys := []tea.KeyMsg{
		{Type: tea.KeyDown},
		{Type: tea.KeyUp},
		{Type: tea.KeyRunes, Runes: []rune("x")},
		{Type: tea.KeyBackspace},
		{Type: tea.KeyTab},
		{Type: tea.KeyCtrlR},
	}
	for _, k := range keys {
		m.Update(k)
	}
	if m.searchIn.Value() != "" {
		t.Logf("search value after typing and deleting: %q", m.searchIn.Value())
	}
}

func TestModelRunConfirmed(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	withFakeProgram(t, []tea.Msg{
		dataLoadedMsg{result: &models.FetchResult{RSS: sampleRSS()}},
		tea.WindowSizeMsg{Width: 100, Height: 30},
	}, nil)

	sel, err := New(f, false).Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sel.Version != "" {
		t.Logf("selection without a confirm: %q", sel.Version)
	}
}

func TestModelRunProgramError(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	withFakeProgram(t, nil, errors.New("no tty"))

	if _, err := New(f, false).Run(); err == nil {
		t.Error("expected the program error to be returned")
	}
}

func TestModelRunWrongModelType(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	orig := newProgram
	newProgram = func(tea.Model) program { return &wrongModelProgram{} }
	t.Cleanup(func() { newProgram = orig })

	sel, err := New(f, false).Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if sel.Version != "" {
		t.Errorf("expected an empty selection, got %q", sel.Version)
	}
}

type wrongModelProgram struct{}

func (w *wrongModelProgram) Run() (tea.Model, error) { return wrongModel{}, nil }

type wrongModel struct{}

func (wrongModel) Init() tea.Cmd                         { return nil }
func (w wrongModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return w, nil }
func (wrongModel) View() string                          { return "" }

func TestPickerInitAndLoadData(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	p := NewPicker(f, false)

	if cmd := p.Init(); cmd == nil {
		t.Error("Init must return a command")
	}
	msg := p.loadData()
	loaded, ok := msg.(dataLoadedMsg)
	if !ok {
		t.Fatalf("loadData returned %T, want dataLoadedMsg", msg)
	}
	if loaded.result == nil {
		t.Fatal("loadData returned a nil result")
	}
}

func TestPickerUpdateDataLoaded(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	p := NewPicker(f, false)

	p.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	p.Update(dataLoadedMsg{result: &models.FetchResult{RSS: sampleRSS()}})

	if len(p.rows) == 0 {
		t.Fatal("expected rows to be built")
	}
	if view := p.View(); !strings.Contains(view, "Version") {
		t.Errorf("expected a header in the view, got:\n%s", view)
	}
}

func TestPickerUpdateDataError(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	p := NewPicker(f, false)

	p.Update(dataLoadedMsg{result: &models.FetchResult{Error: errors.New("boom")}})
	if p.err == nil {
		t.Error("expected the error to be stored")
	}
	// While loading or failed, only quit keys are handled.
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("q should quit while data is unavailable")
	}
}

func TestPickerUpdateTick(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	p := NewPicker(f, false)

	if _, cmd := p.Update(tickMsg{}); cmd == nil {
		t.Error("a tick should produce a command")
	}
}

func TestPickerViewBeforeData(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	p := NewPicker(f, false)

	view := p.View()
	if !strings.Contains(view, "Loading") {
		t.Errorf("expected a loading message, got:\n%s", view)
	}
}

func TestPickVersionsCancelled(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	withFakeProgram(t, nil, nil)

	a, b, err := PickVersions(f, false)
	if err != nil {
		t.Fatalf("PickVersions: %v", err)
	}
	if a != "" || b != "" {
		t.Errorf("expected no versions, got %q %q", a, b)
	}
}

func TestPickVersionsProgramError(t *testing.T) {
	f := seededFetcher(t, sampleRSS())
	withFakeProgram(t, nil, errors.New("no tty"))

	if _, _, err := PickVersions(f, false); err == nil {
		t.Error("expected the program error to be returned")
	}
}

func TestRunDiffViewerError(t *testing.T) {
	withFakeProgram(t, nil, errors.New("no tty"))

	if err := RunDiffViewer(&models.DiffResult{VersionA: "1.0.0", VersionB: "1.1.0"}, config.DiffKeys{}); err == nil {
		t.Error("expected the program error to be returned")
	}
}

func TestRunDiffViewerQuits(t *testing.T) {
	withFakeProgram(t, nil, nil)

	if err := RunDiffViewer(&models.DiffResult{VersionA: "1.0.0", VersionB: "1.1.0"}, config.DiffKeys{}); err != nil {
		t.Fatalf("RunDiffViewer: %v", err)
	}
}

func TestBuildKeybindsDefaults(t *testing.T) {
	kb := buildKeybinds(config.DiffKeys{})
	for _, k := range []string{"prev_hunk", "next_hunk", "scroll_down", "scroll_up",
		"quit", "help", "export", "next_search", "prev_search", "scroll_to_hunk",
		"scroll_to_top", "scroll_to_bottom", "toggle_side", "yank", "yank_all"} {
		if len(kb[k]) == 0 {
			t.Errorf("keybind %q has no default", k)
		}
	}
}

func TestBuildKeybindsOverrides(t *testing.T) {
	kb := buildKeybinds(config.DiffKeys{
		PrevHunk: []string{"p"},
		Quit:     []string{"x", "Q"},
	})
	if len(kb["prev_hunk"]) != 1 || kb["prev_hunk"][0] != "p" {
		t.Errorf("prev_hunk = %v, want [p]", kb["prev_hunk"])
	}
	if len(kb["quit"]) != 2 {
		t.Errorf("quit = %v, want two keys", kb["quit"])
	}
}

func TestDiffModelExportWritesFile(t *testing.T) {
	// The export writes to the working directory, which TestMain has pointed
	// at a scratch directory. This is the path that once left an
	// ocd-tldr-a-b.toml behind in the package source tree.
	withFakeProgram(t, []tea.Msg{
		tea.WindowSizeMsg{Width: 120, Height: 40},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")},
		tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")},
	}, nil)

	result := &models.DiffResult{
		VersionA: "1.0.0",
		VersionB: "1.1.0",
		Diff:     "@@ -1,1 +1,2 @@\n-a\n+b\n+c",
		HasDiff:  true,
	}
	if err := RunDiffViewer(result, config.DiffKeys{}); err != nil {
		t.Fatalf("RunDiffViewer: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	matches, _ := filepath.Glob(filepath.Join(wd, "ocd-tldr-1.0.0-1.1.0.*"))
	if len(matches) == 0 {
		t.Errorf("expected an export in %s, found none", wd)
	}
}
