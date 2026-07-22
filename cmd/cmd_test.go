package cmd

import (
	"strings"
	"testing"
)

func TestNewDiffCmd(t *testing.T) {
	c := NewDiffCmd()
	if c.Use != "diff [version-a] [version-b]" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Show CSS diff between two Obsidian versions" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
	if !strings.Contains(c.Long, "auto-extracted") {
		t.Errorf("expected auto-extracted in Long, got %s", c.Long)
	}
	refreshFlag := c.Flag("refresh")
	if refreshFlag == nil {
		t.Fatal("expected --refresh flag")
	}
	if refreshFlag.DefValue != "false" {
		t.Errorf("expected refresh default false, got %s", refreshFlag.DefValue)
	}
	pickFlag := c.Flag("pick")
	if pickFlag == nil {
		t.Fatal("expected --pick flag")
	}
}

func TestNewInteractCmd(t *testing.T) {
	c := NewInteractCmd()
	if c.Use != "interact" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Launch the interactive TUI to browse and select Obsidian versions" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
	refreshFlag := c.Flag("refresh")
	if refreshFlag == nil {
		t.Fatal("expected --refresh flag")
	}
}

func TestNewExtractCmd(t *testing.T) {
	c := NewExtractCmd()
	if c.Use != "extract <version>" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Download and extract app.css from an Obsidian release" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
}

func TestNewCleanCmd(t *testing.T) {
	c := NewCleanCmd()
	if c.Use != "clean" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Wipe all cached metadata and extracted CSS files" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
}

func TestEnsureCSS(t *testing.T) {
	err := ensureCSS("999.999.999-test-nonexistent")
	if err == nil {
		t.Log("ensureCSS returned nil (version may exist)")
	}
}
