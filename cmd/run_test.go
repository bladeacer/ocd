package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/models"
	"github.com/bladeacer/ocd/internal/sources"
	"github.com/bladeacer/ocd/internal/tui"
)

// stubViewer stops the diff command from opening an interactive viewer.
func stubViewer(t *testing.T) {
	t.Helper()
	orig := runViewer
	runViewer = func(*models.DiffResult, config.DiffKeys) error { return nil }
	t.Cleanup(func() { runViewer = orig })
}

func TestDiffCmdBetweenCachedVersions(t *testing.T) {
	isolate(t)
	stubViewer(t)
	useTempCSSDir(t)
	seedCSS(t, "1.0.0", ":root{--a:1;--gone:1}\n.old{color:#fff}")
	seedCSS(t, "1.1.0", ":root{--a:1;--fresh:2}\n.new{color:#000}")

	out, err := execute(t, NewDiffCmd, "1.0.0", "1.1.0")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "1.0.0") || !strings.Contains(out, "1.1.0") {
		t.Errorf("expected both versions in the output, got:\n%s", out)
	}
}

func TestDiffCmdWithTLDR(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	seedCSS(t, "1.0.0", ".old{color:#fff}")
	seedCSS(t, "1.1.0", ".new{color:#000}")

	if _, err := execute(t, NewDiffCmd, "1.0.0", "1.1.0", "--tldr"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ocd-tldr-1.0.0-1.1.0.toml")); err != nil {
		t.Errorf("expected a TLDR export: %v", err)
	}
}

func TestDiffCmdWithTLDROutputDirectory(t *testing.T) {
	dir := isolate(t)
	useTempCSSDir(t)
	seedCSS(t, "1.0.0", ".old{color:#fff}")
	seedCSS(t, "1.1.0", ".new{color:#000}")
	target := filepath.Join(dir, "reports")

	if _, err := execute(t, NewDiffCmd, "1.0.0", "1.1.0", "--tldr", "--tldr-output", target); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "ocd-tldr-1.0.0-1.1.0.toml")); err != nil {
		t.Errorf("expected the export in the output directory: %v", err)
	}
}

func TestDiffCmdWithOneArgumentIsAUsageError(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)

	if _, err := execute(t, NewDiffCmd, "1.0.0"); err == nil {
		t.Error("expected a usage error for a single version")
	}
}

func TestDiffCmdPickerCancelled(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	orig := pickVersions
	pickVersions = func(*sources.Fetcher, bool) (string, string, error) { return "", "", nil }
	t.Cleanup(func() { pickVersions = orig })

	out, err := execute(t, NewDiffCmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "cancelled") {
		t.Errorf("expected a cancellation message, got:\n%s", out)
	}
}

func TestDiffCmdPickerError(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	orig := pickVersions
	pickVersions = func(*sources.Fetcher, bool) (string, string, error) { return "", "", errors.New("boom") }
	t.Cleanup(func() { pickVersions = orig })

	if _, err := execute(t, NewDiffCmd, "--pick"); err == nil {
		t.Error("expected the picker error to be returned")
	}
}

func TestDiffCmdPickerSelectsVersions(t *testing.T) {
	isolate(t)
	stubViewer(t)
	useTempCSSDir(t)
	seedCSS(t, "1.0.0", ".a{color:red}")
	seedCSS(t, "1.1.0", ".a{color:blue}")

	orig := pickVersions
	pickVersions = func(*sources.Fetcher, bool) (string, string, error) { return "1.0.0", "1.1.0", nil }
	t.Cleanup(func() { pickVersions = orig })

	if _, err := execute(t, NewDiffCmd, "--pick"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestDiffCmdPickFlagCancelled(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	orig := pickVersions
	pickVersions = func(*sources.Fetcher, bool) (string, string, error) { return "", "", nil }
	t.Cleanup(func() { pickVersions = orig })

	out, err := execute(t, NewDiffCmd, "--pick")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "cancelled") {
		t.Errorf("expected a cancellation message, got:\n%s", out)
	}
}

func TestInteractCmdCancelled(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	seedMetadata(t, nil)

	orig := runInteract
	runInteract = func(*sources.Fetcher, bool) (tui.Selection, error) { return tui.Selection{}, nil }
	t.Cleanup(func() { runInteract = orig })

	out, err := execute(t, NewInteractCmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "cancelled") {
		t.Errorf("expected a cancellation message, got:\n%s", out)
	}
}

func TestInteractCmdSelectsAVersion(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	seedMetadata(t, nil)
	seedCSS(t, "1.0.0", ":root{--a:1}")

	orig := runInteract
	runInteract = func(*sources.Fetcher, bool) (tui.Selection, error) {
		return tui.Selection{Version: "1.0.0"}, nil
	}
	t.Cleanup(func() { runInteract = orig })

	out, err := execute(t, NewInteractCmd)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Saved to") {
		t.Errorf("expected a saved message, got:\n%s", out)
	}
}

func TestInteractCmdError(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	seedMetadata(t, nil)

	orig := runInteract
	runInteract = func(*sources.Fetcher, bool) (tui.Selection, error) {
		return tui.Selection{}, errors.New("boom")
	}
	t.Cleanup(func() { runInteract = orig })

	if _, err := execute(t, NewInteractCmd); err == nil {
		t.Error("expected the TUI error to be returned")
	}
}

func TestInteractCmdUnknownSelectedVersion(t *testing.T) {
	isolate(t)
	useTempCSSDir(t)
	seedMetadata(t, nil)

	orig := runInteract
	runInteract = func(*sources.Fetcher, bool) (tui.Selection, error) {
		// Not cached, so the extract fails without a valid release URL.
		return tui.Selection{Version: "0.0.0"}, nil
	}
	t.Cleanup(func() { runInteract = orig })

	if _, err := execute(t, NewInteractCmd); err == nil {
		t.Error("expected an extract error")
	}
}
