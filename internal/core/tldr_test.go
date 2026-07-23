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
