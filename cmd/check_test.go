package cmd

import (
	"strings"
	"testing"

	"github.com/bladeacer/ocd/internal/core"
)

func TestNewCheckCmd(t *testing.T) {
	c := NewCheckCmd()
	if c.Use != "check <version> <theme.css>" {
		t.Errorf("unexpected Use: %s", c.Use)
	}
	if c.Short != "Check a theme's CSS variables against a target Obsidian version" {
		t.Errorf("unexpected Short: %s", c.Short)
	}
	if c.Flag("format") == nil {
		t.Error("expected --format flag")
	}
	if c.Flag("output") == nil {
		t.Error("expected --output flag")
	}
	if c.Flag("silent") == nil {
		t.Error("expected --silent flag")
	}
}

func TestMarshalReportTOML(t *testing.T) {
	report := &core.VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		Missing:       []string{"--a"},
		Extra:         []string{"--b"},
	}
	data, err := marshalReport(report, "toml")
	if err != nil {
		t.Fatalf("marshalReport TOML: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty TOML output")
	}
}

func TestMarshalReportJSON(t *testing.T) {
	report := &core.VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		Missing:       []string{"--a"},
	}
	data, err := marshalReport(report, "json")
	if err != nil {
		t.Fatalf("marshalReport JSON: %v", err)
	}
	if !strings.Contains(string(data), `"missing"`) {
		t.Errorf("JSON output should contain missing, got %s", string(data))
	}
}

func TestMarshalReportYAML(t *testing.T) {
	report := &core.VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		Missing:       []string{"--a"},
	}
	data, err := marshalReport(report, "yaml")
	if err != nil {
		t.Fatalf("marshalReport YAML: %v", err)
	}
	if !strings.Contains(string(data), "missing") {
		t.Errorf("YAML output should contain missing, got %s", string(data))
	}
}

func TestMarshalReportDefaultFormat(t *testing.T) {
	report := &core.VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
	}
	data, err := marshalReport(report, "invalid")
	if err != nil {
		t.Fatalf("marshalReport default: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output for default format")
	}
}
