package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bladeacer/ocd/internal/models"
)

func TestPublicDesktopVersions(t *testing.T) {
	tests := []struct {
		name string
		in   []models.RSSVersion
		want []string
	}{
		{
			name: "empty",
			in:   nil,
			want: nil,
		},
		{
			name: "all public desktop",
			in: []models.RSSVersion{
				{Version: "1.0.0", Type: models.Desktop, IsEarly: false},
				{Version: "1.1.0", Type: models.Desktop, IsEarly: false},
			},
			want: []string{"1.0.0", "1.1.0"},
		},
		{
			name: "mixed types and early",
			in: []models.RSSVersion{
				{Version: "1.0.0", Type: models.Desktop, IsEarly: false},
				{Version: "1.0.1", Type: models.Desktop, IsEarly: true},
				{Version: "1.0.2", Type: models.Mobile, IsEarly: false},
				{Version: "1.0.3", Type: models.Mobile, IsEarly: true},
			},
			want: []string{"1.0.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PublicDesktopVersions(tt.in)
			if !stringsEqual(got, tt.want) {
				t.Errorf("PublicDesktopVersions(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func stringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSortVersions(t *testing.T) {
	t.Run("ascending", func(t *testing.T) {
		in := []string{"1.2.0", "1.0.0", "1.1.0"}
		got := SortVersions(in)
		want := []string{"1.0.0", "1.1.0", "1.2.0"}
		if !stringsEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("already sorted", func(t *testing.T) {
		in := []string{"1.0.0", "1.0.1", "1.1.0"}
		got := SortVersions(in)
		if !stringsEqual(got, in) {
			t.Errorf("got %v, want %v", got, in)
		}
	})

	t.Run("empty", func(t *testing.T) {
		got := SortVersions(nil)
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("single", func(t *testing.T) {
		got := SortVersions([]string{"1.0.0"})
		if !stringsEqual(got, []string{"1.0.0"}) {
			t.Errorf("got %v, want [1.0.0]", got)
		}
	})

	t.Run("with invalid versions", func(t *testing.T) {
		in := []string{"abc", "1.0.0", "xyz"}
		got := SortVersions(in)
		// Invalid versions go to the end, in string order.
		if !stringsEqual(got, []string{"1.0.0", "abc", "xyz"}) {
			t.Errorf("got %v, want [1.0.0 abc xyz]", got)
		}
	})

	t.Run("both invalid", func(t *testing.T) {
		in := []string{"zzz", "aaa"}
		got := SortVersions(in)
		if !stringsEqual(got, []string{"aaa", "zzz"}) {
			t.Errorf("got %v, want [aaa zzz]", got)
		}
	})

	t.Run("invalid before valid", func(t *testing.T) {
		in := []string{"abc", "1.0.0"}
		got := SortVersions(in)
		if !stringsEqual(got, []string{"1.0.0", "abc"}) {
			t.Errorf("got %v, want [1.0.0 abc]", got)
		}
	})

	t.Run("valid before invalid", func(t *testing.T) {
		in := []string{"1.0.0", "abc"}
		got := SortVersions(in)
		if !stringsEqual(got, []string{"1.0.0", "abc"}) {
			t.Errorf("got %v, want [1.0.0 abc]", got)
		}
	})

	t.Run("duplicates", func(t *testing.T) {
		in := []string{"1.1.0", "1.0.0", "1.0.0"}
		got := SortVersions(in)
		if !stringsEqual(got, []string{"1.0.0", "1.0.0", "1.1.0"}) {
			t.Errorf("got %v, want [1.0.0 1.0.0 1.1.0]", got)
		}
	})
}

func TestStripCSSComments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no comments",
			in:   ".foo { color: red; }",
			want: ".foo { color: red; }",
		},
		{
			name: "single comment",
			in:   "/* .foo */ .bar { color: red; }",
			want: " .bar { color: red; }",
		},
		{
			name: "multi-line comment",
			in:   "/* this is\na comment */ .bar { color: red; }",
			want: " .bar { color: red; }",
		},
		{
			name: "comment at end",
			in:   ".foo { color: red; } /* trailing */",
			want: ".foo { color: red; } ",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripCSSComments(tt.in)
			if got != tt.want {
				t.Errorf("stripCSSComments(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestContainsSelector(t *testing.T) {
	css := `.foo {
  color: red;
}
#bar {
  display: block;
}
.foo .baz {
  margin: 0;
}`
	if !ContainsSelector(css, ".foo") {
		t.Error("expected .foo to be found")
	}
	if !ContainsSelector(css, "#bar") {
		t.Error("expected #bar to be found")
	}
	if !ContainsSelector(css, ".foo .baz") {
		t.Error("expected .foo .baz to be found")
	}
	if ContainsSelector(css, ".missing") {
		t.Error("expected .missing to not be found")
	}
	if ContainsSelector(css, "") {
		t.Error("expected empty selector to return false")
	}
	if ContainsSelector("", ".foo") {
		t.Error("expected empty css to return false")
	}

	t.Run("selector inside comment is ignored", func(t *testing.T) {
		cssWithComment := `/* .messageBar { display: none; } */`
		if ContainsSelector(cssWithComment, ".messageBar") {
			t.Error("expected selector inside comment to not be found")
		}
	})

	t.Run("selector found despite comment", func(t *testing.T) {
		cssWithComment := `/* .messageBar { display: none; } */
.messageBar { display: block; }`
		if !ContainsSelector(cssWithComment, ".messageBar") {
			t.Error("expected .messageBar to be found outside comment")
		}
	})
}

func TestContainsVariable(t *testing.T) {
	css := `:root {
  --color-primary: blue;
  --spacing-sm: 8px;
}
.foo { color: var(--color-primary); }`
	if !ContainsVariable(css, "--color-primary") {
		t.Error("expected --color-primary to be found")
	}
	if !ContainsVariable(css, "--spacing-sm") {
		t.Error("expected --spacing-sm to be found")
	}
	if ContainsVariable(css, "--missing") {
		t.Error("expected --missing to not be found")
	}
	if ContainsVariable(css, "") {
		t.Error("expected empty name to return false")
	}
	if ContainsVariable("", "--color-primary") {
		t.Error("expected empty css to return false")
	}

	t.Run("variable inside comment is ignored", func(t *testing.T) {
		cssWithComment := `/* --color-primary: red; */`
		if ContainsVariable(cssWithComment, "--color-primary") {
			t.Error("expected variable inside comment to not be found")
		}
	})

	t.Run("variable found despite comment", func(t *testing.T) {
		cssWithComment := `/* --color-primary: red; */
:root {
  --color-primary: blue;
}`
		if !ContainsVariable(cssWithComment, "--color-primary") {
			t.Error("expected --color-primary to be found outside comment")
		}
	})
}

func TestListCachedVersions(t *testing.T) {
	t.Run("with cached CSS", func(t *testing.T) {
		dir := t.TempDir()
		for _, v := range []string{"1.0.0", "1.1.0"} {
			verDir := filepath.Join(dir, v)
			if err := os.MkdirAll(verDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte("body{}"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		// Subdirectory without app.css should be skipped.
		if err := os.MkdirAll(filepath.Join(dir, "no-css"), 0755); err != nil {
			t.Fatal(err)
		}

		orig := CSSDir
		CSSDir = dir
		defer func() { CSSDir = orig }()

		versions, err := ListCachedVersions()
		if err != nil {
			t.Fatalf("ListCachedVersions: %v", err)
		}
		if len(versions) != 2 {
			t.Errorf("expected 2 versions, got %d: %v", len(versions), versions)
		}
	})

	t.Run("no cache dir", func(t *testing.T) {
		orig := CSSDir
		CSSDir = filepath.Join(t.TempDir(), "nonexistent")
		defer func() { CSSDir = orig }()

		_, err := ListCachedVersions()
		if err == nil {
			t.Error("expected error for nonexistent cache dir")
		}
	})

	t.Run("with non-dir entries", func(t *testing.T) {
		dir := t.TempDir()
		verDir := filepath.Join(dir, "1.0.0")
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte("body{}"), 0644); err != nil {
			t.Fatal(err)
		}
		// Regular file (not a directory) - should be skipped.
		if err := os.WriteFile(filepath.Join(dir, "not-a-dir.txt"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		// Directory without app.css - should be skipped.
		if err := os.MkdirAll(filepath.Join(dir, "no-css"), 0755); err != nil {
			t.Fatal(err)
		}

		orig := CSSDir
		CSSDir = dir
		defer func() { CSSDir = orig }()

		versions, err := ListCachedVersions()
		if err != nil {
			t.Fatalf("ListCachedVersions: %v", err)
		}
		if len(versions) != 1 {
			t.Errorf("expected 1 version, got %d: %v", len(versions), versions)
		}
	})
}

func TestMergeVersions(t *testing.T) {
	tests := []struct {
		name        string
		rss         []string
		cached      []string
		want        []string
		checkSubset bool
	}{
		{
			name:   "both empty",
			rss:    nil,
			cached: nil,
			want:   nil,
		},
		{
			name:   "rss only",
			rss:    []string{"1.0.0", "1.1.0"},
			cached: nil,
			want:   []string{"1.0.0", "1.1.0"},
		},
		{
			name:   "cached only",
			rss:    nil,
			cached: []string{"1.0.0", "1.1.0"},
			want:   []string{"1.0.0", "1.1.0"},
		},
		{
			name:   "no overlap",
			rss:    []string{"1.0.0", "1.1.0"},
			cached: []string{"1.2.0", "1.3.0"},
			want:   []string{"1.0.0", "1.1.0", "1.2.0", "1.3.0"},
		},
		{
			name:   "with duplicates",
			rss:    []string{"1.0.0", "1.1.0"},
			cached: []string{"1.1.0", "1.2.0"},
			want:   []string{"1.0.0", "1.1.0", "1.2.0"},
		},
		{
			name:   "empty slices",
			rss:    []string{},
			cached: []string{},
			want:   nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeVersions(tt.rss, tt.cached)
			if !stringsEqual(got, tt.want) {
				t.Errorf("MergeVersions(%v, %v) = %v, want %v", tt.rss, tt.cached, got, tt.want)
			}
		})
	}
}

func TestFindOriginSelector(t *testing.T) {
	dir := t.TempDir()
	versions := []string{"1.0.0", "1.1.0", "1.2.0"}
	for _, v := range versions {
		verDir := filepath.Join(dir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		var css string
		switch v {
		case "1.0.0":
			css = ".other { color: red; }"
		case "1.1.0":
			css = ".messageBar { color: blue; }"
		case "1.2.0":
			css = ".messageBar { color: green; }"
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}

	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = orig }()

	result := FindOrigin(".messageBar", false, versions)
	if !result.Found {
		t.Fatal("expected Found=true")
	}
	if result.Introduced != "1.1.0" {
		t.Errorf("expected Introduced=1.1.0, got %s", result.Introduced)
	}
	if len(result.Versions) != 2 {
		t.Errorf("expected 2 versions, got %d: %v", len(result.Versions), result.Versions)
	}
	if result.Kind != "selector" {
		t.Errorf("expected Kind=selector, got %s", result.Kind)
	}
	if result.Target != ".messageBar" {
		t.Errorf("expected Target=.messageBar, got %s", result.Target)
	}
}

func TestFindOriginUnsorted(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []string{"1.0.0", "1.1.0", "1.2.0"} {
		verDir := filepath.Join(dir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		var css string
		if v == "1.0.0" {
			css = ".foo { color: red; }"
		} else {
			css = ".bar { color: blue; }"
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}

	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = orig }()

	// Pass unsorted; should still find the earliest.
	result := FindOrigin(".foo", false, []string{"1.2.0", "1.0.0", "1.1.0"})
	if !result.Found {
		t.Fatal("expected Found=true")
	}
	if result.Introduced != "1.0.0" {
		t.Errorf("expected Introduced=1.0.0, got %s", result.Introduced)
	}
}

func TestFindOriginVariable(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []string{"1.0.0", "1.1.0"} {
		verDir := filepath.Join(dir, v)
		if err := os.MkdirAll(verDir, 0755); err != nil {
			t.Fatal(err)
		}
		var css string
		if v == "1.0.0" {
			css = ":root {\n  --old: red;\n}"
		} else {
			css = ":root {\n  --old: red;\n  --new: blue;\n}"
		}
		if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(css), 0644); err != nil {
			t.Fatal(err)
		}
	}

	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = orig }()

	result := FindOrigin("--new", true, []string{"1.0.0", "1.1.0"})
	if !result.Found {
		t.Fatal("expected Found=true")
	}
	if result.Introduced != "1.1.0" {
		t.Errorf("expected Introduced=1.1.0, got %s", result.Introduced)
	}
	if result.Kind != "variable" {
		t.Errorf("expected Kind=variable, got %s", result.Kind)
	}
}

func TestFindOriginNotFound(t *testing.T) {
	dir := t.TempDir()
	verDir := filepath.Join(dir, "1.0.0")
	if err := os.MkdirAll(verDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(".foo { }"), 0644); err != nil {
		t.Fatal(err)
	}

	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = orig }()

	result := FindOrigin(".missing", false, []string{"1.0.0"})
	if result.Found {
		t.Error("expected Found=false")
	}
	if result.Introduced != "" {
		t.Errorf("expected empty Introduced, got %s", result.Introduced)
	}
	if result.Versions != nil {
		t.Errorf("expected nil Versions, got %v", result.Versions)
	}
}

func TestFindOriginVersionWithoutCSS(t *testing.T) {
	dir := t.TempDir()
	// Only 1.1.0 has CSS; 1.0.0 is missing from cache.
	verDir := filepath.Join(dir, "1.1.0")
	if err := os.MkdirAll(verDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(verDir, "app.css"), []byte(".foo { }"), 0644); err != nil {
		t.Fatal(err)
	}

	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = orig }()

	result := FindOrigin(".foo", false, []string{"1.0.0", "1.1.0"})
	if !result.Found {
		t.Fatal("expected Found=true for 1.1.0")
	}
	if result.Introduced != "1.1.0" {
		t.Errorf("expected Introduced=1.1.0, got %s", result.Introduced)
	}
}

func TestFindOriginEmptyVersions(t *testing.T) {
	result := FindOrigin(".foo", false, nil)
	if result.Found {
		t.Error("expected Found=false for empty versions")
	}
}

func TestOriginResultStringFoundSingle(t *testing.T) {
	r := &OriginResult{
		Target:     ".foo",
		Kind:       "selector",
		Introduced: "1.1.0",
		Found:      true,
		Versions:   []string{"1.1.0"},
	}
	s := r.String()
	if !strings.Contains(s, ".foo") {
		t.Errorf("expected target in output, got %s", s)
	}
	if !strings.Contains(s, "1.1.0") {
		t.Errorf("expected version in output, got %s", s)
	}
	if !strings.Contains(s, "(1 version)") {
		t.Errorf("expected '1 version' in output, got %s", s)
	}
}

func TestOriginResultStringFoundMultiple(t *testing.T) {
	r := &OriginResult{
		Target:     ".foo",
		Kind:       "selector",
		Introduced: "1.1.0",
		Found:      true,
		Versions:   []string{"1.1.0", "1.2.0", "1.3.0"},
	}
	s := r.String()
	if !strings.Contains(s, "3 versions") {
		t.Errorf("expected '3 versions' in output, got %s", s)
	}
	if !strings.Contains(s, "1.1.0, 1.2.0, 1.3.0") {
		t.Errorf("expected all versions in output, got %s", s)
	}
}

func TestOriginResultStringNotFound(t *testing.T) {
	r := &OriginResult{
		Target: ".missing",
		Kind:   "selector",
		Found:  false,
	}
	s := r.String()
	if !strings.Contains(s, "not found") {
		t.Errorf("expected 'not found' in output, got %s", s)
	}
}

func TestOriginResultMarshalJSON(t *testing.T) {
	r := &OriginResult{
		Target:     "--my-var",
		Kind:       "variable",
		Introduced: "1.2.0",
		Found:      true,
		Versions:   []string{"1.2.0", "1.3.0"},
	}
	data, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(data), `"target"`) {
		t.Errorf("expected target in JSON, got %s", string(data))
	}
	if !strings.Contains(string(data), `"found": true`) {
		t.Errorf("expected found=true in JSON, got %s", string(data))
	}
}

func TestOriginResultMarshalYAML(t *testing.T) {
	r := &OriginResult{
		Target:     ".foo",
		Kind:       "selector",
		Introduced: "1.0.0",
		Found:      true,
	}
	data, err := r.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if !strings.Contains(string(data), "target") {
		t.Errorf("expected target in YAML, got %s", string(data))
	}
}

func TestOriginResultMarshalTOMLWithVersions(t *testing.T) {
	r := &OriginResult{
		Target:     ".foo",
		Kind:       "selector",
		Introduced: "1.1.0",
		Found:      true,
		Versions:   []string{"1.1.0", "1.2.0"},
	}
	data, err := r.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "target = \".foo\"") {
		t.Errorf("expected target in TOML, got %s", s)
	}
	if !strings.Contains(s, "introduced = \"1.1.0\"") {
		t.Errorf("expected introduced in TOML, got %s", s)
	}
	if !strings.Contains(s, "found = true") {
		t.Errorf("expected found=true in TOML, got %s", s)
	}
	if !strings.Contains(s, "versions") {
		t.Errorf("expected versions in TOML, got %s", s)
	}
}

func TestOriginResultMarshalTOMLNotFound(t *testing.T) {
	r := &OriginResult{
		Target: ".missing",
		Kind:   "selector",
		Found:  false,
	}
	data, err := r.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "found = false") {
		t.Errorf("expected found=false in TOML, got %s", s)
	}
	if strings.Contains(s, "introduced") {
		t.Errorf("expected no introduced in TOML when not found, got %s", s)
	}
	if strings.Contains(s, "versions") {
		t.Errorf("expected no versions in TOML when not found, got %s", s)
	}
}

func TestOriginResultEncodeTOMLError(t *testing.T) {
	r := &OriginResult{
		Target: ".foo",
		Kind:   "selector",
		Found:  true,
	}
	err := r.encodeTOML(failingWriter{})
	if err == nil {
		t.Error("expected error from failing writer")
	}
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("write failed")
}
