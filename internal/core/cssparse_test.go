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

func TestReadThemeFolderSkipsNoiseDirs(t *testing.T) {
	root := writeTree(t, map[string]string{
		"theme.css":              `body{--a:1}`,
		"node_modules/dep/x.css": `body{--b:2}`,
		".git/o.css":             `body{--c:3}`,
		"dist/build.css":         `body{--d:4}`,
		"sub/keep.css":           `body{--e:5}`,
		"sub/notes.md":           "not css",
		"sub/style.CSS":          `body{--f:6}`,
	})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if len(src.Files) != 3 {
		t.Fatalf("read %d files, want 3: %v", len(src.Files), src.Files)
	}
	for _, skipped := range []string{"--b", "--c", "--d"} {
		if src.Has(skipped) {
			t.Errorf("%s came from a directory that should be skipped", skipped)
		}
	}
	for _, want := range []string{"--a", "--e", "--f"} {
		if !src.Has(want) {
			t.Errorf("%s should have been read", want)
		}
	}
}

func TestReadThemeFolderWithNoCSS(t *testing.T) {
	root := writeTree(t, map[string]string{"README.md": "nothing", "src/main.scss": "$x:1;.y{--a:1}"})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if len(src.Files) != 0 {
		t.Errorf("SCSS is no longer read, got %v", src.Files)
	}
	if len(src.Vars) != 0 {
		t.Errorf("expected no variables, got %v", names(src.Vars))
	}
}

func TestReadThemeKeepsSelectorIndex(t *testing.T) {
	root := writeTree(t, map[string]string{
		"theme.css":     `body{--a:1}.one{}`,
		"css/extra.css": `.two{}`,
	})

	src, err := ReadTheme(root)
	if err != nil {
		t.Fatalf("ReadTheme: %v", err)
	}
	if src.Index == nil {
		t.Fatal("Index must be kept for selector comparison")
	}
	if !src.Index.HasSelector(".one") || !src.Index.HasSelector(".two") {
		t.Errorf("Index missing selectors: %v", src.Index.Selectors)
	}
}

func TestParseCSSIgnoresControlBytes(t *testing.T) {
	// Obsidian's app.css starts with NUL bytes before its licence comment.
	idx := ParseCSS("\x00\x00/*\r\n * comment\r\n */\r\n:root{--a:1}")

	if !idx.HasVariable("--a") {
		t.Errorf("variable lost after control bytes: %v", idx.Variables)
	}
	if !idx.HasSelector(":root") {
		t.Errorf("selector lost after control bytes: %v", idx.Selectors)
	}
	for sel := range idx.Selectors {
		for i := 0; i < len(sel); i++ {
			if sel[i] < 0x20 && sel[i] != ' ' {
				t.Errorf("selector %q kept a control byte", sel)
			}
		}
	}
}

func TestParseCSSSelectorLists(t *testing.T) {
	idx := ParseCSS(`.a, .b {color:red}
.c[data-x="p,q"], .d:hover{color:blue}`)

	for _, sel := range []string{".a", ".b", `.c[data-x="p,q"]`, ".d:hover"} {
		if !idx.HasSelector(sel) {
			t.Errorf("selector %q not found; have %v", sel, idx.Selectors)
		}
	}
	if idx.HasSelector(".a, .b") {
		t.Error("a comma list must not be kept as one selector")
	}
}

func TestParseCSSAtRuleNotASelector(t *testing.T) {
	idx := ParseCSS(`@media (min-width: 100px){.inside{color:red}}@supports (display:grid){.grid{}}`)

	if idx.HasSelector("@media (min-width: 100px)") {
		t.Error("at-rules must not be recorded as selectors")
	}
	if !idx.HasSelector(".inside") || !idx.HasSelector(".grid") {
		t.Errorf("rules inside at-rules must be found: %v", idx.Selectors)
	}
}
