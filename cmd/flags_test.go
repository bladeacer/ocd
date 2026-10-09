package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/models"
)

// seedMetadata fills the metadata cache so a fetcher reads from disk instead
// of the network.
func seedMetadata(t *testing.T, rss []models.RSSVersion) {
	t.Helper()
	orig := cache.CacheDir
	cache.CacheDir = t.TempDir()
	t.Cleanup(func() { cache.CacheDir = orig })

	c, err := cache.New(0)
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	if rss == nil {
		rss = []models.RSSVersion{}
	}
	if err := c.Set("rss_versions", rss); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("docker_versions", []models.DockerTag{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("electron_versions", models.ElectronMap{}); err != nil {
		t.Fatal(err)
	}
}

// With no versions in the feed there is nothing to sweep, so the sweep
// returns early and never reaches the network.
func TestSweepAllVersionsWithNoVersions(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	seedMetadata(t, nil)

	if versions := sweepAllVersions(mustResolveConfig(t), 14, false); len(versions) != 0 {
		t.Errorf("expected no versions, got %v", versions)
	}
}

// Every version is already cached, so the sweep does no download.
func TestSweepAllVersionsWithFreshCache(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	seedMetadata(t, []models.RSSVersion{
		{Version: "1.0.0", Type: models.Desktop, Date: "2023-01-01"},
		{Version: "1.1.0", Type: models.Desktop, Date: "2023-02-01"},
	})
	seedCSS(t, "1.0.0", ":root{--a:1}")
	seedCSS(t, "1.1.0", ":root{--a:1}")

	versions := sweepAllVersions(mustResolveConfig(t), 14, false)
	if len(versions) != 2 {
		t.Errorf("expected both cached versions, got %v", versions)
	}
}

func TestMarshalSelectors(t *testing.T) {
	rep := &core.SelectorReport{TargetVersion: "1.0.0", Styled: 3, Covered: 5, Uncovered: 1}

	for _, format := range []string{"toml", "", "TOML"} {
		out, err := marshalSelectors(rep, format)
		if err != nil {
			t.Fatalf("marshalSelectors(%q): %v", format, err)
		}
		if !strings.Contains(string(out), "styled") {
			t.Errorf("toml output missing styled: %s", out)
		}
	}
	out, err := marshalSelectors(rep, "json")
	if err != nil {
		t.Fatalf("marshalSelectors(json): %v", err)
	}
	if !strings.Contains(string(out), `"covered": 5`) {
		t.Errorf("json missing covered: %s", out)
	}
	out, err = marshalSelectors(rep, "yml")
	if err != nil {
		t.Fatalf("marshalSelectors(yml): %v", err)
	}
	if !strings.Contains(string(out), "covered: 5") {
		t.Errorf("yaml missing covered: %s", out)
	}
}

func TestMarshalStatFormats(t *testing.T) {
	r := &core.TLDRResult{VersionA: "1.0.0", VersionB: "1.1.0"}

	out, err := marshalStat(r, "json")
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	if !strings.Contains(string(out), "version_a") {
		t.Errorf("json missing version_a: %s", out)
	}
	out, err = marshalStat(r, "yaml")
	if err != nil {
		t.Fatalf("yaml: %v", err)
	}
	if !strings.Contains(string(out), "version_a") {
		t.Errorf("yaml missing version_a: %s", out)
	}
	out, err = marshalStat(r, "toml")
	if err != nil {
		t.Fatalf("toml: %v", err)
	}
	if !strings.Contains(string(out), "version_a") {
		t.Errorf("toml missing version_a: %s", out)
	}
}

func TestCommandFlagDefaults(t *testing.T) {
	checks := []struct {
		name string
		flag string
		want string
	}{
		{"check", "format", "toml"},
		{"check", "cache-days", "14"},
		{"origin", "format", "toml"},
		{"origin", "cache-days", "14"},
		{"origin", "refresh", "false"},
		{"stat", "format", "toml"},
		{"extract", "refresh", "false"},
		{"diff", "refresh", "false"},
		{"interact", "refresh", "false"},
	}
	cmds := map[string]flagLookup{
		"check":    NewCheckCmd(),
		"origin":   NewOriginCmd(),
		"stat":     NewStatCmd(),
		"extract":  NewExtractCmd(),
		"diff":     NewDiffCmd(),
		"interact": NewInteractCmd(),
	}
	for _, ch := range checks {
		c := cmds[ch.name]
		f := c.Flag(ch.flag)
		if f == nil {
			t.Errorf("%s: missing --%s flag", ch.name, ch.flag)
			continue
		}
		if f.DefValue != ch.want {
			t.Errorf("%s --%s default = %q, want %q", ch.name, ch.flag, f.DefValue, ch.want)
		}
	}
}

func TestNewCleanCmdMetadata(t *testing.T) {
	c := NewCleanCmd()
	if c.Use != "clean [label]" {
		t.Errorf("Use = %q", c.Use)
	}
	if !strings.Contains(c.Long, "label") {
		t.Errorf("Long should explain the label argument: %s", c.Long)
	}
}

func TestNewInteractCmdMetadata(t *testing.T) {
	c := NewInteractCmd()
	if c.Use != "interact" {
		t.Errorf("Use = %q", c.Use)
	}
	if !strings.Contains(c.Long, "interactive") {
		t.Errorf("Long should describe the TUI: %s", c.Long)
	}
	if c.RunE == nil {
		t.Error("interact must have a RunE")
	}
}

func TestNewDiffCmdMetadata(t *testing.T) {
	c := NewDiffCmd()
	if c.Use != "diff [version-a] [version-b]" {
		t.Errorf("Use = %q", c.Use)
	}
	for _, f := range []string{"tldr", "tldr-format", "tldr-output", "pick", "refresh"} {
		if c.Flag(f) == nil {
			t.Errorf("missing --%s flag", f)
		}
	}
}

func TestNewCheckCmdMetadata(t *testing.T) {
	c := NewCheckCmd()
	if c.Use != "check <version> <theme.css>" {
		t.Errorf("Use = %q", c.Use)
	}
	for _, f := range []string{"format", "output", "silent", "compat-mode", "compat-sweep", "cache-days", "no-selectors"} {
		if c.Flag(f) == nil {
			t.Errorf("missing --%s flag", f)
		}
	}
}

func TestCheckNoSelectorsSkipsSelectorReport(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	seedCSS(t, "1.6.3", ":root{--a:1}.x{}")
	theme := filepath.Join(dir, "theme.css")
	if err := os.WriteFile(theme, []byte(":root{--a:1}.y{}"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := execute(t, NewCheckCmd, "1.6.3", theme, "--no-selectors")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "Selector check") {
		t.Errorf("--no-selectors should skip the selector report, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-check-1.6.3-selectors.toml")); err == nil {
		t.Error("--no-selectors should skip the selector export")
	}
}

func TestCheckSelectorExportFormats(t *testing.T) {
	for _, tc := range []struct{ format, ext string }{
		{"toml", ".toml"}, {"json", ".json"}, {"yaml", ".yaml"},
	} {
		dir := isolate(t)
		useTempCSSDir(t)
		seedCSS(t, "1.6.3", ":root{--a:1}.x{}")
		theme := filepath.Join(dir, "theme.css")
		if err := os.WriteFile(theme, []byte(":root{--a:1}.y{}"), 0644); err != nil {
			t.Fatal(err)
		}

		if _, err := execute(t, NewCheckCmd, "1.6.3", theme, "--format", tc.format); err != nil {
			t.Fatalf("Execute(%s): %v", tc.format, err)
		}
		name := "ocd-check-1.6.3-selectors" + tc.ext
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to be exported: %v", name, err)
		}
	}
}

// flagLookup is the part of a cobra command these tests use.
type flagLookup interface {
	Flag(name string) *pflag.Flag
}

func mustResolveConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Resolve()
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	return cfg
}
