package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCSSVariables(t *testing.T) {
	css := `:root {
  --color-primary: blue;
  --spacing-sm: 8px;
}
.foo { color: var(--color-primary); }`
	vars := ExtractCSSVariables(css)
	if len(vars) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(vars))
	}
	if vars[0].Name != "--color-primary" {
		t.Errorf("expected --color-primary, got %q", vars[0].Name)
	}
	if vars[1].Name != "--spacing-sm" {
		t.Errorf("expected --spacing-sm, got %q", vars[1].Name)
	}
}

func TestExtractCSSVariablesDedup(t *testing.T) {
	css := `--a: 1;
--a: 2;
--b: 3;`
	vars := ExtractCSSVariables(css)
	if len(vars) != 2 {
		t.Fatalf("expected 2 unique variables, got %d", len(vars))
	}
}

func TestCompareVariables(t *testing.T) {
	targetVars := []CSSVariable{
		{Name: "--a", Value: "1"},
		{Name: "--b", Value: "2"},
		{Name: "--c", Value: "3"},
	}
	themeVars := []CSSVariable{
		{Name: "--a", Value: "red"},
		{Name: "--b", Value: "blue"},
		{Name: "--d", Value: "4"},
	}
	report := CompareVariables("1.0.0", "theme.css", targetVars, themeVars)
	if len(report.Missing) != 1 || report.Missing[0] != "--c" {
		t.Errorf("expected missing=[--c], got %v", report.Missing)
	}
	if len(report.Extra) != 1 || report.Extra[0] != "--d" {
		t.Errorf("expected extra=[--d], got %v", report.Extra)
	}
	if len(report.TargetVars) != 3 {
		t.Errorf("expected 3 target vars, got %d", len(report.TargetVars))
	}
	if len(report.ThemeVars) != 3 {
		t.Errorf("expected 3 theme vars, got %d", len(report.ThemeVars))
	}
}

func TestCompareVariablesNoMissing(t *testing.T) {
	targetVars := []CSSVariable{{Name: "--a", Value: "1"}}
	themeVars := []CSSVariable{{Name: "--a", Value: "red"}}
	report := CompareVariables("1.0.0", "theme.css", targetVars, themeVars)
	if len(report.Missing) != 0 {
		t.Errorf("expected no missing, got %v", report.Missing)
	}
	if len(report.Extra) != 0 {
		t.Errorf("expected no extra, got %v", report.Extra)
	}
}

func TestVariableReportString(t *testing.T) {
	report := &VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		Missing:       []string{"--a"},
		Extra:         []string{"--b"},
	}
	s := report.String()
	if s == "" {
		t.Error("expected non-empty String()")
	}
}

func TestVariableReportStringNoMissingNoExtra(t *testing.T) {
	report := &VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
	}
	s := report.String()
	if !strings.Contains(s, "covers all target variables") {
		t.Errorf("expected all-variables message, got %s", s)
	}
}

func TestVariableNames(t *testing.T) {
	vars := []CSSVariable{
		{Name: "--a", Value: "1"},
		{Name: "--b", Value: "2"},
	}
	names := VariableNames(vars)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "--a" || names[1] != "--b" {
		t.Errorf("expected [--a --b], got %v", names)
	}
}

func TestVariableNamesEmpty(t *testing.T) {
	names := VariableNames(nil)
	if len(names) != 0 {
		t.Errorf("expected 0 names for nil, got %d", len(names))
	}
}

func TestReadCSSVariables(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "theme.css")
	content := `:root {
  --color-primary: blue;
  --spacing-sm: 8px;
}`
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	vars, err := ReadCSSVariables(tmp)
	if err != nil {
		t.Fatalf("ReadCSSVariables: %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("expected 2 variables, got %d", len(vars))
	}
	if vars[0].Name != "--color-primary" {
		t.Errorf("expected --color-primary, got %q", vars[0].Name)
	}
}

func TestReadCSSVariablesNotFound(t *testing.T) {
	_, err := ReadCSSVariables("/nonexistent/path/theme.css")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestMarshalTOML(t *testing.T) {
	report := &VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		Missing:       []string{"--a"},
		Extra:         []string{"--b"},
	}
	data, err := report.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	if !strings.Contains(string(data), "1.0.0") {
		t.Errorf("TOML output should contain target version, got %s", string(data))
	}
}

func TestMarshalJSON(t *testing.T) {
	report := &VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		Missing:       []string{"--a"},
	}
	data, err := report.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(data), `"missing"`) {
		t.Errorf("JSON output should contain missing, got %s", string(data))
	}
}

func TestMarshalYAML(t *testing.T) {
	report := &VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		Missing:       []string{"--a"},
	}
	data, err := report.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if !strings.Contains(string(data), "missing") {
		t.Errorf("YAML output should contain missing, got %s", string(data))
	}
}
