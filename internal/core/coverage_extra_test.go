package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestReadCustomPropMissingColon(t *testing.T) {
	_, _, _, ok := readCustomProp("a{--x}", 1)
	if ok {
		t.Error("a custom property without a colon must not be ok")
	}

	_, _, _, ok = readCustomProp("a{--}", 1)
	if ok {
		t.Error("a bare -- must not be ok")
	}

	_, _, _, ok = readCustomProp("a{--x }", 1)
	if ok {
		t.Error("a property name followed by space without colon must not be ok")
	}
}

func TestReadCustomPropFinalDeclaration(t *testing.T) {
	name, value, next, ok := readCustomProp("a{--x:1}", 2)
	if !ok || name != "--x" || value != "1" || next != 7 {
		t.Errorf("readCustomProp = %q %q %d %v, want --x 1 7 true", name, value, next, ok)
	}
}

func TestReadThemeFileReadError(t *testing.T) {
	src := &ThemeSource{}
	if err := readThemeFile(src, filepath.Join(t.TempDir(), "missing.css")); err == nil {
		t.Error("expected an error reading a missing theme file")
	}
	if src.Index != nil {
		t.Error("a failed read must not set an index")
	}
}

func TestReadThemeFolderRecursive(t *testing.T) {
	root := writeTree(t, map[string]string{
		"theme.css":   `body{--a:1}`,
		"deep/nested/b.css": `.b{--b:2}`,
		"deep/nested/c.css": `.c{--c:3}`,
	})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if len(src.Files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(src.Files), src.Files)
	}
	if !src.Has("--b") || !src.Has("--c") {
		t.Error("nested css files should be read")
	}
}

func TestCollectCSSFilesSortsAndPrefersThemeCSS(t *testing.T) {
	files, err := collectCSSFiles(writeTree(t, map[string]string{
		"z.css":   `.z{}`,
		"theme.css": `body{}`,
		"a.css":   `.a{}`,
	}))
	if err != nil {
		t.Fatalf("collectCSSFiles: %v", err)
	}
	if len(files) != 3 || filepath.Base(files[0]) != "theme.css" {
		t.Fatalf("theme.css should come first, got %v", files)
	}
	if files[1] != files[0]+"..a" && filepath.Base(files[1]) != "a.css" {
		t.Errorf("unexpected order: %v", files)
	}
}

func TestImportFileASAR(t *testing.T) {
	dir := t.TempDir()
	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = orig }()

	asarPath := writeASAR(t, []asarFile{{"app.css", "body{--a:1}"}})
	path, err := ImportFile(asarPath, "asar-label")
	if err != nil {
		t.Fatalf("ImportFile asar: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "body{--a:1}" {
		t.Errorf("imported asar content = %q", string(data))
	}
}

func TestExtractCSSRefreshPath(t *testing.T) {
	useTempCSSDir(t)
	orig := CSSDir
	CSSDir = t.TempDir()
	defer func() { CSSDir = orig }()

	// No cache entry, use a server that returns 404.
	origURL := asarReleaseURL
	ts := httptestServer404()
	defer ts.Close()
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	defer func() { asarReleaseURL = origURL }()

	if _, err := ExtractCSS("999.999.999"); err == nil {
		t.Error("expected error for nonexistent version via ExtractCSS")
	}
}

func httptestServer404() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestReleaseExistsDefaultStatus(t *testing.T) {
	useTempCSSDir(t)
	origClient := httpClient
	origURL := asarReleaseURL
	defer func() { httpClient = origClient; asarReleaseURL = origURL }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer ts.Close()
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()

	_, err := ReleaseExists("1.0.0")
	if err == nil {
		t.Error("expected error for non-OK/non-404 status")
	}
}

func TestExtractCSSAtomicFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "app.css")
	if err := extractCSSAtomic(filepath.Join(dir, "missing.asar"), dest); err == nil {
		t.Error("expected error extracting a missing asar")
	}
}

func TestCopyFileAtomicAndTempPathIn(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.css")
	if err := os.WriteFile(src, []byte("body{--a:1}"), 0644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "out.css")
	if err := copyFileAtomic(src, dst); err != nil {
		t.Fatalf("copyFileAtomic: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "body{--a:1}" {
		t.Errorf("copied content = %q", string(data))
	}
}

func TestParallelFor(t *testing.T) {
	var mu sync.Mutex
	var seen []int
	ParallelFor(5, 2, func(i int) {
		mu.Lock()
		seen = append(seen, i)
		mu.Unlock()
	})
	if len(seen) != 5 {
		t.Errorf("ParallelFor ran %d iterations, want 5", len(seen))
	}

	// n <= 0 is a no-op.
	ParallelFor(0, 2, func(i int) {
		t.Error("should not run for n=0")
	})
	ParallelFor(-1, 2, func(i int) {
		t.Error("should not run for n<0")
	})
}

func TestEnsureVersionCached(t *testing.T) {
	useTempCSSDir(t)
	seedCSSFile(t, "1.0.0", "body{}")
	if status, err := ensureVersion("1.0.0", 0, false); status != SweepCached || err != nil {
		t.Errorf("fresh entry = %v, %v; want cached", status, err)
	}
}

func TestAnalyzeCSSCountsLOC(t *testing.T) {
	r := AnalyzeCSS(".a { color: red; }\n.b { color: blue; }\n")
	if r.AdditionsLOC != 3 {
		t.Errorf("AdditionsLOC = %d, want 3", r.AdditionsLOC)
	}
}

func TestTLDRStringExtraParts(t *testing.T) {
	r := &TLDRResult{
		VersionA:                 "1.0.0",
		VersionB:                 "1.1.0",
		AdditionsLOC:             10,
		DeletionsLOC:             5,
		ImportantCount:           3,
		TotalSelectorsAnalyzed:   2,
		Specificities:            []float64{10, 20},
		ColorCounts:              map[string]int{"hex": 2},
	}
	out := r.String()
	if out == "" {
		t.Error("String() should not be empty")
	}
	if !strings.Contains(out, "1.0.0") {
		t.Error("String should name version A")
	}
}

func TestTLDRStatStringExtraParts(t *testing.T) {
	r := &TLDRResult{
		VersionA:                "1.0.0",
		AdditionsLOC:            10,
		TotalSelectorsAnalyzed:  1,
		Specificities:           []float64{5},
	}
	out := r.StatString()
	if out == "" {
		t.Error("StatString() should not be empty")
	}
}

func TestExportTLDRAllFormats(t *testing.T) {
	dir := t.TempDir()
	r := &TLDRResult{VersionA: "1.0.0", VersionB: "1.1.0", AdditionsLOC: 5}
	path := filepath.Join(dir, "out.toml")
	if err := ExportTLDR(r, path, "toml"); err != nil {
		t.Fatalf("ExportTLDR: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "version_a") {
		t.Errorf("expected version_a in export, got %s", string(data))
	}

	if err := ExportTLDR(r, path, "json"); err != nil {
		t.Fatalf("ExportTLDR json: %v", err)
	}
	if err := ExportTLDR(r, path, "yaml"); err != nil {
		t.Fatalf("ExportTLDR yaml: %v", err)
	}
}

func TestTLDRMarshalTOMLFull(t *testing.T) {
	r := &TLDRResult{
		VersionA:               "1.0.0",
		VersionB:               "1.1.0",
		VersionADate:           "2023-01-01",
		VersionBDate:           "2023-02-01",
		SemverBump:             "minor",
		AdditionsLOC:           1,
		DeletionsLOC:           1,
		SelectorsAdded:         []string{".a"},
		SelectorsRemoved:       []string{".b"},
		CSSVariablesAdded:      []string{"--a"},
		CSSVariablesRemoved:    []string{"--b"},
		CSSVariablesChanged:    []VariableChange{{Name: "--c", OldValue: "1", NewValue: "2"}},
		ImportantCount:         1,
		AverageSpecificity:     10.5,
		TotalSelectorsAnalyzed: 1,
		ColorCounts:            map[string]int{"hex": 1},
		Specificities:          []float64{10},
	}
	out, err := r.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	s := string(out)
	for _, want := range []string{"version_a_date", "version_b_date", "semver_bump", "important_count", "average_specificity", "total_selectors_analyzed", "color_counts"} {
		if !strings.Contains(s, want) {
			t.Errorf("TOML missing %q: %s", want, s)
		}
	}
}

func TestVariablesMarshalTOML(t *testing.T) {
	r := &VariableReport{
		TargetVersion: "1.0.0",
		ThemePath:     "theme.css",
		TargetVars:    []CSSVariable{{Name: "--a", Value: "1"}},
		ThemeVars:     []CSSVariable{{Name: "--b", Value: "2"}},
		Missing:       []string{"--c"},
		Extra:         []string{"--d"},
	}
	out, err := r.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	s := string(out)
	for _, want := range []string{"target_variables", "theme_variables", "missing", "extra", "missing_count", "extra_count"} {
		if !strings.Contains(s, want) {
			t.Errorf("TOML missing %q: %s", want, s)
		}
	}
}
