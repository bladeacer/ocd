package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bladeacer/ocd/internal/models"
)

// writeConfig puts a project config file in the working directory, which
// isolate() has already pointed at a temp dir.
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(wd, ".ocd.toml")
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckCmdUsesConfigFile(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	writeConfig(t, "check_format = \"json\"\ncheck_compat_mode = \"relaxed\"\ncache_days = 30\n")
	seedCSS(t, "1.6.3", ":root{--a:1}")
	theme := filepath.Join(dir, "theme.css")
	if err := os.WriteFile(theme, []byte(":root{--a:1}"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := execute(t, NewCheckCmd, "1.6.3", theme); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// check_format = json drives the extension.
	if _, err := os.Stat(filepath.Join(dir, "ocd-check-1.6.3.json")); err != nil {
		t.Errorf("expected a json export from the config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-check-1.6.3-selectors.json")); err != nil {
		t.Errorf("expected a json selector export from the config: %v", err)
	}
}

func TestCheckCmdOutputFromConfig(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	reports := filepath.Join(dir, "from-config")
	writeConfig(t, "check_dir = \""+reports+"\"\n")
	seedCSS(t, "1.6.3", ":root{--a:1}")
	theme := filepath.Join(dir, "theme.css")
	if err := os.WriteFile(theme, []byte(":root{--a:1}"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := execute(t, NewCheckCmd, "1.6.3", theme); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(reports, "ocd-check-1.6.3.toml")); err != nil {
		t.Errorf("expected the export in the configured directory: %v", err)
	}
}

func TestCheckCmdFlagBeatsConfig(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	writeConfig(t, "check_format = \"json\"\n")
	seedCSS(t, "1.6.3", ":root{--a:1}")
	theme := filepath.Join(dir, "theme.css")
	if err := os.WriteFile(theme, []byte(":root{--a:1}"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := execute(t, NewCheckCmd, "1.6.3", theme, "--format", "yaml"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-check-1.6.3.yaml")); err != nil {
		t.Errorf("the flag should win over the config: %v", err)
	}
}

// With a seeded feed and both versions already cached, origin runs without
// touching the network and honours origin_format from the config.
func TestOriginCmdUsesConfigFile(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	seedMetadata(t, []models.RSSVersion{
		{Version: "1.0.0", Type: models.Desktop, Date: "2023-01-01"},
	})
	seedCSS(t, "1.0.0", ":root{--a:1}")
	writeConfig(t, "origin_format = \"json\"\norigin_dir = \"\"\n")

	if _, err := execute(t, NewOriginCmd, "--", "--a"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-origin-a.json")); err != nil {
		t.Errorf("expected a json export from the config: %v", err)
	}
}

func TestDiffCmdUsesConfigFile(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	stubViewer(t)
	seedCSS(t, "1.0.0", ".a{color:#fff}")
	seedCSS(t, "1.1.0", ".a{color:#000}")
	writeConfig(t, "tldr = true\ntldr_format = \"json\"\ntldr_dir = \"\"\n")

	out, err := execute(t, NewDiffCmd, "1.0.0", "1.1.0")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Exported") {
		t.Errorf("expected a TLDR export driven by the config, got:\n%s", out)
	}
}

func TestDiffCmdRefreshFromConfig(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	stubViewer(t)
	seedCSS(t, "1.0.0", ".a{color:#fff}")
	seedCSS(t, "1.1.0", ".a{color:#000}")
	writeConfig(t, "refresh = true\n")

	// refresh forces a metadata re-fetch, which is fine because both CSS
	// files are already cached.
	if _, err := execute(t, NewDiffCmd, "1.0.0", "1.1.0", "--tldr"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestStatCmdUsesConfigFile(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	writeConfig(t, "stat_format = \"yaml\"\nstat_dir = \"\"\n")
	seedCSS(t, "1.6.3", ":root{--a:1}")

	if _, err := execute(t, NewStatCmd, "1.6.3"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-stat-1.6.3.yaml")); err != nil {
		t.Errorf("expected a yaml export from the config: %v", err)
	}
}

func TestConfigWithDiffKeys(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	stubViewer(t)
	seedCSS(t, "1.0.0", ".a{color:#fff}")
	seedCSS(t, "1.1.0", ".a{color:#000}")
	writeConfig(t, "pick = false\n[diff_keys]\nprev_hunk = [\"1\"]\nquit = [\"q\"]\n")

	if _, err := execute(t, NewDiffCmd, "1.0.0", "1.1.0"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}
