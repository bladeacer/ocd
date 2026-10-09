package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"testing"

	"github.com/bladeacer/ocd/internal/core"
)

// isolate points the CSS cache and the working directory at a temp area so a
// command runs without touching the developer's cache.
func isolate(t *testing.T) string {
	t.Helper()
	useTempCSSDir(t)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	return tmp
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

func execute(t *testing.T, build func() *cobra.Command, args ...string) (string, error) {
	t.Helper()
	cmd := build()
	cmd.SetArgs(args)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stdout)

	origOut, origErr := os.Stdout, os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe: %v", err)
	}
	os.Stdout, os.Stderr = w, w

	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 0, 4096)
		chunk := make([]byte, 4096)
		for {
			n, rerr := r.Read(chunk)
			if n > 0 {
				buf = append(buf, chunk[:n]...)
			}
			if rerr != nil {
				break
			}
		}
		done <- string(buf)
	}()

	execErr := cmd.Execute()

	_ = w.Close()
	os.Stdout, os.Stderr = origOut, origErr
	out := <-done
	_ = r.Close()

	return out, execErr
}

func TestStatCmdWithCachedVersion(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1;--b:2}\n.x{--c:3}")

	out, err := execute(t, NewStatCmd, "1.6.3")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Exported:") {
		t.Errorf("expected an export line, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-stat-1.6.3.toml")); err != nil {
		t.Errorf("expected a toml export: %v", err)
	}
}

func TestStatCmdUnknownVersion(t *testing.T) {
	isolate(t)
	_, err := execute(t, NewStatCmd, "does-not-exist")
	if err == nil {
		t.Error("expected an error for a version with no release")
	}
}

func TestStatCmdWrongArgCount(t *testing.T) {
	isolate(t)
	if _, err := execute(t, NewStatCmd); err == nil {
		t.Error("expected an error with no arguments")
	}
}

func TestCleanCmdWipesEverything(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", "body{}")
	writeFile(t, dir, "marker.json", "{}")

	out, err := execute(t, NewCleanCmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "cleared") {
		t.Errorf("expected a cleared message, got %q", out)
	}
	if core.CSSCached("1.6.3") {
		t.Error("cached CSS should be gone")
	}
}

func TestCleanCmdRemovesOneLabel(t *testing.T) {
	isolate(t)
	seedCSS(t, "1.6.3", "body{}")
	seedCSS(t, "1.6.4", "body{}")

	out, err := execute(t, NewCleanCmd, "1.6.3")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "1.6.3") {
		t.Errorf("expected the label in the message, got %q", out)
	}
	if core.CSSCached("1.6.3") {
		t.Error("the named version should be gone")
	}
	if !core.CSSCached("1.6.4") {
		t.Error("other versions must be kept")
	}
}

func TestExtractCmdImportsCSSFile(t *testing.T) {
	dir := isolate(t)
	src := writeFile(t, dir, "mytheme.css", "body{--a:1}")

	out, err := execute(t, NewExtractCmd, "--from-file", src, "my-label")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Imported app.css") {
		t.Errorf("expected an import message, got %q", out)
	}
	if !core.CSSCached("my-label") {
		t.Error("the label should be cached")
	}
}

func TestExtractCmdDerivesLabelFromFilename(t *testing.T) {
	dir := isolate(t)
	src := writeFile(t, dir, "derived-name.css", "body{--a:1}")

	if _, err := execute(t, NewExtractCmd, "--from-file", src); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !core.CSSCached("derived-name") {
		t.Error("the label should come from the filename")
	}
}

func TestExtractCmdRejectsUnsupportedFile(t *testing.T) {
	dir := isolate(t)
	src := writeFile(t, dir, "thing.txt", "nope")

	if _, err := execute(t, NewExtractCmd, "--from-file", src); err == nil {
		t.Error("expected an error for an unsupported extension")
	}
}

func TestExtractCmdRequiresAnArgument(t *testing.T) {
	isolate(t)
	if _, err := execute(t, NewExtractCmd); err == nil {
		t.Error("expected a usage error with no arguments")
	}
}

func TestCheckCmdWithThemeFile(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1;--b:2}")
	theme := writeFile(t, dir, "theme.css", ":root{--a:9}")

	out, err := execute(t, NewCheckCmd, "1.6.3", theme)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Variable check") {
		t.Errorf("expected a variable report, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-check-1.6.3.toml")); err != nil {
		t.Errorf("expected a toml export: %v", err)
	}
}

func TestCheckCmdAcceptsThemeFolder(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1;--b:2}")
	writeFile(t, dir, "theme/theme.css", ":root{--a:1}")
	writeFile(t, dir, "theme/scss/_p.scss", "$x:1;.y{--b:2}")

	out, err := execute(t, NewCheckCmd, "1.6.3", filepath.Join(dir, "theme"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "1 css file(s)") {
		t.Errorf("expected the folder to be read as css, got %q", out)
	}
}

func TestCheckCmdAcceptsSCSSFolder(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1;--b:2}")
	writeFile(t, dir, "scss/main.scss", "$x:1;.y{--a:1;.z{--b:2}}")

	out, err := execute(t, NewCheckCmd, "1.6.3", filepath.Join(dir, "scss"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "scss file(s)") {
		t.Errorf("expected the folder to be read as scss, got %q", out)
	}
}

func TestCheckCmdFolderWithNoVariables(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1}")
	folder := filepath.Join(dir, "empty")
	writeFile(t, folder, "README.md", "nothing here")

	if _, err := execute(t, NewCheckCmd, "1.6.3", folder); err == nil {
		t.Error("expected an error when no custom properties are found")
	}
}

func TestCheckCmdMissingTheme(t *testing.T) {
	isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1}")

	if _, err := execute(t, NewCheckCmd, "1.6.3", "/nope/missing.css"); err == nil {
		t.Error("expected an error for a missing theme")
	}
}

func TestCheckCmdSilentStillExports(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1}")
	theme := writeFile(t, dir, "theme.css", ":root{--a:1}")

	out, err := execute(t, NewCheckCmd, "1.6.3", theme, "--silent")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "Missing from theme") {
		t.Errorf("--silent should suppress the report, got %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-check-1.6.3.toml")); err != nil {
		t.Errorf("--silent must still export: %v", err)
	}
}

func TestCheckCmdOutputDirectory(t *testing.T) {
	dir := isolate(t)
	seedCSS(t, "1.6.3", ":root{--a:1}")
	theme := writeFile(t, dir, "theme.css", ":root{--a:1}")
	target := filepath.Join(dir, "reports")

	if _, err := execute(t, NewCheckCmd, "1.6.3", theme, "--output", target); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "ocd-check-1.6.3.toml")); err != nil {
		t.Errorf("expected the export in the output directory: %v", err)
	}
}

func TestDiffCmdRejectsOneUnknownVersion(t *testing.T) {
	isolate(t)
	if _, err := execute(t, NewDiffCmd, "1.6.3", "nope-1.0.0"); err == nil {
		t.Error("expected an error for an unknown version")
	}
}

func TestInteractCmdHelpDoesNotRun(t *testing.T) {
	isolate(t)
	c := NewInteractCmd()
	if c.Use != "interact" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Flag("refresh") == nil {
		t.Error("expected --refresh flag")
	}
}
