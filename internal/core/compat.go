package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type CompatMode string

const (
	CompatModeStrict  CompatMode = "strict"
	CompatModeRelaxed CompatMode = "relaxed"
)

type TargetOrigin struct {
	Name       string   `json:"name" yaml:"name" toml:"name"`
	Kind       string   `json:"kind" yaml:"kind" toml:"kind"`
	Introduced string   `json:"introduced,omitempty" yaml:"introduced,omitempty" toml:"introduced,omitempty"`
	Found      bool     `json:"found" yaml:"found" toml:"found"`
	Versions   []string `json:"versions,omitempty" yaml:"versions,omitempty" toml:"versions,omitempty"`
}

type CompatCheckResult struct {
	TargetVersion string         `json:"target_version" yaml:"target_version" toml:"target_version"`
	ThemePath     string         `json:"theme_path" yaml:"theme_path" toml:"theme_path"`
	Mode          string         `json:"mode" yaml:"mode" toml:"mode"`
	Compatible    bool           `json:"compatible" yaml:"compatible" toml:"compatible"`
	Violations    []TargetOrigin `json:"violations,omitempty" yaml:"violations,omitempty" toml:"violations,omitempty"`
	Warnings      []TargetOrigin `json:"warnings,omitempty" yaml:"warnings,omitempty" toml:"warnings,omitempty"`
	Unknown       []TargetOrigin `json:"unknown,omitempty" yaml:"unknown,omitempty" toml:"unknown,omitempty"`
	AllChecked    []TargetOrigin `json:"all_checked,omitempty" yaml:"all_checked,omitempty" toml:"all_checked,omitempty"`
}

// FindOrigins reports, for every target, the versions where it appears.
//
// Each version is read and parsed once, in parallel, and the result is a
// lookup per target. Parsing the file once per target instead would make a
// theme with hundreds of variables quadratic in the number of variables.
func FindOrigins(targets []string, isVariables bool, versions []string) []TargetOrigin {
	kind := "selector"
	if isVariables {
		kind = "variable"
	}

	results := make([]TargetOrigin, len(targets))
	for i, target := range targets {
		results[i] = TargetOrigin{
			Name:  target,
			Kind:  kind,
			Found: false,
		}
	}
	if len(targets) == 0 || len(versions) == 0 {
		return results
	}

	sorted := SortVersions(versions)

	// presence[i][j] reports whether target j is defined in version i.
	presence := make([][]bool, len(sorted))
	ParallelFor(len(sorted), DefaultSweepConcurrency, func(i int) {
		row := scanTargets(sorted[i], targets, isVariables)
		presence[i] = row
	})

	for i, v := range sorted {
		row := presence[i]
		if row == nil {
			continue
		}
		for j, present := range row {
			if !present {
				continue
			}
			if !results[j].Found {
				results[j].Found = true
				results[j].Introduced = v
			}
			results[j].Versions = append(results[j].Versions, v)
		}
	}
	return results
}

// scanTargets parses one version and reports which targets it defines. It
// returns nil when the version has no cached CSS.
func scanTargets(version string, targets []string, isVariables bool) []bool {
	data, err := os.ReadFile(filepath.Join(CSSDir, version, "app.css"))
	if err != nil {
		return nil
	}
	idx := ParseCSS(string(data))

	row := make([]bool, len(targets))
	for i, t := range targets {
		if isVariables {
			row[i] = idx.HasVariable(t)
		} else {
			row[i] = idx.HasSelector(t)
		}
	}
	return row
}

func CheckCompatibility(themeTargets []string, isVariables bool, targetVersion string, mode CompatMode, versions []string) *CompatCheckResult {
	origins := FindOrigins(themeTargets, isVariables, versions)

	result := &CompatCheckResult{
		TargetVersion: targetVersion,
		Mode:          string(mode),
		Compatible:    true,
		AllChecked:    origins,
	}

	targetParts, targetOK := parseVersion(targetVersion)

	for _, origin := range origins {
		if !origin.Found {
			result.Unknown = append(result.Unknown, origin)
			if mode == CompatModeStrict {
				result.Compatible = false
			}
			continue
		}

		originParts, originOK := parseVersion(origin.Introduced)
		if !targetOK || !originOK {
			continue
		}

		originIsNewer := false
		for i := 0; i < 3; i++ {
			if originParts[i] > targetParts[i] {
				originIsNewer = true
				break
			}
			if originParts[i] < targetParts[i] {
				break
			}
		}

		if originIsNewer {
			switch mode {
			case CompatModeStrict:
				result.Compatible = false
				result.Violations = append(result.Violations, origin)
			default:
				result.Warnings = append(result.Warnings, origin)
			}
		}
	}

	if len(result.Violations) > 0 {
		result.Compatible = false
	}

	return result
}

func (r *CompatCheckResult) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Compatibility check: target v%s, mode %s\n", r.TargetVersion, r.Mode)
	if r.Compatible {
		fmt.Fprintf(&b, "  result:           compatible\n")
	} else {
		fmt.Fprintf(&b, "  result:           incompatible\n")
	}
	fmt.Fprintf(&b, "  total checked:    %d\n", len(r.AllChecked))
	if len(r.Violations) > 0 {
		fmt.Fprintf(&b, "  violations:      %d\n", len(r.Violations))
		b.WriteString("\nViolations (introduced after target version):\n")
		for _, v := range r.Violations {
			fmt.Fprintf(&b, "  - %s (introduced in %s)\n", v.Name, v.Introduced)
		}
	}
	if len(r.Warnings) > 0 {
		fmt.Fprintf(&b, "  warnings:        %d\n", len(r.Warnings))
		b.WriteString("\nWarnings (introduced after target version):\n")
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "  - %s (introduced in %s)\n", w.Name, w.Introduced)
		}
	}
	if len(r.Unknown) > 0 {
		fmt.Fprintf(&b, "  unknown origin:  %d\n", len(r.Unknown))
		b.WriteString("\nUnknown origin (not found in any cached version):\n")
		for _, u := range r.Unknown {
			fmt.Fprintf(&b, "  - %s\n", u.Name)
		}
	}
	if len(r.Violations) == 0 && len(r.Warnings) == 0 && len(r.Unknown) == 0 {
		b.WriteString("\nAll targets were introduced at or before the target version.\n")
	}
	b.WriteString("\nNote: the first version where a variable or selector is introduced")
	b.WriteString("\ndoes not imply it was used significantly or extensively enough")
	b.WriteString("\nin that version. Usage depends on the CSS the development team writes.\n")
	return b.String()
}

func (r *CompatCheckResult) MarshalJSON() ([]byte, error) {
	type alias CompatCheckResult
	return json.MarshalIndent((*alias)(r), "", "  ")
}

func (r *CompatCheckResult) MarshalYAML() ([]byte, error) {
	type alias CompatCheckResult
	return yaml.Marshal((*alias)(r))
}

func (r *CompatCheckResult) MarshalTOML() ([]byte, error) {
	var buf strings.Builder
	err := r.encodeTOML(&buf)
	return []byte(buf.String()), err
}

func (r *CompatCheckResult) encodeTOML(w io.Writer) error {
	encoder := toml.NewEncoder(w)
	v := map[string]any{
		"target_version": r.TargetVersion,
		"mode":           r.Mode,
		"compatible":     r.Compatible,
		"counts": map[string]int{
			"total_checked": len(r.AllChecked),
			"violations":    len(r.Violations),
			"warnings":      len(r.Warnings),
			"unknown":       len(r.Unknown),
		},
	}
	if len(r.Violations) > 0 {
		arr := make([]map[string]any, len(r.Violations))
		for i, vv := range r.Violations {
			arr[i] = targetOriginMap(vv)
		}
		v["violations"] = arr
	}
	if len(r.Warnings) > 0 {
		arr := make([]map[string]any, len(r.Warnings))
		for i, vv := range r.Warnings {
			arr[i] = targetOriginMap(vv)
		}
		v["warnings"] = arr
	}
	if len(r.Unknown) > 0 {
		arr := make([]map[string]any, len(r.Unknown))
		for i, vv := range r.Unknown {
			arr[i] = targetOriginMap(vv)
		}
		v["unknown"] = arr
	}
	return encoder.Encode(v)
}

func targetOriginMap(t TargetOrigin) map[string]any {
	m := map[string]any{
		"name":  t.Name,
		"kind":  t.Kind,
		"found": t.Found,
	}
	if t.Introduced != "" {
		m["introduced"] = t.Introduced
	}
	if len(t.Versions) > 0 {
		m["versions"] = t.Versions
	}
	return m
}
