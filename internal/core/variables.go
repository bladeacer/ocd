package core

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// cssVarNameRe matches a CSS custom property definition: `--name: value;`.
var cssVarNameRe = regexp.MustCompile(`^\s*(--[\w-]+)\s*:`)

// CSSVariable holds the name and value of a single CSS custom property.
type CSSVariable struct {
	Name  string `json:"name" yaml:"name" toml:"name"`
	Value string `json:"value" yaml:"value" toml:"value"`
}

// VariableReport describes the result of comparing a theme's CSS variables
// against a target Obsidian version's variables.
type VariableReport struct {
	TargetVersion string        `json:"target_version" yaml:"target_version" toml:"target_version"`
	ThemePath     string        `json:"theme_path" yaml:"theme_path" toml:"theme_path"`
	TargetVars    []CSSVariable `json:"target_variables" yaml:"target_variables" toml:"target_variables"`
	ThemeVars     []CSSVariable `json:"theme_variables" yaml:"theme_variables" toml:"theme_variables"`
	Missing       []string      `json:"missing" yaml:"missing" toml:"missing"`
	Extra         []string      `json:"extra" yaml:"extra" toml:"extra"`
}

// ExtractCSSVariables parses CSS source and returns the custom properties
// defined in it, in source order.
func ExtractCSSVariables(css string) []CSSVariable {
	var out []CSSVariable
	seen := make(map[string]bool)
	for _, line := range strings.Split(css, "\n") {
		m := cssVarNameRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		val := extractVarValue(line, varSelectorRe)
		out = append(out, CSSVariable{Name: name, Value: val})
	}
	return out
}

// VariableNames returns the names of the variables in order.
func VariableNames(vars []CSSVariable) []string {
	out := make([]string, len(vars))
	for i, v := range vars {
		out[i] = v.Name
	}
	return out
}

// CompareVariables builds a VariableReport comparing the variables defined in
// a target version against those defined in a theme file. A variable is
// "missing" when it is present in the target but absent from the theme.
func CompareVariables(targetVersion, themePath string, targetVars, themeVars []CSSVariable) *VariableReport {
	targetNames := make(map[string]string, len(targetVars))
	for _, v := range targetVars {
		targetNames[v.Name] = v.Value
	}
	themeNames := make(map[string]bool, len(themeVars))
	for _, v := range themeVars {
		themeNames[v.Name] = true
	}

	var missing []string
	for name := range targetNames {
		if !themeNames[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	var extra []string
	for name := range themeNames {
		if _, ok := targetNames[name]; !ok {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)

	return &VariableReport{
		TargetVersion: targetVersion,
		ThemePath:     themePath,
		TargetVars:    targetVars,
		ThemeVars:     themeVars,
		Missing:       missing,
		Extra:         extra,
	}
}

// VariableReportString renders the report for stdout.
func (r *VariableReport) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Variable check: target v%s vs theme %s\n", r.TargetVersion, r.ThemePath))
	b.WriteString(fmt.Sprintf("  target variables: %d\n", len(r.TargetVars)))
	b.WriteString(fmt.Sprintf("  theme variables:   %d\n", len(r.ThemeVars)))
	b.WriteString(fmt.Sprintf("  missing:           %d\n", len(r.Missing)))
	b.WriteString(fmt.Sprintf("  extra:             %d\n", len(r.Extra)))
	if len(r.Missing) > 0 {
		b.WriteString("\nMissing from theme (present in target):\n")
		for _, name := range r.Missing {
			b.WriteString("  - " + name + "\n")
		}
	}
	if len(r.Extra) > 0 {
		b.WriteString("\nExtra in theme (not in target):\n")
		for _, name := range r.Extra {
			b.WriteString("  - " + name + "\n")
		}
	}
	if len(r.Missing) == 0 && len(r.Extra) == 0 {
		b.WriteString("\nTheme covers all target variables.\n")
	}
	return b.String()
}

// MarshalTOML renders the report as TOML.
func (r *VariableReport) MarshalTOML() ([]byte, error) {
	var buf strings.Builder
	encoder := toml.NewEncoder(&buf)
	v := map[string]any{
		"target_version": r.TargetVersion,
		"theme_path":     r.ThemePath,
		"missing_count":  len(r.Missing),
		"extra_count":    len(r.Extra),
		"target_counts": map[string]int{
			"target_variables": len(r.TargetVars),
			"theme_variables":  len(r.ThemeVars),
		},
	}
	if len(r.TargetVars) > 0 {
		var arr []map[string]string
		for _, vv := range r.TargetVars {
			arr = append(arr, map[string]string{"name": vv.Name, "value": vv.Value})
		}
		v["target_variables"] = arr
	}
	if len(r.ThemeVars) > 0 {
		var arr []map[string]string
		for _, vv := range r.ThemeVars {
			arr = append(arr, map[string]string{"name": vv.Name, "value": vv.Value})
		}
		v["theme_variables"] = arr
	}
	if len(r.Missing) > 0 {
		v["missing"] = r.Missing
	}
	if len(r.Extra) > 0 {
		v["extra"] = r.Extra
	}
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// MarshalJSON renders the report as indented JSON.
func (r *VariableReport) MarshalJSON() ([]byte, error) {
	type alias VariableReport
	return json.MarshalIndent((*alias)(r), "", "  ")
}

// MarshalYAML renders the report as YAML.
func (r *VariableReport) MarshalYAML() ([]byte, error) {
	type alias VariableReport
	return yaml.Marshal((*alias)(r))
}

// ReadCSSVariables reads a CSS file from disk and returns its variables.
func ReadCSSVariables(path string) ([]CSSVariable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read theme %q: %w", path, err)
	}
	return ExtractCSSVariables(string(data)), nil
}
