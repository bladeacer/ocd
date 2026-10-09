package core

import (
	"os"
	"path/filepath"
	"strings"
)

// CSSIndex holds the selectors and custom properties found in one or more
// stylesheets. It is the parsed form of a theme or an Obsidian app.css.
//
// Parsing is done once per stylesheet and then looked up in constant time,
// so checking many targets against many versions stays fast.
type CSSIndex struct {
	// Variables maps a custom property name to its value.
	Variables map[string]string
	// Selectors holds the normalised text of every rule selector.
	Selectors map[string]bool
	// Files lists the source files that were parsed, in read order.
	Files []string
	// varOrder keeps the custom properties in the order they first appear.
	varOrder []string
}

// VariableList returns the custom properties in source order.
func (idx *CSSIndex) VariableList() []CSSVariable {
	out := make([]CSSVariable, 0, len(idx.varOrder))
	for _, name := range idx.varOrder {
		out = append(out, CSSVariable{Name: name, Value: idx.Variables[name]})
	}
	return out
}

// HasVariable reports whether a custom property is defined.
func (idx *CSSIndex) HasVariable(name string) bool {
	_, ok := idx.Variables[name]
	return ok
}

// HasSelector reports whether a rule with this selector exists. The selector
// is normalised before lookup, so spacing does not matter.
func (idx *CSSIndex) HasSelector(selector string) bool {
	return idx.Selectors[NormalizeSelector(selector)]
}

// ParseCSS scans a stylesheet and collects its custom properties and
// selectors.
//
// The scanner understands minified CSS. It does not rely on line breaks, so
// a rule such as `body{--a:1;--b:2}` yields both properties.
//
// A custom property is only read where a declaration can start, which is
// after a brace or a semicolon. That keeps text inside a quoted string from
// being read as a definition, even when the string spans many lines, as the
// ASCII art in some themes does.
func ParseCSS(src string) *CSSIndex {
	idx := &CSSIndex{
		Variables: make(map[string]string, 256),
		Selectors: make(map[string]bool, 256),
	}

	var sel []byte
	declStart := true

	flushSelector := func() {
		sel = sel[:0]
	}

	addSelector := func() {
		for _, s := range splitSelectorList(string(sel)) {
			s = NormalizeSelector(s)
			if s == "" || s[0] == '@' {
				continue
			}
			idx.Selectors[s] = true
		}
		flushSelector()
	}

	for i := 0; i < len(src); i++ {
		c := src[i]

		switch {
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return finishCSS(idx, sel)
			}
			i += 2 + end + 1

		case c == '"' || c == '\'':
			end := skipString(src, i)
			sel = append(sel, src[i:end]...)
			i = end - 1

		case c == '\\':
			// A backslash outside a string escapes the next byte. Themes
			// use this in attribute selectors such as [data-task=\"].
			// A quote escaped this way must not be read as a string
			// start, or the scanner loses its place for the rest of the
			// file and misses every declaration after it.
			if i+1 < len(src) {
				sel = append(sel, c, src[i+1])
				i++
			} else {
				sel = append(sel, c)
			}

		case c == '{':
			addSelector()
			declStart = true

		case c == ';' || c == '}':
			flushSelector()
			declStart = true

		case isCSSSpace(c):
			sel = append(sel, c)

		case isCSSControl(c):
			// Obsidian's app.css begins with NUL bytes. A control byte is
			// not part of any name, so drop it rather than letting it end
			// up inside a selector.

		case c == '-' && declStart && i+1 < len(src) && src[i+1] == '-':
			name, value, next, ok := readCustomProp(src, i)
			if !ok {
				declStart = false
				continue
			}
			if _, seen := idx.Variables[name]; !seen {
				idx.varOrder = append(idx.varOrder, name)
			}
			idx.Variables[name] = value
			i = next - 1

		default:
			declStart = false
			sel = append(sel, c)
		}
	}

	finishCSS(idx, sel)
	return idx
}

// finishCSS records any selector text left at the end of the input.
func finishCSS(idx *CSSIndex, sel []byte) *CSSIndex {
	for _, s := range splitSelectorList(string(sel)) {
		s = NormalizeSelector(s)
		if s == "" || s[0] == '@' {
			continue
		}
		idx.Selectors[s] = true
	}
	return idx
}

// splitSelectorList breaks a comma-separated selector list into its parts.
// A comma inside brackets, parentheses, or a quoted string does not split,
// because `[data-x="a,b"]` is one attribute value.
func splitSelectorList(s string) []string {
	var parts []string
	depth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\'':
			i = skipString(s, i) - 1
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// readCustomProp reads a custom property definition that starts at i. It
// returns the name, the value, the index just past the definition, and
// whether a definition was there at all.
func readCustomProp(src string, i int) (name, value string, next int, ok bool) {
	j := i + 2
	for j < len(src) && isCSSNameByte(src[j]) {
		j++
	}
	if j == i+2 {
		return "", "", i, false
	}
	name = src[i:j]

	for j < len(src) && isCSSSpace(src[j]) {
		j++
	}
	if j >= len(src) || src[j] != ':' {
		return "", "", i, false
	}
	j++

	start := j
	depth := 0
	for j < len(src) {
		c := src[j]
		switch c {
		case '"', '\'':
			j = skipString(src, j)
			continue
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ';', '}', '{':
			if depth == 0 {
				return name, strings.TrimSpace(src[start:j]), j, true
			}
		}
		j++
	}
	return name, strings.TrimSpace(src[start:j]), j, true
}

func isCSSSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

// isCSSControl reports whether a byte is a control character that is not
// ordinary whitespace. Obsidian ships an app.css that starts with NUL bytes.
func isCSSControl(c byte) bool {
	return c < 0x20 && !isCSSSpace(c)
}

func isCSSNameByte(c byte) bool {
	return c == '-' || c == '_' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// skipString returns the index just past the quoted string that starts at i.
// An unescaped newline ends the string, as the CSS syntax requires, and a
// backslash before a newline continues it onto the next line.
func skipString(src string, i int) int {
	quote := src[i]
	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case quote:
			return j + 1
		case '\n':
			return j
		}
	}
	return len(src)
}

// NormalizeSelector collapses runs of whitespace and trims the ends, so a
// selector written across several lines matches one written on a single line.
func NormalizeSelector(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// skipDirs are directories that never hold theme CSS worth reading.
var skipDirs = map[string]bool{
	"node_modules":    true,
	".git":            true,
	".cache":          true,
	"dist":            true,
	"build":           true,
	"vendor":          true,
	".obsidian_cache": true,
	"coverage":        true,
}

// ThemeSource describes the stylesheets read for one theme.
type ThemeSource struct {
	// Path is the file or folder the user gave.
	Path string
	// Files lists the stylesheets that were read, in read order.
	Files []string
	// Vars holds every custom property found, in source order.
	Vars []CSSVariable
	// Index is the merged parse of every stylesheet, kept so a selector
	// comparison does not have to read the files a second time.
	Index *CSSIndex
}

// Has reports whether the theme defines a custom property.
func (s *ThemeSource) Has(name string) bool {
	for _, v := range s.Vars {
		if v.Name == name {
			return true
		}
	}
	return false
}

// ReadTheme collects the custom properties of a theme. The path may be a
// single stylesheet or a folder.
//
// For a folder, every `*.css` file below it is read, with `theme.css` first
// because that is the file Obsidian loads. Noise directories such as
// `node_modules` are skipped.
//
// Only custom properties (`--name`) are collected. Sass variables (`$name`)
// are skipped, because Obsidian never sees them.
func ReadTheme(path string) (*ThemeSource, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	src := &ThemeSource{Path: path}
	if !info.IsDir() {
		if err := readThemeFile(src, path); err != nil {
			return nil, err
		}
		return src, nil
	}

	files, err := collectCSSFiles(path)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if err := readThemeFile(src, f); err != nil {
			return nil, err
		}
	}
	return src, nil
}

// readThemeFile parses one stylesheet and appends its new variables.
func readThemeFile(src *ThemeSource, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	idx := ParseCSS(string(data))
	src.Files = append(src.Files, path)
	if src.Index == nil {
		src.Index = idx
	} else {
		src.Index.merge(idx)
	}

	seen := make(map[string]bool, len(src.Vars))
	for _, v := range src.Vars {
		seen[v.Name] = true
	}
	for _, v := range idx.VariableList() {
		if seen[v.Name] {
			continue
		}
		seen[v.Name] = true
		src.Vars = append(src.Vars, v)
	}
	return nil
}

// merge folds another parse of a stylesheet into this one.
func (idx *CSSIndex) merge(other *CSSIndex) {
	if other == nil {
		return
	}
	for name, value := range other.Variables {
		if _, seen := idx.Variables[name]; !seen {
			idx.varOrder = append(idx.varOrder, name)
		}
		idx.Variables[name] = value
	}
	for sel := range other.Selectors {
		idx.Selectors[sel] = true
	}
}

// collectCSSFiles walks root and returns every CSS file below it, sorted,
// with noise directories skipped. A theme.css at the root comes first,
// because it is the file Obsidian loads.
func collectCSSFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".css") {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sortThemeFiles(files)
	return files, nil
}

// sortThemeFiles puts the entry stylesheet first, then sorts by path so the
// result is stable across runs.
func sortThemeFiles(files []string) {
	for i := 1; i < len(files); i++ {
		for j := i; j > 0; j-- {
			if themeFileLess(files[j], files[j-1]) {
				files[j], files[j-1] = files[j-1], files[j]
				continue
			}
			break
		}
	}
}

// themeFileLess orders files so theme.css comes first, then lexical order.
func themeFileLess(a, b string) bool {
	aMain := filepath.Base(a) == "theme.css"
	bMain := filepath.Base(b) == "theme.css"
	if aMain != bMain {
		return aMain
	}
	return a < b
}
