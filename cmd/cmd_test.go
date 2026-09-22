package cmd

import (
	"strings"
	"testing"

	"github.com/bladeacer/ocd/internal/core"
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
	if c.Use != "extract <version|label>" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Download and extract app.css from an Obsidian release, or import from a local file" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
	if c.Flag("from-file") == nil {
		t.Fatal("expected --from-file flag")
	}
}

func TestNewCleanCmd(t *testing.T) {
	c := NewCleanCmd()
	if c.Use != "clean [label]" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Wipe cached metadata and extracted CSS files" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
}

func TestEnsureCSS(t *testing.T) {
	err := ensureCSS("999.999.999-test-nonexistent")
	if err == nil {
		t.Log("ensureCSS returned nil (version may exist)")
	}
}

func TestMarshalTLDR(t *testing.T) {
	tldr := &core.TLDRResult{
		VersionA:     "1.0.0",
		VersionB:     "1.1.0",
		AdditionsLOC: 10,
		DeletionsLOC: 5,
	}
	data, err := marshalTLDR(tldr, "toml")
	if err != nil {
		t.Fatalf("marshalTLDR: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty TOML output")
	}

	data, err = marshalTLDR(tldr, "json")
	if err != nil {
		t.Fatalf("marshalTLDR JSON: %v", err)
	}
	if !strings.Contains(string(data), `"version_a"`) {
		t.Error("expected JSON to contain version_a")
	}

	data, err = marshalTLDR(tldr, "yaml")
	if err != nil {
		t.Fatalf("marshalTLDR YAML: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty YAML output")
	}
}

func TestMarshalStat(t *testing.T) {
	tldr := &core.TLDRResult{
		VersionA:     "1.0.0",
		VersionB:     "1.1.0",
		AdditionsLOC: 10,
		DeletionsLOC: 5,
	}
	data, err := marshalStat(tldr, "toml")
	if err != nil {
		t.Fatalf("marshalStat: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty TOML output")
	}

	data, err = marshalStat(tldr, "json")
	if err != nil {
		t.Fatalf("marshalStat JSON: %v", err)
	}
	if !strings.Contains(string(data), `"version_a"`) {
		t.Error("expected JSON to contain version_a")
	}
}

func TestPrintTLDR(t *testing.T) {
	tldr := &core.TLDRResult{
		VersionA:     "1.0.0",
		VersionB:     "1.1.0",
		AdditionsLOC: 10,
		DeletionsLOC: 5,
	}
	printTLDR(tldr, "/tmp/test.toml")
}

func TestNewStatCmd(t *testing.T) {
	c := NewStatCmd()
	if c.Use != "stat <version>" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Show CSS composition stats for a single version" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
	if c.Flag("format") == nil {
		t.Error("expected --format flag")
	}
	if c.Flag("output") == nil {
		t.Error("expected --output flag")
	}
}
