package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"github.com/bladeacer/ocd/internal/models"
)

// OriginResult describes the earliest known Obsidian version where a
// selector or CSS variable was first introduced.
type OriginResult struct {
	Target     string   `json:"target" yaml:"target" toml:"target"`
	Kind       string   `json:"kind" yaml:"kind" toml:"kind"`
	Introduced string   `json:"introduced,omitempty" yaml:"introduced,omitempty" toml:"introduced,omitempty"`
	Found      bool     `json:"found" yaml:"found" toml:"found"`
	Versions   []string `json:"versions,omitempty" yaml:"versions,omitempty" toml:"versions,omitempty"`
}

// PublicDesktopVersions extracts version strings from RSSVersion entries,
// keeping only public (non-insider) Desktop releases.
func PublicDesktopVersions(versions []models.RSSVersion) []string {
	var result []string
	for _, v := range versions {
		if v.Type == models.Desktop && !v.IsEarly {
			result = append(result, v.Version)
		}
	}
	return result
}

// SortVersions returns versions sorted in ascending semantic order.
// Versions that do not parse as semver are placed at the end, in string order.
func SortVersions(versions []string) []string {
	sorted := make([]string, len(versions))
	copy(sorted, versions)
	sort.SliceStable(sorted, func(i, j int) bool {
		vi, iok := parseVersion(sorted[i])
		vj, jok := parseVersion(sorted[j])
		switch {
		case !iok && !jok:
			return sorted[i] < sorted[j]
		case !iok:
			return false
		case !jok:
			return true
		}
		for k := 0; k < 3; k++ {
			if vi[k] != vj[k] {
				return vi[k] < vj[k]
			}
		}
		return false
	})
	return sorted
}

var cssCommentRe = regexp.MustCompile(`(?s)/\*.*?\*/`)

func stripCSSComments(s string) string {
	return cssCommentRe.ReplaceAllString(s, "")
}

// ContainsSelector reports whether the given selector is defined in css.
func ContainsSelector(css, selector string) bool {
	if selector == "" {
		return false
	}
	css = stripCSSComments(css)
	for _, line := range strings.Split(css, "\n") {
		if extractSelector(line) == selector {
			return true
		}
	}
	return false
}

// ContainsVariable reports whether the given CSS variable name is defined in css.
func ContainsVariable(css, name string) bool {
	if name == "" {
		return false
	}
	css = stripCSSComments(css)
	for _, line := range strings.Split(css, "\n") {
		m := cssVarNameRe.FindStringSubmatch(line)
		if m != nil && m[1] == name {
			return true
		}
	}
	return false
}

// ListCachedVersions returns the version labels for which CSS has been
// extracted and cached under CSSDir.
func ListCachedVersions() ([]string, error) {
	entries, err := os.ReadDir(CSSDir)
	if err != nil {
		return nil, fmt.Errorf("list cache dir %q: %w", CSSDir, err)
	}
	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cssPath := filepath.Join(CSSDir, e.Name(), "app.css")
		if _, err := os.Stat(cssPath); err == nil {
			versions = append(versions, e.Name())
		}
	}
	return versions, nil
}

// MergeVersions combines two version lists, removing duplicates.
// Versions present in the cached list but missing from the RSS list are
// included under the assumption that cached CSS was extracted from a public
// desktop release.
func MergeVersions(rssVersions, cachedVersions []string) []string {
	seen := make(map[string]bool, len(rssVersions))
	var result []string
	for _, v := range rssVersions {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	for _, v := range cachedVersions {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// FindOrigin scans the cached CSS for each version (sorted ascending) and
// returns the earliest version where target was first introduced.
// The versions argument should contain the public desktop version strings
// to search; versions without cached CSS are silently skipped.
func FindOrigin(target string, isVariable bool, versions []string) *OriginResult {
	kind := "selector"
	if isVariable {
		kind = "variable"
	}

	result := &OriginResult{
		Target: target,
		Kind:   kind,
		Found:  false,
	}

	sorted := SortVersions(versions)
	for _, v := range sorted {
		cssPath := filepath.Join(CSSDir, v, "app.css")
		css, err := os.ReadFile(cssPath)
		if err != nil {
			continue
		}
		var present bool
		if isVariable {
			present = ContainsVariable(string(css), target)
		} else {
			present = ContainsSelector(string(css), target)
		}
		if present {
			result.Versions = append(result.Versions, v)
			if !result.Found {
				result.Introduced = v
				result.Found = true
			}
		}
	}
	return result
}

// String renders the result for stdout.
func (r *OriginResult) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Origin: %s\n", r.Target)
	fmt.Fprintf(&b, "  kind:        %s\n", r.Kind)
	if r.Found {
		fmt.Fprintf(&b, "  introduced:  %s\n", r.Introduced)
		if len(r.Versions) == 1 {
			fmt.Fprintf(&b, "  found in:    %s (1 version)\n", r.Versions[0])
		} else {
			fmt.Fprintf(&b, "  found in:    %s (%d versions)\n", strings.Join(r.Versions, ", "), len(r.Versions))
		}
	} else {
		b.WriteString("  not found in any cached public desktop version\n")
	}
	return b.String()
}

// MarshalJSON renders the result as indented JSON.
func (r *OriginResult) MarshalJSON() ([]byte, error) {
	type Alias OriginResult
	return json.MarshalIndent((*Alias)(r), "", "  ")
}

// MarshalYAML renders the result as YAML.
func (r *OriginResult) MarshalYAML() ([]byte, error) {
	type Alias OriginResult
	return yaml.Marshal((*Alias)(r))
}

// MarshalTOML renders the result as TOML.
func (r *OriginResult) MarshalTOML() ([]byte, error) {
	var buf strings.Builder
	err := r.encodeTOML(&buf)
	return []byte(buf.String()), err
}

// encodeTOML writes the TOML representation of the result to w.
func (r *OriginResult) encodeTOML(w io.Writer) error {
	encoder := toml.NewEncoder(w)
	v := map[string]any{
		"target": r.Target,
		"kind":   r.Kind,
		"found":  r.Found,
	}
	if r.Introduced != "" {
		v["introduced"] = r.Introduced
	}
	if len(r.Versions) > 0 {
		v["versions"] = r.Versions
	}
	return encoder.Encode(v)
}
