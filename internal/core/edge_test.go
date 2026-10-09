package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCSSTrailingBackslash(t *testing.T) {
	// A backslash at the very end of the input must not read past it.
	idx := ParseCSS("body{--a:1}\\")
	if !idx.HasVariable("--a") {
		t.Errorf("variable lost: %v", idx.Variables)
	}
}

func TestParseCSSUnterminatedString(t *testing.T) {
	// An unterminated string ends at the newline, as CSS requires.
	idx := ParseCSS("a{--before:1}\nbody{content:\"open\n--after:2}")
	if !idx.HasVariable("--before") {
		t.Errorf("declaration before the string lost: %v", idx.Variables)
	}
}

func TestSkipStringEdges(t *testing.T) {
	if got := skipString("\"abc\"", 0); got != 5 {
		t.Errorf("closed string = %d, want 5", got)
	}
	if got := skipString("'a\nb'", 0); got != 2 {
		t.Errorf("string ending at a newline = %d, want 2", got)
	}
	if got := skipString("\"abc", 0); got != 4 {
		t.Errorf("unterminated string = %d, want the input length", got)
	}
}

func TestSelectorCoversEdgeCases(t *testing.T) {
	cases := []struct {
		theme, target string
		want          bool
	}{
		{"", "", false},
		{".a", "", false},
		{"", ".a", false},
		{".a >", ".a > .b", true},
		{".a > .b", ".a .b", false},
		{".a + .b", ".a + .b", true},
		{".a ~ .b", ".a ~ .c", false},
		{"#id", "#id", true},
		{"#id", "#other", false},
		{".a", "#a", false},
		{"a.b", "a.b", true},
		{"a.b", "a", false},
	}
	for _, tc := range cases {
		if got := SelectorCovers(tc.theme, tc.target); got != tc.want {
			t.Errorf("SelectorCovers(%q, %q) = %v, want %v", tc.theme, tc.target, got, tc.want)
		}
	}
}

func TestSelectorReportStringTruncation(t *testing.T) {
	var target, theme strings.Builder
	for i := 0; i < SelectorListLimit+5; i++ {
		target.WriteString(".t" + itoa(i) + "{}")
		theme.WriteString(".x" + itoa(i) + "{}")
	}
	rep := CompareSelectors("1.0.0", "t.css", ParseCSS(target.String()), ParseCSS(theme.String()))
	out := rep.String()

	if !strings.Contains(out, "and 5 more") {
		t.Errorf("expected the uncovered list to note the remainder:\n%s", out)
	}
}

func TestSelectorReportMarshalNilCounts(t *testing.T) {
	rep := &SelectorReport{}
	if _, err := rep.MarshalJSON(); err != nil {
		t.Errorf("MarshalJSON: %v", err)
	}
	if _, err := rep.MarshalYAML(); err != nil {
		t.Errorf("MarshalYAML: %v", err)
	}
	if _, err := rep.MarshalTOML(); err != nil {
		t.Errorf("MarshalTOML: %v", err)
	}
}

func TestReadThemeFileRemovedMidWalk(t *testing.T) {
	// A file that disappears between the walk and the read is an error,
	// not a panic.
	root := writeTree(t, map[string]string{"theme.css": "body{--a:1}"})

	src := &ThemeSource{Path: root}
	if err := readThemeFile(src, filepath.Join(root, "gone.css")); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestCSSIndexMergeNil(t *testing.T) {
	idx := ParseCSS("body{--a:1}.x{}")
	idx.merge(nil)
	if !idx.HasVariable("--a") {
		t.Error("merging nil must not change the index")
	}
}

func TestParseCSSWithOnlyAStr(t *testing.T) {
	idx := ParseCSS(`"`)
	if len(idx.Variables) != 0 {
		t.Errorf("expected no variables, got %v", idx.Variables)
	}
}

func TestEnsureCSSOfMissingVersionIsFalse(t *testing.T) {
	useTempCSSDir(t)
	if CSSCached("1.2.3") {
		t.Error("an unknown version must not be cached")
	}
	if CSSFresh("1.2.3", 0) {
		t.Error("an unknown version is never fresh")
	}
}

func TestExtractCSSAtomicLeavesNoTempOnBadASAR(t *testing.T) {
	useTempCSSDir(t)

	dir := t.TempDir()
	asar := filepath.Join(dir, "bad.asar")
	if err := os.WriteFile(asar, []byte("junk"), 0644); err != nil {
		t.Fatal(err)
	}
	dest := CSSPath("1.0.0")
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		t.Fatal(err)
	}

	if err := extractCSSAtomic(asar, dest); err == nil {
		t.Fatal("expected an extraction error")
	}
	if _, err := os.Stat(dest); err == nil {
		t.Error("no file should have been written")
	}
	entries, err := os.ReadDir(filepath.Dir(dest))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp file %q left behind", e.Name())
		}
	}
}
