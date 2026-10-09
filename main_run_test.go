package main

import (
	"os"
	"strings"
	"testing"
)

// withArgs runs run() with a fake command line.
func withArgs(t *testing.T, args ...string) error {
	t.Helper()
	orig := os.Args
	os.Args = append([]string{"ocd"}, args...)
	t.Cleanup(func() { os.Args = orig })
	return run()
}

func TestRunHelp(t *testing.T) {
	if err := withArgs(t, "--help"); err != nil {
		t.Fatalf("run(--help): %v", err)
	}
}

func TestRunVersion(t *testing.T) {
	if err := withArgs(t, "--version"); err != nil {
		t.Fatalf("run(--version): %v", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := withArgs(t, "definitely-not-a-command")
	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	if !strings.Contains(err.Error(), "definitely-not-a-command") {
		t.Errorf("error should name the command, got %v", err)
	}
}

func TestRunCleanInTempDir(t *testing.T) {
	// clean touches the cache directory, so run it somewhere harmless.
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := withArgs(t, "clean"); err != nil {
		t.Fatalf("run(clean): %v", err)
	}
}
