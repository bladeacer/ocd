package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func names(vars []CSSVariable) []string {
	out := make([]string, len(vars))
	for i, v := range vars {
		out[i] = v.Name
	}
	return out
}

func TestParseCSSMinified(t *testing.T) {
	idx := ParseCSS(`body{--a:1;--b:2}.x{--c:3}`)

	if len(idx.Variables) != 3 {
		t.Fatalf("got %d variables, want 3: %v", len(idx.Variables), idx.Variables)
	}
	for name, want := range map[string]string{"--a": "1", "--b": "2", "--c": "3"} {
		if got := idx.Variables[name]; got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestParseCSSOrderAndValues(t *testing.T) {
	vars := ExtractCSSVariables(`body{--first: 1px;--second: red;--first: 9px}`)

	if got := strings.Join(names(vars), ","); got != "--first,--second" {
		t.Errorf("order = %s, want --first,--second", got)
	}
	if vars[0].Value != "9px" {
		t.Errorf("last definition should win, got %q", vars[0].Value)
	}
}

func TestParseCSSValueWithColonAndSemicolon(t *testing.T) {
	idx := ParseCSS(`a{--url: url(http://x/y;z);--next: 1}`)

	if idx.Variables["--url"] != "url(http://x/y;z)" {
		t.Errorf("--url = %q, want the full url", idx.Variables["--url"])
	}
	if idx.Variables["--next"] != "1" {
		t.Errorf("--next = %q, want 1", idx.Variables["--next"])
	}
}

func TestParseCSSValueWithSemicolonInsideString(t *testing.T) {
	idx := ParseCSS(`a{--a: "x;y";--b: 2}`)

	if idx.Variables["--a"] != `"x;y"` {
		t.Errorf("--a = %q, want \"x;y\"", idx.Variables["--a"])
	}
	if !idx.HasVariable("--b") {
		t.Error("a semicolon inside a string must not end the declaration")
	}
}

func TestParseCSSSkipsComments(t *testing.T) {
	idx := ParseCSS(`body{--real: 1}/* --fake: 2; .ghost {} */.y{--also: 3}`)

	if idx.HasVariable("--fake") {
		t.Error("a variable inside a comment must be ignored")
	}
	if idx.HasSelector(".ghost") {
		t.Error("a selector inside a comment must be ignored")
	}
	if !idx.HasVariable("--real") || !idx.HasVariable("--also") {
		t.Error("parsing must continue after a comment")
	}
}

func TestParseCSSUnterminatedComment(t *testing.T) {
	idx := ParseCSS(`body{--a: 1}/* never closed`)

	if !idx.HasVariable("--a") {
		t.Error("declarations before an unterminated comment must still be found")
	}
}

// A quote escaped outside a string, as it appears in attribute selectors,
// must not open a string. Regression test: getting this wrong swallowed the
// rest of the file and hid every declaration after it.
func TestParseCSSEscapedQuoteInSelector(t *testing.T) {
	src := `input[data-task=\"]:checked{color:red}` +
		`body.theme-dark{--interactive-accent: oklch(from var(--x) l c h / 80%);--interactive-accent-hsl: var(--interactive-accent)}`

	idx := ParseCSS(src)

	if !idx.HasVariable("--interactive-accent-hsl") {
		t.Errorf("declaration after an escaped quote was lost: %v", idx.Variables)
	}
	if !idx.HasVariable("--interactive-accent") {
		t.Errorf("first declaration lost: %v", idx.Variables)
	}
}

// A multi-line string, as used for ASCII art in some themes, must not hide
// the declarations that follow it.
func TestParseCSSMultilineString(t *testing.T) {
	src := "body{\n  --art: \" \\\\a\\\n   art\";\n  --after: 2;\n}"

	idx := ParseCSS(src)

	if !idx.HasVariable("--after") {
		t.Errorf("declaration after a multi-line string was lost: %v", idx.Variables)
	}
}

func TestParseCSSSingleQuotedValueWithDoubleQuote(t *testing.T) {
	idx := ParseCSS(`a{content:'"';--next: 1}`)

	if !idx.HasVariable("--next") {
		t.Errorf("declaration after a quoted quote was lost: %v", idx.Variables)
	}
}

func TestParseCSSSelectors(t *testing.T) {
	idx := ParseCSS(`.messageBar{color:red}
body .leaf[data-x="a b"] .gutter{color:blue}
@media screen{.in-media{color:green}}`)

	for _, sel := range []string{".messageBar", `body .leaf[data-x="a b"] .gutter`, ".in-media"} {
		if !idx.HasSelector(sel) {
			t.Errorf("selector %q not found; have %v", sel, idx.Selectors)
		}
	}
	if idx.HasSelector("@media screen") {
		t.Error("at-rules are not selectors")
	}
}

func TestParseCSSSelectorSpanningLines(t *testing.T) {
	idx := ParseCSS(".foo\n.bar {\n  color: red;\n}")

	if !idx.HasSelector(".foo .bar") {
		t.Errorf("multi-line selector not found; have %v", idx.Selectors)
	}
	if !idx.HasSelector(".foo\n.bar") {
		t.Error("lookup should normalise whitespace too")
	}
}

func TestNormalizeSelector(t *testing.T) {
	cases := map[string]string{
		"  .a   .b  ": ".a .b",
		".a\n.b":      ".a .b",
		".a":          ".a",
		"":            "",
		"\t\n":        "",
	}
	for in, want := range cases {
		if got := NormalizeSelector(in); got != want {
			t.Errorf("NormalizeSelector(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseCSSEdgeCases(t *testing.T) {
	if idx := ParseCSS(""); len(idx.Variables) != 0 || len(idx.Selectors) != 0 {
		t.Error("empty input must give an empty index")
	}
	if idx := ParseCSS("a{--:1}"); idx.HasVariable("--") {
		t.Error("a bare -- is not a custom property")
	}
	if idx := ParseCSS("a{--x}"); idx.HasVariable("--x") {
		t.Error("a custom property without a colon is not a definition")
	}
	if idx := ParseCSS("a{--x:1}"); !idx.HasVariable("--x") {
		t.Error("a final declaration without a semicolon must be found")
	}
}

func TestContainsSelectorAndVariable(t *testing.T) {
	css := `body{--a:1}.messageBar{color:red}/* --ghost: 1 */`

	if !ContainsVariable(css, "--a") {
		t.Error("ContainsVariable should find --a")
	}
	if ContainsVariable(css, "--ghost") {
		t.Error("ContainsVariable must ignore comments")
	}
	if ContainsVariable(css, "") {
		t.Error("an empty name never matches")
	}
	if !ContainsSelector(css, ".messageBar") {
		t.Error("ContainsSelector should find .messageBar")
	}
	if ContainsSelector(css, "") {
		t.Error("an empty selector never matches")
	}
}

// writeTree creates a directory tree from a map of relative path to content.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}
	return root
}

func TestReadThemeSingleFile(t *testing.T) {
	root := writeTree(t, map[string]string{"theme.css": `body{--a:1;--b:2}`})
	path := filepath.Join(root, "theme.css")

	src, err := ReadTheme(path)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if len(src.Vars) != 2 || len(src.Files) != 1 {
		t.Errorf("got %d vars from %d files, want 2 from 1", len(src.Vars), len(src.Files))
	}
}

func TestReadThemeFolderPrefersCSS(t *testing.T) {
	root := writeTree(t, map[string]string{
		"theme.css":        `body{--from-css:1;--shared:css}`,
		"css/extra.css":    `.x{--from-extra:2}`,
		"scss/main.scss":   `$sass-var: 3; .y{--from-scss:4}`,
		"node_modules/p/x": `body{--from-node:5}`,
	})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if src.Kind != "css" {
		t.Errorf("Kind = %q, want css", src.Kind)
	}
	if src.Has("--from-scss") {
		t.Error("SCSS must not be read when the folder has built CSS")
	}
	if src.Has("--from-node") {
		t.Error("node_modules must be skipped")
	}
	if !src.Has("--from-css") || !src.Has("--from-extra") {
		t.Errorf("missing CSS variables: %v", names(src.Vars))
	}
	if len(src.Vars) != 3 {
		t.Errorf("got %d vars, want 3: %v", len(src.Vars), names(src.Vars))
	}
}

func TestReadThemeFolderFallsBackToSCSS(t *testing.T) {
	root := writeTree(t, map[string]string{
		"scss/main.scss":       `$sass-only: 1;\n.a{--from-scss:2}`,
		"scss/partial/_b.scss": `.b{--from-partial:3}`,
		"README.md":            "not css",
	})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if src.Kind != "scss" {
		t.Errorf("Kind = %q, want scss", src.Kind)
	}
	if !src.Has("--from-scss") || !src.Has("--from-partial") {
		t.Errorf("missing SCSS variables: %v", names(src.Vars))
	}
	if src.Has("--sass-only") {
		t.Error("Sass variables are not custom properties and must be skipped")
	}
}

func TestReadThemeFolderPrefersThemeCSSFirst(t *testing.T) {
	root := writeTree(t, map[string]string{
		"theme.css":   `body{--a:1}`,
		"a-first.css": `.a{--b:2}`,
		"z-last.css":  `.z{--c:3}`,
	})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if filepath.Base(src.Files[0]) != "theme.css" {
		t.Errorf("first file = %q, want theme.css", src.Files[0])
	}
	if got := names(src.Vars); got[0] != "--a" {
		t.Errorf("theme.css variables should come first, got %v", got)
	}
}

func TestReadThemeFolderWithNothingToRead(t *testing.T) {
	root := writeTree(t, map[string]string{"README.md": "hello"})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if src.Kind != "" || len(src.Files) != 0 {
		t.Errorf("expected no sources, got kind=%q files=%v", src.Kind, src.Files)
	}
}

func TestReadThemeMissingPath(t *testing.T) {
	if _, err := ReadTheme(filepath.Join(t.TempDir(), "nope.css")); err == nil {
		t.Error("expected an error for a missing path")
	}
}

func TestReadThemeDedupesAcrossFiles(t *testing.T) {
	root := writeTree(t, map[string]string{
		"theme.css":     `body{--shared:1;--only-here:2}`,
		"css/extra.css": `.x{--shared:99;--only-there:3}`,
	})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	count := 0
	for _, v := range src.Vars {
		if v.Name == "--shared" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("--shared appears %d times, want 1", count)
	}
	if len(src.Vars) != 3 {
		t.Errorf("got %d vars, want 3: %v", len(src.Vars), names(src.Vars))
	}
}
