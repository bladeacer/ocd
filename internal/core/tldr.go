package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type TLDRResult struct {
	VersionA               string            `json:"version_a" yaml:"version_a" toml:"version_a"`
	VersionB               string            `json:"version_b" yaml:"version_b" toml:"version_b"`
	AdditionsLOC           int               `json:"additions_loc" yaml:"additions_loc" toml:"additions_loc"`
	DeletionsLOC           int               `json:"deletions_loc" yaml:"deletions_loc" toml:"deletions_loc"`
	SelectorsAdded         []string          `json:"selectors_added" yaml:"selectors_added" toml:"selectors_added"`
	SelectorsRemoved       []string          `json:"selectors_removed" yaml:"selectors_removed" toml:"selectors_removed"`
	CSSVariablesAdded      []string          `json:"css_variables_added" yaml:"css_variables_added" toml:"css_variables_added"`
	CSSVariablesRemoved    []string          `json:"css_variables_removed" yaml:"css_variables_removed" toml:"css_variables_removed"`
	CSSVariablesChanged    []VariableChange  `json:"css_variables_changed" yaml:"css_variables_changed" toml:"css_variables_changed"`
	ImportantCount         int               `json:"important_count" yaml:"important_count" toml:"important_count"`
	AverageSpecificity     float64           `json:"average_specificity" yaml:"average_specificity" toml:"average_specificity"`
	TotalSelectorsAnalyzed int               `json:"total_selectors_analyzed" yaml:"total_selectors_analyzed" toml:"total_selectors_analyzed"`
}

type VariableChange struct {
	Name    string `json:"name" yaml:"name" toml:"name"`
	OldValue string `json:"old_value" yaml:"old_value" toml:"old_value"`
	NewValue string `json:"new_value" yaml:"new_value" toml:"new_value"`
}

var (
	cssVarRe      = regexp.MustCompile(`--[\w-]+`)
	selectorRe    = regexp.MustCompile(`^\s*([.#][\w-]+(?:\s*[+>~\s][.#][\w-]+)*)\s*\{`)
	specificityRe = regexp.MustCompile(`^(\s*)([.#][\w-]+)`)
	importantRe   = regexp.MustCompile(`!important`)
)

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
			r.CSSVariablesChanged = append(r.CSSVariablesChanged, VariableChange{
				Name:     name,
				OldValue: oldVal,
				NewValue: newVal,
			})
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
	type Alias TLDRResult
	if err := encoder.Encode((*Alias)(r)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *TLDRResult) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("TLDR Diff: %s -> %s\n", r.VersionA, r.VersionB))
	b.WriteString(fmt.Sprintf("  Additions: %d LOC\n", r.AdditionsLOC))
	b.WriteString(fmt.Sprintf("  Deletions: %d LOC\n", r.DeletionsLOC))
	b.WriteString(fmt.Sprintf("  Selectors added: %d\n", len(r.SelectorsAdded)))
	b.WriteString(fmt.Sprintf("  Selectors removed: %d\n", len(r.SelectorsRemoved)))
	b.WriteString(fmt.Sprintf("  CSS variables added: %d\n", len(r.CSSVariablesAdded)))
	b.WriteString(fmt.Sprintf("  CSS variables removed: %d\n", len(r.CSSVariablesRemoved)))
	b.WriteString(fmt.Sprintf("  CSS variables changed: %d\n", len(r.CSSVariablesChanged)))
	b.WriteString(fmt.Sprintf("  !important count: %d\n", r.ImportantCount))
	b.WriteString(fmt.Sprintf("  Average specificity: %.1f\n", r.AverageSpecificity))
	if len(r.SelectorsAdded) > 0 {
		b.WriteString(fmt.Sprintf("  Selectors added: %v\n", r.SelectorsAdded))
	}
	if len(r.SelectorsRemoved) > 0 {
		b.WriteString(fmt.Sprintf("  Selectors removed: %v\n", r.SelectorsRemoved))
	}
	if len(r.CSSVariablesChanged) > 0 {
		b.WriteString("  CSS variable changes:\n")
		for _, vc := range r.CSSVariablesChanged {
			b.WriteString(fmt.Sprintf("    %s: %q -> %q\n", vc.Name, vc.OldValue, vc.NewValue))
		}
	}
	return b.String()
}
