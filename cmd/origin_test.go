package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/models"
)

func TestNewOriginCmd(t *testing.T) {
	c := NewOriginCmd()
	if c.Use != "origin <selector|variable>" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Find the earliest public desktop Obsidian version where a selector or variable was introduced" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
	if c.Flag("format") == nil {
		t.Error("expected --format flag")
	}
	if c.Flag("output") == nil {
		t.Error("expected --output flag")
	}
	if c.Flag("refresh") == nil {
		t.Error("expected --refresh flag")
	}
}

func TestMarshalOriginTOML(t *testing.T) {
	r := &core.OriginResult{
		Target:     ".foo",
		Kind:       "selector",
		Introduced: "1.1.0",
		Found:      true,
		Versions:   []string{"1.1.0"},
	}
	data, err := marshalOrigin(r, "toml")
	if err != nil {
		t.Fatalf("marshalOrigin TOML: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty TOML output")
	}
}

func TestMarshalOriginJSON(t *testing.T) {
	r := &core.OriginResult{
		Target:     "--my-var",
		Kind:       "variable",
		Introduced: "1.2.0",
		Found:      true,
	}
	data, err := marshalOrigin(r, "json")
	if err != nil {
		t.Fatalf("marshalOrigin JSON: %v", err)
	}
	if !strings.Contains(string(data), `"target"`) {
		t.Errorf("expected target in JSON, got %s", string(data))
	}
}

func TestMarshalOriginYAML(t *testing.T) {
	r := &core.OriginResult{
		Target: ".foo",
		Kind:   "selector",
		Found:  false,
	}
	data, err := marshalOrigin(r, "yaml")
	if err != nil {
		t.Fatalf("marshalOrigin YAML: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty YAML output")
	}
}

func TestMarshalOriginYML(t *testing.T) {
	r := &core.OriginResult{
		Target: ".foo",
		Kind:   "selector",
		Found:  true,
	}
	data, err := marshalOrigin(r, "yml")
	if err != nil {
		t.Fatalf("marshalOrigin YML: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty YML output")
	}
}

func TestMarshalOriginDefaultFormat(t *testing.T) {
	r := &core.OriginResult{
		Target: ".foo",
		Kind:   "selector",
		Found:  true,
	}
	data, err := marshalOrigin(r, "unknown")
	if err != nil {
		t.Fatalf("marshalOrigin default: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output for default format")
	}
}

func TestSanitizeFilenameSelector(t *testing.T) {
	got := sanitizeFilename(".messageBar")
	if got != "messageBar" {
		t.Errorf("expected messageBar, got %s", got)
	}
}

func TestSanitizeFilenameVariable(t *testing.T) {
	got := sanitizeFilename("--my-var")
	if got != "my-var" {
		t.Errorf("expected my-var, got %s", got)
	}
}

func TestSanitizeFilenameWithSpaces(t *testing.T) {
	got := sanitizeFilename(".foo .bar")
	if !strings.Contains(got, "-") {
		t.Errorf("expected dashes for spaces, got %s", got)
	}
}

func TestSanitizeFilenameAllInvalid(t *testing.T) {
	got := sanitizeFilename("...")
	if got != "untitled" {
		t.Errorf("expected untitled, got %s", got)
	}
}

func TestSanitizeFilenameID(t *testing.T) {
	got := sanitizeFilename("#editor")
	if got != "editor" {
		t.Errorf("expected editor, got %s", got)
	}
}

func TestOriginCmdExecution(t *testing.T) {
	dir := t.TempDir()

	origWd, _ := os.Getwd()
	wd, err := os.MkdirTemp("", "ocd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		_ = os.RemoveAll(wd)
	}()
	_ = os.Chdir(wd)

	origCacheDir := cache.CacheDir
	cache.CacheDir = dir
	defer func() { cache.CacheDir = origCacheDir }()

	cssDir := filepath.Join(dir, "css")
	origCSSDir := core.CSSDir
	core.CSSDir = cssDir
	defer func() { core.CSSDir = origCSSDir }()

	c, err := cache.New(0)
	if err != nil {
		t.Fatal(err)
	}
	rssData := []models.RSSVersion{
		{Version: "1.0.0", Type: models.Desktop, IsEarly: false},
		{Version: "1.1.0", Type: models.Desktop, IsEarly: false},
		{Version: "1.0.1", Type: models.Desktop, IsEarly: true},
	}
	if err := c.Set("rss_versions", rssData); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("docker_versions", []models.DockerTag{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("electron_versions", models.ElectronMap{}); err != nil {
		t.Fatal(err)
	}

	for _, v := range []string{"1.0.0", "1.1.0"} {
		verDir := filepath.Join(cssDir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		css := ".old { color: red; }"
		if v == "1.1.0" {
			css = ".old { color: red; }\n.messageBar { color: blue; }"
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := NewOriginCmd()
	cmd.SetArgs([]string{".messageBar"})
	output := captureStdout(t, func() {
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	if !strings.Contains(output, "1.1.0") {
		t.Errorf("expected 1.1.0 in output, got %s", output)
	}
	if !strings.Contains(output, "Exported:") {
		t.Errorf("expected Exported in output, got %s", output)
	}
}

func TestOriginCmdExecutionFallback(t *testing.T) {
	dir := t.TempDir()

	origWd, _ := os.Getwd()
	wd, err := os.MkdirTemp("", "ocd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		_ = os.RemoveAll(wd)
	}()
	_ = os.Chdir(wd)

	origCacheDir := cache.CacheDir
	cache.CacheDir = dir
	defer func() { cache.CacheDir = origCacheDir }()

	cssDir := filepath.Join(dir, "css")
	origCSSDir := core.CSSDir
	core.CSSDir = cssDir
	defer func() { core.CSSDir = origCSSDir }()

	c, err := cache.New(0)
	if err != nil {
		t.Fatal(err)
	}
	// Empty RSS cache forces the fallback to ListCachedVersions.
	if err := c.Set("rss_versions", []models.RSSVersion{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("docker_versions", []models.DockerTag{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("electron_versions", models.ElectronMap{}); err != nil {
		t.Fatal(err)
	}

	for _, v := range []string{"1.0.0", "1.1.0"} {
		verDir := filepath.Join(cssDir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		css := ".old { color: red; }"
		if v == "1.1.0" {
			css = ".old { color: red; }\n.messageBar { color: blue; }"
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := NewOriginCmd()
	cmd.SetArgs([]string{".messageBar"})
	output := captureStdout(t, func() {
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	if !strings.Contains(output, "1.1.0") {
		t.Errorf("expected 1.1.0 in output, got %s", output)
	}
}

func TestOriginCmdWithConfig(t *testing.T) {
	dir := t.TempDir()

	origWd, _ := os.Getwd()
	wd, err := os.MkdirTemp("", "ocd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		_ = os.Chdir(origWd)
		_ = os.RemoveAll(wd)
		_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
	}()
	_ = os.Chdir(wd)
	_ = os.Setenv("XDG_CONFIG_HOME", dir)

	if err := os.WriteFile(filepath.Join(wd, ".ocd.toml"), []byte("origin_format = \"json\"\norigin_dir = \".\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	origCacheDir := cache.CacheDir
	cache.CacheDir = dir
	defer func() { cache.CacheDir = origCacheDir }()

	cssDir := filepath.Join(dir, "css")
	origCSSDir := core.CSSDir
	core.CSSDir = cssDir
	defer func() { core.CSSDir = origCSSDir }()

	c, err := cache.New(0)
	if err != nil {
		t.Fatal(err)
	}
	rssData := []models.RSSVersion{
		{Version: "1.0.0", Type: models.Desktop, IsEarly: false},
		{Version: "1.1.0", Type: models.Desktop, IsEarly: false},
	}
	if err := c.Set("rss_versions", rssData); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("docker_versions", []models.DockerTag{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("electron_versions", models.ElectronMap{}); err != nil {
		t.Fatal(err)
	}

	for _, v := range []string{"1.0.0", "1.1.0"} {
		verDir := filepath.Join(cssDir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		css := ".messageBar { color: blue; }"
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := NewOriginCmd()
	cmd.SetArgs([]string{".messageBar"})
	output := captureStdout(t, func() {
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	if !strings.Contains(output, "1.0.0") {
		t.Errorf("expected 1.0.0 in output, got %s", output)
	}
	// Config sets format to json, so the export file should be .json
	expectedFile := filepath.Join(wd, "ocd-origin-messageBar.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected exported JSON file at %s", expectedFile)
	}
}

func TestOriginCmdVariable(t *testing.T) {
	dir := t.TempDir()

	origWd, _ := os.Getwd()
	wd, err := os.MkdirTemp("", "ocd-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(origWd)
		_ = os.RemoveAll(wd)
	}()
	_ = os.Chdir(wd)

	origCacheDir := cache.CacheDir
	cache.CacheDir = dir
	defer func() { cache.CacheDir = origCacheDir }()

	cssDir := filepath.Join(dir, "css")
	origCSSDir := core.CSSDir
	core.CSSDir = cssDir
	defer func() { core.CSSDir = origCSSDir }()

	c, err := cache.New(0)
	if err != nil {
		t.Fatal(err)
	}
	rssData := []models.RSSVersion{
		{Version: "1.0.0", Type: models.Desktop, IsEarly: false},
		{Version: "1.1.0", Type: models.Desktop, IsEarly: false},
	}
	if err := c.Set("rss_versions", rssData); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("docker_versions", []models.DockerTag{}); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("electron_versions", models.ElectronMap{}); err != nil {
		t.Fatal(err)
	}

	for _, v := range []string{"1.0.0", "1.1.0"} {
		verDir := filepath.Join(cssDir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		css := ".old { color: red; }"
		if v == "1.1.0" {
			css = ":root {\n  --my-var: blue;\n}"
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := NewOriginCmd()
	cmd.SetArgs([]string{"--", "--my-var"})
	output := captureStdout(t, func() {
		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	if !strings.Contains(output, "1.1.0") {
		t.Errorf("expected 1.1.0 in output, got %s", output)
	}
	if !strings.Contains(output, "variable") {
		t.Errorf("expected variable kind in output, got %s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}
