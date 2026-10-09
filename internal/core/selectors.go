package core

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// SelectorReport describes how a theme's selectors compare with those of a
// target Obsidian version.
type SelectorReport struct {
	TargetVersion string `json:"target_version" yaml:"target_version" toml:"target_version"`
	ThemePath     string `json:"theme_path" yaml:"theme_path" toml:"theme_path"`
	// TargetCount is how many distinct selectors the target defines.
	TargetCount int `json:"target_count" yaml:"target_count" toml:"target_count"`
	// ThemeCount is how many distinct selectors the theme defines.
	ThemeCount int `json:"theme_count" yaml:"theme_count" toml:"theme_count"`
	// Styled counts target selectors the theme matches exactly.
	Styled int `json:"styled" yaml:"styled" toml:"styled"`
	// Covered counts target selectors a theme selector reaches, including
	// the exact matches. See SelectorCovers.
	Covered int `json:"covered" yaml:"covered" toml:"covered"`
	// Uncovered counts target selectors no theme selector reaches.
	Uncovered int `json:"uncovered" yaml:"uncovered" toml:"uncovered"`
	// UncoveredList lists some of those selectors, capped at
	// SelectorListLimit entries.
	UncoveredList []string `json:"uncovered_list,omitempty" yaml:"uncovered_list,omitempty" toml:"uncovered_list,omitempty"`
	// Extra lists theme selectors that the target does not define. These are
	// usually the theme's own classes, which is normal.
	Extra []string `json:"extra,omitempty" yaml:"extra,omitempty" toml:"extra,omitempty"`
}

// SelectorListLimit caps how many entries each selector list holds.
const SelectorListLimit = 25

// splitSelector breaks a selector into compounds and the combinators between
// them. Whitespace is a descendant combinator.
func splitSelector(sel string) []string {
	var parts []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() > 0 {
			parts = append(parts, cur.String())
			cur.Reset()
		}
	}
	skipSpace := func(i int) int {
		for i < len(sel) && isSelSpace(sel[i]) {
			i++
		}
		return i
	}

	i := 0
	for i < len(sel) {
		c := sel[i]
		switch {
		case isSelSpace(c):
			flush()
			i = skipSpace(i)
			if i >= len(sel) {
				return parts
			}
			// A combinator that follows emits its own token, so no
			// descendant token is added here.
			if isSelCombinator(sel[i]) {
				continue
			}
			parts = append(parts, " ")

		case isSelCombinator(c):
			flush()
			parts = append(parts, string(c))
			i = skipSpace(i + 1)

		default:
			cur.WriteByte(c)
			i++
		}
	}
	flush()
	return parts
}

func isSelSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func isSelCombinator(c byte) bool {
	return c == '>' || c == '+' || c == '~'
}

// isCombinatorToken reports whether a split part is a combinator rather than
// a compound.
func isCombinatorToken(part string) bool {
	switch part {
	case " ", ">", "+", "~":
		return true
	default:
		return false
	}
}

// compoundBase returns the part of a compound before any pseudo-class,
// pseudo-element, or attribute selector. `.a:hover` and `.a` share the base
// `.a`, so a rule for one reaches the other.
func compoundBase(comp string) string {
	if i := strings.IndexAny(comp, ":["); i >= 0 {
		return comp[:i]
	}
	return comp
}

// compoundTokens collects the class and id names in a compound, ignoring
// anything from the first pseudo-class or attribute selector onwards.
func compoundTokens(comp string) (classes, ids map[string]bool) {
	classes = map[string]bool{}
	ids = map[string]bool{}
	comp = compoundBase(comp)

	for i := 0; i < len(comp); i++ {
		switch comp[i] {
		case '.':
			j := i + 1
			for j < len(comp) && (comp[j] == '-' || comp[j] == '_' || isAlphaNum(comp[j])) {
				j++
			}
			if j > i+1 {
				classes[comp[i+1:j]] = true
			}
			i = j - 1
		case '#':
			j := i + 1
			for j < len(comp) && (comp[j] == '-' || comp[j] == '_' || isAlphaNum(comp[j])) {
				j++
			}
			if j > i+1 {
				ids[comp[i+1:j]] = true
			}
			i = j - 1
		}
	}
	return classes, ids
}

func isAlphaNum(c byte) bool {
	return c == '-' || c == '_' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// compoundCovers reports whether a theme compound reaches a target
// compound. A theme rule for `.a` reaches `.a.b`, because both name the same
// element with the class `a`.
//
// The universal selector is never treated as coverage, because a reset in
// Obsidian's own stylesheet says nothing about what a theme does.
func compoundCovers(theme, target string) bool {
	if theme == target {
		return true
	}
	if theme == "*" || target == "*" {
		return false
	}
	tc, ti := compoundTokens(theme)
	c, d := compoundTokens(target)
	if len(tc)+len(ti) == 0 {
		// An element-only selector matches on the element name, which
		// this comparison does not model.
		return false
	}
	for name := range tc {
		if !c[name] {
			return false
		}
	}
	for name := range ti {
		if !d[name] {
			return false
		}
	}
	return true
}

// SelectorCovers reports whether a theme selector reaches a target selector.
//
// It is true when the theme selector is the target selector, or when the
// theme selector matches the leftmost part of the target selector. A theme
// rule for `.workspace-leaf` reaches
// `.workspace-leaf.mod-active .cm-content`, because the theme sets the base
// of the chain even though the target names more of it.
//
// This is a heuristic, not a cascade evaluation. It answers "does this theme
// have a say in this selector", not "which declaration wins".
func SelectorCovers(themeSel, targetSel string) bool {
	if themeSel == "" {
		return false
	}
	if themeSel == targetSel {
		return true
	}
	themeParts := splitSelector(themeSel)
	targetParts := splitSelector(targetSel)
	if len(themeParts) > len(targetParts) {
		return false
	}

	ci, ti := 0, 0
	for ci < len(themeParts) && ti < len(targetParts) {
		tp := themeParts[ci]

		if isCombinatorToken(tp) {
			if !combinatorsCompatible(tp, targetParts[ti]) {
				return false
			}
			ci++
			ti++
			continue
		}

		if isCombinatorToken(targetParts[ti]) {
			return false
		}
		if !compoundCovers(tp, targetParts[ti]) {
			return false
		}
		ci++
		ti++
	}

	// The theme ran out first, which is the case we accept: a rule for the
	// leftmost compound still applies to a target that continues deeper or
	// adds pseudo-classes.
	return ci >= len(themeParts)
}

// combinatorsCompatible reports whether a theme combinator reaches a target
// one. A descendant rule is the most general, so it reaches a child, sibling,
// or descendant target. A child rule does not reach a descendant target.
func combinatorsCompatible(theme, target string) bool {
	if !isCombinatorToken(target) {
		return false
	}
	if theme == " " {
		return true
	}
	return theme == target
}

// CompareSelectors compares the selectors of a target version with those of a
// theme. Both sides are parsed with ParseCSS, so minified and multi-line CSS
// are both read.
//
// Styled counts exact matches. Covered counts target selectors that a theme
// selector reaches, using SelectorCovers, which is the more useful number
// because Obsidian selectors are often long chains.
func CompareSelectors(targetVersion, themePath string, targetIdx, themeIdx *CSSIndex) *SelectorReport {
	rep := &SelectorReport{
		TargetVersion: targetVersion,
		ThemePath:     themePath,
	}
	if targetIdx == nil || themeIdx == nil {
		return rep
	}

	rep.TargetCount = len(targetIdx.Selectors)
	rep.ThemeCount = len(themeIdx.Selectors)

	themeSels := make([]string, 0, len(themeIdx.Selectors))
	for sel := range themeIdx.Selectors {
		themeSels = append(themeSels, sel)
	}
	sort.Strings(themeSels)

	var uncovered []string
	for sel := range targetIdx.Selectors {
		if themeIdx.Selectors[sel] {
			rep.Styled++
			rep.Covered++
			continue
		}
		covered := false
		for _, ts := range themeSels {
			if SelectorCovers(ts, sel) {
				covered = true
				break
			}
		}
		if covered {
			rep.Covered++
		} else {
			uncovered = append(uncovered, sel)
		}
	}
	sort.Strings(uncovered)
	rep.Uncovered = len(uncovered)
	rep.UncoveredList = capList(uncovered, SelectorListLimit)

	var extra []string
	for sel := range themeIdx.Selectors {
		if !targetIdx.Selectors[sel] {
			extra = append(extra, sel)
		}
	}
	sort.Strings(extra)
	rep.Extra = capList(extra, SelectorListLimit)

	return rep
}

// capList trims a list to at most limit entries.
func capList(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	return items[:limit]
}

// String renders the report for stdout.
func (r *SelectorReport) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Selector check: target v%s vs theme %s\n", r.TargetVersion, r.ThemePath)
	fmt.Fprintf(&b, "  target selectors:  %d\n", r.TargetCount)
	fmt.Fprintf(&b, "  theme selectors:    %d\n", r.ThemeCount)
	fmt.Fprintf(&b, "  exact match:        %d\n", r.Styled)
	fmt.Fprintf(&b, "  covered by theme:   %d\n", r.Covered)
	fmt.Fprintf(&b, "  not covered:        %d\n", r.Uncovered)
	if r.Uncovered > 0 {
		fmt.Fprintf(&b, "\nTarget selectors no theme selector reaches (first %d):\n", len(r.UncoveredList))
		for _, sel := range r.UncoveredList {
			b.WriteString("  - " + sel + "\n")
		}
		if r.Uncovered > len(r.UncoveredList) {
			fmt.Fprintf(&b, "  ... and %d more\n", r.Uncovered-len(r.UncoveredList))
		}
	}
	if len(r.Extra) > 0 {
		fmt.Fprintf(&b, "\nTheme selectors the target does not define (first %d):\n", len(r.Extra))
		for _, sel := range r.Extra {
			b.WriteString("  - " + sel + "\n")
		}
		if len(r.Extra) >= SelectorListLimit {
			b.WriteString("  ... list truncated\n")
		}
	}
	return b.String()
}

// MarshalTOML renders the report as TOML.
func (r *SelectorReport) MarshalTOML() ([]byte, error) {
	var buf strings.Builder
	err := r.encodeTOML(&buf)
	return []byte(buf.String()), err
}

func (r *SelectorReport) encodeTOML(w io.Writer) error {
	encoder := toml.NewEncoder(w)
	v := map[string]any{
		"target_version": r.TargetVersion,
		"theme_path":     r.ThemePath,
		"target_count":   r.TargetCount,
		"theme_count":    r.ThemeCount,
		"styled":         r.Styled,
		"covered":        r.Covered,
		"uncovered":      r.Uncovered,
	}
	if len(r.UncoveredList) > 0 {
		v["uncovered_list"] = r.UncoveredList
	}
	if len(r.Extra) > 0 {
		v["extra"] = r.Extra
	}
	return encoder.Encode(v)
}

// MarshalJSON renders the report as indented JSON.
func (r *SelectorReport) MarshalJSON() ([]byte, error) {
	type Alias SelectorReport
	return json.MarshalIndent((*Alias)(r), "", "  ")
}

// MarshalYAML renders the report as YAML.
func (r *SelectorReport) MarshalYAML() ([]byte, error) {
	type Alias SelectorReport
	return yaml.Marshal((*Alias)(r))
}
