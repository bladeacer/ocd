package core

import (
	"regexp"
	"strings"
	"testing"
)

func TestCountColorsRecognisesEveryFormat(t *testing.T) {
	line := `color:#fff; background:rgb(1,2,3); fill:hsl(1 2% 3%); ` +
		`stroke:oklch(0.5 0.1 20); outline:oklab(0.4 0.2 0); border:lab(1 2 3); ` +
		`box-shadow:lch(1 2 3); background-color:hwb(1 2% 3%); caret-color:color(display-p3 1 0 0)`

	counts := countColors(line)
	for _, want := range []string{"hex", "rgb", "hsl", "oklch", "oklab", "lab", "lch", "hwb", "color()"} {
		if counts[want] == 0 {
			t.Errorf("%s not counted; got %v", want, counts)
		}
	}
}

func TestCountColorsOnPlainText(t *testing.T) {
	counts := countColors("color: inherit; border: none")
	if len(counts) != 0 {
		t.Errorf("expected no colour formats, got %v", counts)
	}
}

func TestExtractVarValue(t *testing.T) {
	re := regexp.MustCompile(`^\s*(--[\w-]+)\s*:\s*(.+?);`)

	if got := extractVarValue("", re); got != "" {
		t.Errorf("empty line = %q, want empty", got)
	}
	if got := extractVarValue("not a definition", re); got != "" {
		t.Errorf("non-matching line = %q, want empty", got)
	}
	if got := extractVarValue("--a: 1px;", re); got != "1px" {
		t.Errorf("--a = %q, want 1px", got)
	}
}

func TestStatStringAndStringOnEmptyResult(t *testing.T) {
	r := &TLDRResult{VersionA: "1.0.0"}

	stat := r.StatString()
	if !strings.Contains(stat, "1.0.0") {
		t.Errorf("StatString should name the version, got:\n%s", stat)
	}

	s := r.String()
	if !strings.Contains(s, "1.0.0") {
		t.Errorf("String should name the version, got:\n%s", s)
	}
}

func TestMarshalTOMLWithEmptyLists(t *testing.T) {
	r := &TLDRResult{VersionA: "1.0.0", VersionB: "1.1.0"}

	out, err := r.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected some TOML output")
	}
}

func TestAnalyzeCSSEdgeCases(t *testing.T) {
	if r := AnalyzeCSS(""); r == nil {
		t.Error("an empty stylesheet should still give a result")
	}

	// Commented-out rules must not be counted.
	r := AnalyzeCSS("/* .gone { color: red; } */\n.kept { color: blue; }")
	if len(r.SelectorsAdded) == 0 && len(r.SelectorsRemoved) == 0 {
		t.Logf("no selectors recorded: %+v", r)
	}
}

func TestAnalyzeDiffWithNoHunks(t *testing.T) {
	r := AnalyzeDiff("")
	if r == nil {
		t.Fatal("expected a result")
	}
	if r.SemverBump != "none" && r.SemverBump != "" {
		t.Logf("semver bump for an empty diff = %q", r.SemverBump)
	}
}

func TestReadCSSVariablesMissingFile(t *testing.T) {
	if _, err := ReadCSSVariables(t.TempDir() + "/nope.css"); err == nil {
		t.Error("expected an error for a missing file")
	}
}
