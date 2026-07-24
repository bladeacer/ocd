package core

import (
	"testing"
)

func TestAnalyzeDiff(t *testing.T) {
	diff := `--- v1.0.0
+++ v1.1.0
@@ -1,5 +1,6 @@
 .class1 {
   color: red;
+  font-size: 14px;
 }
-.old-class {
-  display: none;
+.new-class {
+  display: block;
 }
 :root {
   --primary: blue;
+  --secondary: green;
-  --tertiary: yellow;
+  --primary: darkblue;
 }`

	r := AnalyzeDiff(diff)
	if r.AdditionsLOC != 5 {
		t.Errorf("AdditionsLOC = %d, want 5", r.AdditionsLOC)
	}
	if r.DeletionsLOC != 3 {
		t.Errorf("DeletionsLOC = %d, want 3", r.DeletionsLOC)
	}
	if len(r.SelectorsAdded) == 0 {
		t.Error("expected selectors added")
	}
	if len(r.CSSVariablesAdded) == 0 {
		t.Error("expected CSS variables added")
	}
	if r.ImportantCount != 0 {
		t.Errorf("ImportantCount = %d, want 0", r.ImportantCount)
	}
	if r.AverageSpecificity <= 0 {
		t.Error("expected positive average specificity")
	}
}

func TestAnalyzeDiffImportant(t *testing.T) {
	diff := `--- a
+++ b
@@ -1 +1 @@
+  color: red !important;`

	r := AnalyzeDiff(diff)
	if r.ImportantCount != 1 {
		t.Errorf("ImportantCount = %d, want 1", r.ImportantCount)
	}
}

func TestAnalyzeDiffEmpty(t *testing.T) {
	r := AnalyzeDiff("")
	if r.AdditionsLOC != 0 || r.DeletionsLOC != 0 {
		t.Error("expected zero counts for empty diff")
	}
}

func TestAnalyzeDiffSelectorAdded(t *testing.T) {
	diff := `--- a
+++ b
@@ -1 +1 @@
+.new-selector { }`

	r := AnalyzeDiff(diff)
	if len(r.SelectorsAdded) != 1 || r.SelectorsAdded[0] != ".new-selector" {
		t.Errorf("SelectorsAdded = %v, want [.new-selector]", r.SelectorsAdded)
	}
}

func TestAnalyzeDiffSelectorRemoved(t *testing.T) {
	diff := `--- a
+++ b
@@ -1 +1 @@
-.old-selector { }`

	r := AnalyzeDiff(diff)
	if len(r.SelectorsRemoved) != 1 || r.SelectorsRemoved[0] != ".old-selector" {
		t.Errorf("SelectorsRemoved = %v, want [.old-selector]", r.SelectorsRemoved)
	}
}

func TestAnalyzeDiffVariableChange(t *testing.T) {
	diff := `--- a
+++ b
@@ -1,2 +1,2 @@
-  --primary: red;
+  --primary: blue;`

	r := AnalyzeDiff(diff)
	if len(r.CSSVariablesChanged) == 0 {
		t.Fatal("expected variable change")
	}
	if r.CSSVariablesChanged[0].OldValue != "red" || r.CSSVariablesChanged[0].NewValue != "blue" {
		t.Errorf("variable change = %+v, want OldValue=red NewValue=blue", r.CSSVariablesChanged[0])
	}
	if len(r.CSSVariablesAdded) != 0 {
		t.Errorf("CSSVariablesAdded = %v, want empty", r.CSSVariablesAdded)
	}
	if len(r.CSSVariablesRemoved) != 0 {
		t.Errorf("CSSVariablesRemoved = %v, want empty", r.CSSVariablesRemoved)
	}
}

func TestAnalyzeDiffUnchangedVar(t *testing.T) {
	diff := `--- a
+++ b
@@ -1,2 +1,2 @@
-  --primary: red;
+  --primary: red;`

	r := AnalyzeDiff(diff)
	if len(r.CSSVariablesChanged) != 0 {
		t.Errorf("expected no changed variables for identical values, got %+v", r.CSSVariablesChanged)
	}
	if len(r.CSSVariablesAdded) != 0 {
		t.Errorf("CSSVariablesAdded = %v, want empty", r.CSSVariablesAdded)
	}
	if len(r.CSSVariablesRemoved) != 0 {
		t.Errorf("CSSVariablesRemoved = %v, want empty", r.CSSVariablesRemoved)
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		sel  string
		want float64
	}{
		{".class", 10},
		{"#id", 100},
		{"div", 1},
		{".class1 .class2", 20},
		{"#id .class", 110},
	}
	for _, tt := range tests {
		got := specificity(tt.sel)
		if got != tt.want {
			t.Errorf("specificity(%q) = %f, want %f", tt.sel, got, tt.want)
		}
	}
}

func TestExtractSelector(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{".class { }", ".class"},
		{"#id { color: red; }", "#id"},
		{"div { }", ""},
		{"  .nested .class { }", ".nested .class"},
		{"not a selector", ""},
	}
	for _, tt := range tests {
		got := extractSelector(tt.line)
		if got != tt.want {
			t.Errorf("extractSelector(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestIsVarDefinition(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"--primary: red;", true},
		{"  --secondary: blue;", true},
		{"color: red;", false},
		{"--invalid", false},
		{"var(--primary)", false},
	}
	for _, tt := range tests {
		got := isVarDefinition(tt.line)
		if got != tt.want {
			t.Errorf("isVarDefinition(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestMarshalMethods(t *testing.T) {
	r := &TLDRResult{
		VersionA:     "1.0.0",
		VersionB:     "1.1.0",
		AdditionsLOC: 10,
		DeletionsLOC: 5,
	}
	jsonData, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("expected non-empty JSON")
	}

	yamlData, err := r.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if len(yamlData) == 0 {
		t.Error("expected non-empty YAML")
	}

	tomlData, err := r.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	if len(tomlData) == 0 {
		t.Error("expected non-empty TOML")
	}
}

func TestTLDRString(t *testing.T) {
	r := &TLDRResult{
		VersionA:     "1.0.0",
		VersionB:     "1.1.0",
		AdditionsLOC: 10,
		DeletionsLOC: 5,
	}
	s := r.String()
	if s == "" {
		t.Error("expected non-empty string")
	}
}

func TestSemverBump(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		{"1.0.0", "2.0.0", "major"},
		{"1.0.0", "1.1.0", "minor"},
		{"1.0.0", "1.0.1", "patch"},
		{"1.0.0", "1.0.0", "none"},
		{"abc", "def", "unknown"},
	}
	for _, tt := range tests {
		got := SemverBump(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("SemverBump(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestCountColors(t *testing.T) {
	line := "color: #fff; background: rgb(0,0,0); border: hsl(0,0%,0%);"
	counts := countColors(line)
	if counts["hex"] != 1 {
		t.Errorf("hex count = %d, want 1", counts["hex"])
	}
	if counts["rgb"] != 1 {
		t.Errorf("rgb count = %d, want 1", counts["rgb"])
	}
	if counts["hsl"] != 1 {
		t.Errorf("hsl count = %d, want 1", counts["hsl"])
	}
}

func TestCountColorsNone(t *testing.T) {
	line := "font-size: 14px; display: block;"
	counts := countColors(line)
	if len(counts) != 0 {
		t.Errorf("expected no colors, got %v", counts)
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		v      string
		want   []int
		wantOK bool
	}{
		{"1.2.3", []int{1, 2, 3}, true},
		{"0.0.0", []int{0, 0, 0}, true},
		{"10.20.30", []int{10, 20, 30}, true},
		{"1.2", []int{1, 2, 0}, true},
		{"abc", []int{0, 0, 0}, false},
		{"1.2.3.4", []int{1, 2, 3}, true},
	}
	for _, tt := range tests {
		got, ok := parseVersion(tt.v)
		if ok != tt.wantOK {
			t.Errorf("parseVersion(%q) ok = %v, want %v", tt.v, ok, tt.wantOK)
			continue
		}
		if len(got) != 3 {
			t.Errorf("parseVersion(%q) len = %d, want 3", tt.v, len(got))
			continue
		}
		for i := 0; i < 3; i++ {
			if got[i] != tt.want[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.v, i, got[i], tt.want[i])
			}
		}
	}
}

func TestAnalyzeDiffColorCounts(t *testing.T) {
	diff := `--- a
+++ b
@@ -1 +1 @@
+  color: #fff; background: rgb(0,0,0);`
	r := AnalyzeDiff(diff)
	if r.ColorCounts == nil {
		t.Fatal("expected color counts")
	}
	if r.ColorCounts["hex"] != 1 {
		t.Errorf("hex count = %d, want 1", r.ColorCounts["hex"])
	}
	if r.ColorCounts["rgb"] != 1 {
		t.Errorf("rgb count = %d, want 1", r.ColorCounts["rgb"])
	}
}

func TestAnalyzeDiffOklch(t *testing.T) {
	diff := `--- a
+++ b
@@ -1 +1 @@
+  color: oklch(0.5 0.2 180);`
	r := AnalyzeDiff(diff)
	if r.ColorCounts == nil {
		t.Fatal("expected color counts")
	}
	if r.ColorCounts["oklch"] != 1 {
		t.Errorf("oklch count = %d, want 1", r.ColorCounts["oklch"])
	}
}

func TestAnalyzeDiffVarOnly(t *testing.T) {
	diff := `--- a
+++ b
@@ -1,2 +1,2 @@
-  --x: red;
+  --y: blue;`
	r := AnalyzeDiff(diff)
	if len(r.CSSVariablesAdded) != 1 || r.CSSVariablesAdded[0] != "--y" {
		t.Errorf("CSSVariablesAdded = %v, want [--y]", r.CSSVariablesAdded)
	}
	if len(r.CSSVariablesRemoved) != 1 || r.CSSVariablesRemoved[0] != "--x" {
		t.Errorf("CSSVariablesRemoved = %v, want [--x]", r.CSSVariablesRemoved)
	}
	if len(r.CSSVariablesChanged) != 0 {
		t.Errorf("CSSVariablesChanged = %v, want []", r.CSSVariablesChanged)
	}
}

func TestAnalyzeCSS(t *testing.T) {
	css := ".foo {\n  color: red;\n  padding: 8px;\n}\n"
	r := AnalyzeCSS(css)
	if r == nil {
		t.Fatal("AnalyzeCSS returned nil")
	}
	if r.AdditionsLOC == 0 {
		t.Errorf("AdditionsLOC = %d, want >0", r.AdditionsLOC)
	}
	if r.TotalSelectorsAnalyzed == 0 {
		t.Error("TotalSelectorsAnalyzed should be >0 for .foo selector")
	}
	if r.AverageSpecificity <= 0 {
		t.Error("expected positive average specificity")
	}
}
