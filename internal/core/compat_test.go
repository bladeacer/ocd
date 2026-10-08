package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupCSSCache(t *testing.T, dir string, versions map[string]string) {
	t.Helper()
	for v, css := range versions {
		verDir := filepath.Join(dir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}
	orig := CSSDir
	CSSDir = dir
	t.Cleanup(func() { CSSDir = orig })
}

func TestFindOriginsSelector(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".old { color: red; }",
		"1.1.0": ".old { color: red; }\n.messageBar { color: blue; }",
		"1.2.0": ".old { color: red; }\n.messageBar { color: green; }",
	})

	targets := []string{".old", ".messageBar", ".new"}
	results := FindOrigins(targets, false, []string{"1.0.0", "1.1.0", "1.2.0"})

	found := make(map[string]TargetOrigin)
	for _, r := range results {
		found[r.Name] = r
	}

	if found[".old"].Introduced != "1.0.0" {
		t.Errorf(".old: expected introduced 1.0.0, got %s", found[".old"].Introduced)
	}
	if found[".messageBar"].Introduced != "1.1.0" {
		t.Errorf(".messageBar: expected introduced 1.1.0, got %s", found[".messageBar"].Introduced)
	}
	if found[".new"].Found {
		t.Error(".new: expected not found")
	}
	if len(found[".messageBar"].Versions) != 2 {
		t.Errorf(".messageBar: expected 2 versions, got %d: %v", len(found[".messageBar"].Versions), found[".messageBar"].Versions)
	}
	if found[".old"].Kind != "selector" {
		t.Errorf(".old: expected kind=selector, got %s", found[".old"].Kind)
	}
}

func TestFindOriginsVariable(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ":root {\n  --old: red;\n}",
		"1.1.0": ":root {\n  --old: red;\n  --new: blue;\n}",
	})

	targets := []string{"--old", "--new", "--missing"}
	results := FindOrigins(targets, true, []string{"1.0.0", "1.1.0"})

	found := make(map[string]TargetOrigin)
	for _, r := range results {
		found[r.Name] = r
	}

	if found["--old"].Introduced != "1.0.0" {
		t.Errorf("--old: expected introduced 1.0.0, got %s", found["--old"].Introduced)
	}
	if found["--new"].Introduced != "1.1.0" {
		t.Errorf("--new: expected introduced 1.1.0, got %s", found["--new"].Introduced)
	}
	if found["--missing"].Found {
		t.Error("--missing: expected not found")
	}
	if found["--old"].Kind != "variable" {
		t.Errorf("--old: expected kind=variable, got %s", found["--old"].Kind)
	}
}

func TestFindOriginsEmptyTargets(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": "",
	})

	results := FindOrigins(nil, false, []string{"1.0.0"})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFindOriginsVersionWithoutCSS(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.1.0": ".foo { }",
	})

	results := FindOrigins([]string{".foo"}, false, []string{"1.0.0", "1.1.0"})
	if !results[0].Found {
		t.Error("expected .foo found in 1.1.0")
	}
	if results[0].Introduced != "1.1.0" {
		t.Errorf("expected introduced 1.1.0, got %s", results[0].Introduced)
	}
}

func TestFindOriginsUnsortedVersions(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".target { color: blue; }",
		"1.1.0": ".foo { color: red; }",
		"1.2.0": ".foo { color: red; }",
	})

	results := FindOrigins([]string{".target"}, false, []string{"1.2.0", "1.0.0", "1.1.0"})
	if results[0].Introduced != "1.0.0" {
		t.Errorf("expected introduced 1.0.0 (earliest), got %s", results[0].Introduced)
	}
}

func TestFindOriginsWithComments(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": "/* .target { display: none; } */\n.real { color: blue; }",
	})

	results := FindOrigins([]string{".target"}, false, []string{"1.0.0"})
	if results[0].Found {
		t.Error("expected .target NOT found when only in comment")
	}
}

func TestCheckCompatibilityStrictPass(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".foo { color: red; }",
		"1.1.0": ".foo { color: red; }\n.bar { color: blue; }",
	})

	result := CheckCompatibility([]string{".foo"}, false, "1.1.0", CompatModeStrict, []string{"1.0.0", "1.1.0"})
	if !result.Compatible {
		t.Error("expected compatible=true for strict mode with no violations")
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result.Violations))
	}
	if result.Mode != "strict" {
		t.Errorf("expected mode=strict, got %s", result.Mode)
	}
}

func TestCheckCompatibilityStrictFail(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".foo { color: red; }",
		"1.1.0": ".foo { color: red; }\n.bar { color: blue; }",
	})

	result := CheckCompatibility([]string{".bar"}, false, "1.0.0", CompatModeStrict, []string{"1.0.0", "1.1.0"})
	if result.Compatible {
		t.Error("expected compatible=false for strict mode with violations")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	if result.Violations[0].Introduced != "1.1.0" {
		t.Errorf("expected introduced 1.1.0, got %s", result.Violations[0].Introduced)
	}
}

func TestCheckCompatibilityRelaxed(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".foo { color: red; }",
		"1.1.0": ".foo { color: red; }\n.bar { color: blue; }",
	})

	result := CheckCompatibility([]string{".bar"}, false, "1.0.0", CompatModeRelaxed, []string{"1.0.0", "1.1.0"})
	if !result.Compatible {
		t.Error("expected compatible=true for relaxed mode even with warnings")
	}
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Introduced != "1.1.0" {
		t.Errorf("expected introduced 1.1.0, got %s", result.Warnings[0].Introduced)
	}
}

func TestCheckCompatibilityUnknown(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".foo { color: red; }",
	})

	result := CheckCompatibility([]string{".missing"}, false, "1.0.0", CompatModeStrict, []string{"1.0.0"})
	if len(result.Unknown) != 1 {
		t.Errorf("expected 1 unknown, got %d", len(result.Unknown))
	}
	if result.Unknown[0].Found {
		t.Error("expected unknown to have Found=false")
	}
}

func TestCheckCompatibilityAllValid(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".foo { color: red; }",
		"1.1.0": ".foo { color: red; }",
	})

	result := CheckCompatibility([]string{".foo"}, false, "1.1.0", CompatModeStrict, []string{"1.0.0", "1.1.0"})
	if !result.Compatible {
		t.Error("expected compatible=true")
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations, got %d", len(result.Violations))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
	}
	if len(result.Unknown) != 0 {
		t.Errorf("expected 0 unknown, got %d", len(result.Unknown))
	}
}

func TestCheckCompatibilitySameVersion(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".foo { color: red; }",
		"1.1.0": ".foo { color: red; }\n.bar { color: blue; }",
	})

	result := CheckCompatibility([]string{".bar"}, false, "1.1.0", CompatModeStrict, []string{"1.0.0", "1.1.0"})
	if !result.Compatible {
		t.Error("expected compatible=true when introduced equals target")
	}
}

func TestCheckCompatibilityInvalidTargetVersion(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": ".foo { color: red; }",
		"1.1.0": ".foo { color: red; }\n.bar { color: blue; }",
	})

	result := CheckCompatibility([]string{".bar"}, false, "invalid", CompatModeStrict, []string{"1.0.0", "1.1.0"})
	if len(result.Violations) != 0 {
		t.Errorf("expected 0 violations with invalid target version, got %d", len(result.Violations))
	}
}

func TestCheckCompatibilityEmptyVersions(t *testing.T) {
	dir := t.TempDir()
	setupCSSCache(t, dir, map[string]string{
		"1.0.0": "",
	})

	result := CheckCompatibility([]string{".foo"}, false, "1.0.0", CompatModeStrict, nil)
	if result.Compatible {
		t.Error("expected compatible=false when all targets unknown")
	}
	if len(result.Unknown) != 1 {
		t.Errorf("expected 1 unknown, got %d", len(result.Unknown))
	}
}

func TestCompatCheckResultString(t *testing.T) {
	r := &CompatCheckResult{
		TargetVersion: "1.0.0",
		Mode:          "strict",
		Compatible:    false,
		Violations: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
		},
		AllChecked: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
		},
	}
	s := r.String()
	if !strings.Contains(s, "1.0.0") {
		t.Errorf("expected target version in output, got %s", s)
	}
	if !strings.Contains(s, "incompatible") {
		t.Errorf("expected incompatible in output, got %s", s)
	}
	if !strings.Contains(s, ".bar") {
		t.Errorf("expected .bar in output, got %s", s)
	}
	if !strings.Contains(s, "Note:") {
		t.Error("expected caveat note in output")
	}
}

func TestCompatCheckResultStringCompatible(t *testing.T) {
	r := &CompatCheckResult{
		TargetVersion: "1.0.0",
		Mode:          "relaxed",
		Compatible:    true,
		AllChecked: []TargetOrigin{
			{Name: ".foo", Kind: "selector", Introduced: "1.0.0", Found: true},
		},
	}
	s := r.String()
	if !strings.Contains(s, "compatible") {
		t.Errorf("expected compatible in output, got %s", s)
	}
	if !strings.Contains(s, "All targets") {
		t.Errorf("expected summary in output, got %s", s)
	}
}

func TestCompatCheckResultStringWithUnknown(t *testing.T) {
	r := &CompatCheckResult{
		TargetVersion: "1.0.0",
		Mode:          "strict",
		Compatible:    false,
		Violations: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
		},
		Unknown: []TargetOrigin{
			{Name: ".missing", Kind: "selector", Found: false},
		},
		AllChecked: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
			{Name: ".missing", Kind: "selector", Found: false},
		},
	}
	s := r.String()
	if !strings.Contains(s, "Unknown origin") {
		t.Errorf("expected unknown origin section in output, got %s", s)
	}
	if !strings.Contains(s, ".missing") {
		t.Errorf("expected .missing in output, got %s", s)
	}
}

func TestCompatCheckResultMarshalJSON(t *testing.T) {
	r := &CompatCheckResult{
		TargetVersion: "1.0.0",
		Mode:          "strict",
		Compatible:    false,
		Violations: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
		},
		AllChecked: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
		},
	}
	data, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(data), `"compatible": false`) {
		t.Errorf("expected compatible=false in JSON, got %s", string(data))
	}
}

func TestCompatCheckResultMarshalYAML(t *testing.T) {
	r := &CompatCheckResult{
		TargetVersion: "1.0.0",
		Mode:          "strict",
		Compatible:    true,
		AllChecked: []TargetOrigin{
			{Name: ".foo", Kind: "selector", Introduced: "1.0.0", Found: true},
		},
	}
	data, err := r.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if !strings.Contains(string(data), "target_version") {
		t.Errorf("expected target_version in YAML, got %s", string(data))
	}
}

func TestCompatCheckResultMarshalTOML(t *testing.T) {
	r := &CompatCheckResult{
		TargetVersion: "1.0.0",
		Mode:          "strict",
		Compatible:    false,
		Violations: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
		},
		Warnings: []TargetOrigin{
			{Name: ".baz", Kind: "selector", Introduced: "1.0.5", Found: true},
		},
		Unknown: []TargetOrigin{
			{Name: ".missing", Kind: "selector", Found: false},
		},
		AllChecked: []TargetOrigin{
			{Name: ".bar", Kind: "selector", Introduced: "1.1.0", Found: true},
		},
	}
	data, err := r.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `target_version = "1.0.0"`) {
		t.Errorf("expected target_version in TOML, got %s", s)
	}
	if !strings.Contains(s, "compatible = false") {
		t.Errorf("expected compatible = false in TOML, got %s", s)
	}
	if !strings.Contains(s, "violations") {
		t.Errorf("expected violations in TOML, got %s", s)
	}
	if !strings.Contains(s, "warnings") {
		t.Errorf("expected warnings in TOML, got %s", s)
	}
	if !strings.Contains(s, "unknown") {
		t.Errorf("expected unknown in TOML, got %s", s)
	}
}

func TestCompatCheckResultEncodeTOMLError(t *testing.T) {
	r := &CompatCheckResult{
		TargetVersion: "1.0.0",
		Mode:          "strict",
		Compatible:    true,
	}
	err := r.encodeTOML(failingWriter{})
	if err == nil {
		t.Error("expected error from failing writer")
	}
}
