package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type TLDRResult struct {
	VersionA               string           `json:"version_a" yaml:"version_a" toml:"version_a"`
	VersionB               string           `json:"version_b" yaml:"version_b" toml:"version_b"`
	VersionADate           string           `json:"version_a_date,omitempty" yaml:"version_a_date,omitempty" toml:"version_a_date,omitempty"`
	VersionBDate           string           `json:"version_b_date,omitempty" yaml:"version_b_date,omitempty" toml:"version_b_date,omitempty"`
	SemverBump             string           `json:"semver_bump" yaml:"semver_bump" toml:"semver_bump"`
	AdditionsLOC           int              `json:"additions_loc" yaml:"additions_loc" toml:"additions_loc"`
	DeletionsLOC           int              `json:"deletions_loc" yaml:"deletions_loc" toml:"deletions_loc"`
	SelectorsAdded         []string         `json:"selectors_added" yaml:"selectors_added" toml:"selectors_added"`
	SelectorsRemoved       []string         `json:"selectors_removed" yaml:"selectors_removed" toml:"selectors_removed"`
	CSSVariablesAdded      []string         `json:"css_variables_added" yaml:"css_variables_added" toml:"css_variables_added"`
	CSSVariablesRemoved    []string         `json:"css_variables_removed" yaml:"css_variables_removed" toml:"css_variables_removed"`
	CSSVariablesChanged    []VariableChange `json:"css_variables_changed" yaml:"css_variables_changed" toml:"css_variables_changed"`
	ImportantCount         int              `json:"important_count" yaml:"important_count" toml:"important_count"`
	AverageSpecificity     float64          `json:"average_specificity" yaml:"average_specificity" toml:"average_specificity"`
	TotalSelectorsAnalyzed int              `json:"total_selectors_analyzed" yaml:"total_selectors_analyzed" toml:"total_selectors_analyzed"`
	ColorCounts            map[string]int   `json:"color_counts,omitempty" yaml:"color_counts,omitempty" toml:"color_counts,omitempty"`
	Specificities          []float64        `json:"-" yaml:"-" toml:"-"`
}

type VariableChange struct {
	Name     string `json:"name" yaml:"name" toml:"name"`
	OldValue string `json:"old_value" yaml:"old_value" toml:"old_value"`
	NewValue string `json:"new_value" yaml:"new_value" toml:"new_value"`
}

var (
	cssVarRe     = regexp.MustCompile(`--[\w-]+`)
	selectorRe   = regexp.MustCompile(`^\s*([.#][\w-]+(?:\s*[+>~\s][.#][\w-]+)*)\s*\{`)
	importantRe  = regexp.MustCompile(`!important`)
	hexRe        = regexp.MustCompile(`(?i)#[0-9a-f]{3,8}`)
	rgbRe        = regexp.MustCompile(`(?i)rgba?\(`)
	hslRe        = regexp.MustCompile(`(?i)hsla?\(`)
	oklchRe      = regexp.MustCompile(`(?i)oklch\(`)
	otherColorRe = regexp.MustCompile(`(?i)(?:oklab\(|lab\(|lch\(|hwb\(|color\()`)
)

func SemverBump(a, b string) string {
	va, aok := parseVersion(a)
	vb, bok := parseVersion(b)
	if !aok || !bok {
		return "unknown"
	}
	if vb[0] > va[0] {
		return "major"
	}
	if vb[1] > va[1] {
		return "minor"
	}
	if vb[2] > va[2] {
		return "patch"
	}
	return "none"
}

func parseVersion(v string) ([]int, bool) {
	var parts []int
	for _, s := range strings.SplitN(v, ".", 3) {
		var n int
		if _, err := fmt.Sscanf(s, "%d", &n); err == nil {
			parts = append(parts, n)
		} else {
			return []int{0, 0, 0}, false
		}
	}
	for len(parts) < 3 {
		parts = append(parts, 0)
	}
	return parts[:3], true
}

func countColors(line string) map[string]int {
	counts := make(map[string]int)
	for _, m := range hexRe.FindAllString(line, -1) {
		counts["hex"]++
		_ = m
	}
	for range rgbRe.FindAllString(line, -1) {
		counts["rgb"]++
	}
	for range hslRe.FindAllString(line, -1) {
		counts["hsl"]++
	}
	for range oklchRe.FindAllString(line, -1) {
		counts["oklch"]++
	}
	for _, m := range otherColorRe.FindAllString(line, -1) {
		switch {
		case strings.HasPrefix(m, "oklab"):
			counts["oklab"]++
		case strings.HasPrefix(m, "lab("):
			counts["lab"]++
		case strings.HasPrefix(m, "lch("):
			counts["lch"]++
		case strings.HasPrefix(m, "hwb("):
			counts["hwb"]++
		case strings.HasPrefix(m, "color("):
			counts["color()"]++
		}
	}
	return counts
}

func AnalyzeCSS(css string) *TLDRResult {
	r := &TLDRResult{}
	if css == "" {
		return r
	}
	var currentSelector string
	varSelectorRe := regexp.MustCompile(`^\s*(--[\w-]+)\s*:\s*(.+?);`)
	addedVars := map[string]string{}
	for _, line := range strings.Split(css, "\n") {
		r.AdditionsLOC++
		r.analyzeLine(line, "+", &currentSelector, varSelectorRe, addedVars, nil)
	}
	for name := range addedVars {
		r.CSSVariablesAdded = append(r.CSSVariablesAdded, name)
	}
	return r
}

func AnalyzeDiff(diff string) *TLDRResult {
	r := &TLDRResult{}
	if diff == "" {
		return r
	}

	var currentSelector string
	varSelectorRe := regexp.MustCompile(`^\s*(--[\w-]+)\s*:\s*(.+?);`)
	addedVars := map[string]string{}
	removedVars := map[string]string{}

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			r.AdditionsLOC++
			content := line[1:]
			r.analyzeLine(content, "+", &currentSelector, varSelectorRe, addedVars, removedVars)
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			r.DeletionsLOC++
			content := line[1:]
			r.analyzeLine(content, "-", &currentSelector, varSelectorRe, addedVars, removedVars)
		} else if !strings.HasPrefix(line, "@@") && !strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "+++") {
			if sel := extractSelector(line); sel != "" {
				currentSelector = sel
			}
		}
	}

	for name, newVal := range addedVars {
		if oldVal, ok := removedVars[name]; ok {
			if oldVal != newVal {
				r.CSSVariablesChanged = append(r.CSSVariablesChanged, VariableChange{
					Name:     name,
					OldValue: oldVal,
					NewValue: newVal,
				})
			}
		} else {
			r.CSSVariablesAdded = append(r.CSSVariablesAdded, name)
		}
	}
	for name := range removedVars {
		if _, ok := addedVars[name]; !ok {
			r.CSSVariablesRemoved = append(r.CSSVariablesRemoved, name)
		}
	}

	return r
}

func (r *TLDRResult) analyzeLine(content, prefix string, currentSelector *string, varSelectorRe *regexp.Regexp, addedVars, removedVars map[string]string) {
	if sel := extractSelector(content); sel != "" {
		*currentSelector = sel
		r.TotalSelectorsAnalyzed++
		sp := specificity(sel)
		r.AverageSpecificity = (r.AverageSpecificity*float64(r.TotalSelectorsAnalyzed-1) + sp) / float64(r.TotalSelectorsAnalyzed)
		r.Specificities = append(r.Specificities, sp)
		if prefix == "+" {
			r.SelectorsAdded = append(r.SelectorsAdded, sel)
		} else if prefix == "-" {
			r.SelectorsRemoved = append(r.SelectorsRemoved, sel)
		}
	}

	if matches := cssVarRe.FindString(content); matches != "" {
		if isVarDefinition(content) {
			val := extractVarValue(content, varSelectorRe)
			if prefix == "+" {
				addedVars[matches] = val
			} else if prefix == "-" {
				removedVars[matches] = val
			}
		}
	}

	if importantRe.MatchString(content) {
		r.ImportantCount++
	}

	for color, count := range countColors(content) {
		if r.ColorCounts == nil {
			r.ColorCounts = make(map[string]int)
		}
		r.ColorCounts[color] += count
	}
}

func extractVarValue(line string, re *regexp.Regexp) string {
	if line == "" {
		return ""
	}
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 3 {
		return strings.TrimSpace(matches[2])
	}
	return ""
}

func isVarDefinition(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "--") && strings.Contains(trimmed, ":")
}

func extractSelector(line string) string {
	trimmed := strings.TrimSpace(line)
	matches := selectorRe.FindStringSubmatch(trimmed)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func specificity(sel string) float64 {
	var score float64
	parts := strings.FieldsFunc(sel, func(r rune) bool {
		return r == ' ' || r == '>' || r == '+' || r == '~'
	})
	for _, p := range parts {
		if strings.HasPrefix(p, "#") {
			score += 100
		} else if strings.HasPrefix(p, ".") {
			score += 10
		} else {
			score += 1
		}
	}
	return score
}

func specificityStats(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)

	sum := 0.0
	freq := make(map[float64]int)
	for _, v := range sorted {
		sum += v
		freq[v]++
	}
	mean := sum / float64(len(sorted))

	median := 0.0
	if len(sorted)%2 == 1 {
		median = sorted[len(sorted)/2]
	} else {
		mid := len(sorted) / 2
		median = (sorted[mid-1] + sorted[mid]) / 2
	}

	modeVal := sorted[0]
	maxFreq := 0
	for v, c := range freq {
		if c > maxFreq {
			maxFreq = c
			modeVal = v
		}
	}

	return fmt.Sprintf("specificity mean %.1f  median %.1f  mode %.1f", mean, median, modeVal)
}

func (r *TLDRResult) MarshalJSON() ([]byte, error) {
	type Alias TLDRResult
	return json.MarshalIndent((*Alias)(r), "", "  ")
}

func (r *TLDRResult) MarshalYAML() ([]byte, error) {
	type Alias TLDRResult
	return yaml.Marshal((*Alias)(r))
}

func (r *TLDRResult) MarshalTOML() ([]byte, error) {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	v := map[string]any{
		"version_a":                r.VersionA,
		"version_b":                r.VersionB,
		"additions_loc":            r.AdditionsLOC,
		"deletions_loc":            r.DeletionsLOC,
		"selectors_added":          r.SelectorsAdded,
		"selectors_removed":        r.SelectorsRemoved,
		"css_variables_added":      r.CSSVariablesAdded,
		"css_variables_removed":    r.CSSVariablesRemoved,
		"css_variables_changed":    r.CSSVariablesChanged,
		"important_count":          r.ImportantCount,
		"average_specificity":      r.AverageSpecificity,
		"total_selectors_analyzed": r.TotalSelectorsAnalyzed,
	}
	if r.VersionADate != "" {
		v["version_a_date"] = r.VersionADate
	}
	if r.VersionBDate != "" {
		v["version_b_date"] = r.VersionBDate
	}
	if r.SemverBump != "" {
		v["semver_bump"] = r.SemverBump
	}
	if len(r.ColorCounts) > 0 {
		v["color_counts"] = r.ColorCounts
	}
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *TLDRResult) String() string {
	figlet := []string{
		"   ____  __________ ",
		"  / __ \\/ ____/ __ \\",
		" / / / / /   / / / /",
		"/ /_/ / /___/ /_/ / ",
		"\\____/\\____/_____/  ",
	}
	artWidth := 0
	for _, l := range figlet {
		if len(l) > artWidth {
			artWidth = len(l)
		}
	}

	var right []string
	right = append(right, fmt.Sprintf("  %s -> %s  (%s)", r.VersionA, r.VersionB, r.SemverBump))
	right = append(right, fmt.Sprintf("  %d insertions(+), %d deletions(-)", r.AdditionsLOC, r.DeletionsLOC))

	var parts []string
	netDelta := r.AdditionsLOC - r.DeletionsLOC
	if netDelta >= 0 {
		parts = append(parts, fmt.Sprintf("+%d LOC", netDelta))
	} else {
		parts = append(parts, fmt.Sprintf("%d LOC", netDelta))
	}
	if n := len(r.SelectorsAdded); n > 0 {
		parts = append(parts, fmt.Sprintf("+%d selectors", n))
	}
	if n := len(r.SelectorsRemoved); n > 0 {
		parts = append(parts, fmt.Sprintf("-%d selectors", n))
	}
	if n := len(r.CSSVariablesAdded); n > 0 {
		parts = append(parts, fmt.Sprintf("+%d variables", n))
	}
	if n := len(r.CSSVariablesRemoved); n > 0 {
		parts = append(parts, fmt.Sprintf("-%d variables", n))
	}
	if n := len(r.CSSVariablesChanged); n > 0 {
		parts = append(parts, fmt.Sprintf("~%d changed", n))
	}
	if r.ImportantCount > 0 {
		parts = append(parts, fmt.Sprintf("%d !important", r.ImportantCount))
	}
	var specificityLine string
	if r.TotalSelectorsAnalyzed > 0 {
		specificityLine = specificityStats(r.Specificities)
	}
	if len(r.ColorCounts) > 0 {
		colors := make([]string, 0, len(r.ColorCounts))
		for c := range r.ColorCounts {
			colors = append(colors, c)
		}
		sort.Strings(colors)
		var colorParts []string
		for _, c := range colors {
			colorParts = append(colorParts, fmt.Sprintf("%s:%d", c, r.ColorCounts[c]))
		}
		parts = append(parts, strings.Join(colorParts, " "))
	}
	for len(parts) > 0 {
		n := 3
		if n > len(parts) {
			n = len(parts)
		}
		right = append(right, "  "+strings.Join(parts[:n], ", "))
		parts = parts[n:]
	}
	if specificityLine != "" {
		right = append(right, "  "+specificityLine)
	}

	maxLines := len(figlet)
	if len(right) > maxLines {
		maxLines = len(right)
	}
	var out strings.Builder
	for i := 0; i < maxLines; i++ {
		l := ""
		if i < len(figlet) {
			l = figlet[i]
		}
		r := ""
		if i < len(right) {
			r = right[i]
		}
		out.WriteString(l)
		if r != "" {
			if i >= len(figlet) {
				out.WriteString(strings.Repeat(" ", artWidth))
			}
			out.WriteString("  ")
			out.WriteString(r)
		}
		out.WriteString("\n")
	}
	return out.String()
}

func (r *TLDRResult) StatString() string {
	figlet := []string{
		"   ____  __________ ",
		"  / __ \\/ ____/ __ \\",
		" / / / / /   / / / /",
		"/ /_/ / /___/ /_/ / ",
		"\\____/\\____/_____/  ",
	}
	artWidth := 0
	for _, l := range figlet {
		if len(l) > artWidth {
			artWidth = len(l)
		}
	}

	var right []string
	right = append(right, fmt.Sprintf("  %s", r.VersionA))

	var parts []string
	parts = append(parts, fmt.Sprintf("%d LOC", r.AdditionsLOC))
	if n := len(r.SelectorsAdded); n > 0 {
		parts = append(parts, fmt.Sprintf("+%d selectors", n))
	}
	if n := len(r.SelectorsRemoved); n > 0 {
		parts = append(parts, fmt.Sprintf("-%d selectors", n))
	}
	if n := len(r.CSSVariablesAdded); n > 0 {
		parts = append(parts, fmt.Sprintf("+%d variables", n))
	}
	if n := len(r.CSSVariablesRemoved); n > 0 {
		parts = append(parts, fmt.Sprintf("-%d variables", n))
	}
	if n := len(r.CSSVariablesChanged); n > 0 {
		parts = append(parts, fmt.Sprintf("~%d changed", n))
	}
	if r.ImportantCount > 0 {
		parts = append(parts, fmt.Sprintf("%d !important", r.ImportantCount))
	}
	var specificityLine string
	if r.TotalSelectorsAnalyzed > 0 {
		specificityLine = specificityStats(r.Specificities)
	}
	if len(r.ColorCounts) > 0 {
		colors := make([]string, 0, len(r.ColorCounts))
		for c := range r.ColorCounts {
			colors = append(colors, c)
		}
		sort.Strings(colors)
		var colorParts []string
		for _, c := range colors {
			colorParts = append(colorParts, fmt.Sprintf("%s:%d", c, r.ColorCounts[c]))
		}
		parts = append(parts, strings.Join(colorParts, " "))
	}
	for len(parts) > 0 {
		n := 3
		if n > len(parts) {
			n = len(parts)
		}
		right = append(right, "  "+strings.Join(parts[:n], ", "))
		parts = parts[n:]
	}
	if specificityLine != "" {
		right = append(right, "  "+specificityLine)
	}

	maxLines := len(figlet)
	if len(right) > maxLines {
		maxLines = len(right)
	}
	var out strings.Builder
	for i := 0; i < maxLines; i++ {
		l := ""
		if i < len(figlet) {
			l = figlet[i]
		}
		r := ""
		if i < len(right) {
			r = right[i]
		}
		out.WriteString(l)
		if r != "" {
			if i >= len(figlet) {
				out.WriteString(strings.Repeat(" ", artWidth))
			}
			out.WriteString("  ")
			out.WriteString(r)
		}
		out.WriteString("\n")
	}
	return out.String()
}
