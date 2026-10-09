package core

import (
	"strings"
	"testing"
)

func TestSplitSelector(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{".a", []string{".a"}},
		{".a .b", []string{".a", " ", ".b"}},
		{".a>.b", []string{".a", ">", ".b"}},
		{".a > .b", []string{".a", ">", ".b"}},
		{".a  +  .b ~ .c", []string{".a", "+", ".b", "~", ".c"}},
		{"a b c", []string{"a", " ", "b", " ", "c"}},
		{"", nil},
	}
	for _, tc := range cases {
		got := splitSelector(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("splitSelector(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitSelector(%q) = %v, want %v", tc.in, got, tc.want)
				break
			}
		}
	}
}

func TestCompoundBase(t *testing.T) {
	cases := map[string]string{
		".a":         ".a",
		".a:hover":   ".a",
		".a::before": ".a",
		".a[data-x]": ".a",
		"[data-x]":   "",
		"":           "",
	}
	for in, want := range cases {
		if got := compoundBase(in); got != want {
			t.Errorf("compoundBase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSelectorCovers(t *testing.T) {
	cases := []struct {
		theme, target string
		want          bool
	}{
		{".a", ".a", true},
		{".a", ".a:hover", true},
		{".a", ".a .b", true},
		{".a", ".a > .b", true},
		{".a", ".b", false},
		{".a", ".ab", false},
		{".a .b", ".a .b", true},
		{".a .b", ".a .b .c", true},
		{".a .b", ".a", false},
		{".a .b", ".a .c", false},
		{"body", "body .x", true},
		{"*", "*", true},
		{"*", ".a", false},
		{".workspace-leaf", ".workspace-leaf.mod-active .cm-content", true},
		{".cm", ".cm-header", false},
	}
	for _, tc := range cases {
		if got := SelectorCovers(tc.theme, tc.target); got != tc.want {
			t.Errorf("SelectorCovers(%q, %q) = %v, want %v", tc.theme, tc.target, got, tc.want)
		}
	}
}

func TestCompareSelectors(t *testing.T) {
	target := ParseCSS(`.a{color:red}.b{color:red}.c{color:red}`)
	theme := ParseCSS(`.a{color:blue}.d{color:green}`)

	rep := CompareSelectors("1.6.3", "theme.css", target, theme)

	if rep.TargetCount != 3 {
		t.Errorf("TargetCount = %d, want 3", rep.TargetCount)
	}
	if rep.ThemeCount != 2 {
		t.Errorf("ThemeCount = %d, want 2", rep.ThemeCount)
	}
	if rep.Styled != 1 {
		t.Errorf("Styled = %d, want 1", rep.Styled)
	}
	if rep.Covered != 1 {
		t.Errorf("Covered = %d, want 1", rep.Covered)
	}
	if rep.Uncovered != 2 {
		t.Errorf("Uncovered = %d, want 2", rep.Uncovered)
	}
	if len(rep.Extra) != 1 || rep.Extra[0] != ".d" {
		t.Errorf("Extra = %v, want [.d]", rep.Extra)
	}
}

func TestCompareSelectorsNilIndex(t *testing.T) {
	rep := CompareSelectors("1.6.3", "theme.css", nil, ParseCSS(".a{}"))
	if rep.TargetCount != 0 || rep.Styled != 0 {
		t.Errorf("a nil target index must give an empty report: %+v", rep)
	}
	rep = CompareSelectors("1.6.3", "theme.css", ParseCSS(".a{}"), nil)
	if rep.ThemeCount != 0 {
		t.Errorf("a nil theme index must give an empty report: %+v", rep)
	}
}

func TestCompareSelectorsCapsLists(t *testing.T) {
	var targetCSS, themeCSS strings.Builder
	for i := 0; i < SelectorListLimit+10; i++ {
		targetCSS.WriteString(".t" + itoa(i) + "{color:red}")
		themeCSS.WriteString(".x" + itoa(i) + "{color:red}")
	}

	rep := CompareSelectors("1.6.3", "theme.css",
		ParseCSS(targetCSS.String()), ParseCSS(themeCSS.String()))

	if rep.Uncovered != SelectorListLimit+10 {
		t.Errorf("Uncovered = %d, want %d", rep.Uncovered, SelectorListLimit+10)
	}
	if len(rep.UncoveredList) != SelectorListLimit {
		t.Errorf("UncoveredList has %d entries, want the cap of %d", len(rep.UncoveredList), SelectorListLimit)
	}
	out := rep.String()
	if !strings.Contains(out, "and "+itoa(10)+" more") {
		t.Errorf("expected a truncation note, got:\n%s", out)
	}
	if !strings.Contains(out, "list truncated") {
		t.Errorf("expected the extra list to be marked truncated, got:\n%s", out)
	}
}

func TestSelectorReportMarshal(t *testing.T) {
	rep := &SelectorReport{
		TargetVersion: "1.6.3",
		ThemePath:     "theme.css",
		TargetCount:   3,
		ThemeCount:    2,
		Styled:        1,
		Covered:       2,
		Uncovered:     1,
		UncoveredList: []string{".c"},
		Extra:         []string{".d"},
	}

	jsonOut, err := rep.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(jsonOut), `"covered": 2`) {
		t.Errorf("JSON missing covered: %s", jsonOut)
	}

	yamlOut, err := rep.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if !strings.Contains(string(yamlOut), "covered: 2") {
		t.Errorf("YAML missing covered: %s", yamlOut)
	}

	tomlOut, err := rep.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	for _, want := range []string{"covered = 2", "uncovered = 1", `uncovered_list = [".c"]`, `extra = [".d"]`} {
		if !strings.Contains(string(tomlOut), want) {
			t.Errorf("TOML missing %q: %s", want, tomlOut)
		}
	}
}

func TestSelectorReportMarshalOmitsEmptyLists(t *testing.T) {
	rep := &SelectorReport{TargetVersion: "1.0.0"}

	tomlOut, err := rep.MarshalTOML()
	if err != nil {
		t.Fatalf("MarshalTOML: %v", err)
	}
	if strings.Contains(string(tomlOut), "uncovered_list") || strings.Contains(string(tomlOut), "extra") {
		t.Errorf("empty lists should be omitted: %s", tomlOut)
	}
}

func TestSelectorReportStringEmpty(t *testing.T) {
	rep := &SelectorReport{TargetVersion: "1.0.0", ThemePath: "t.css"}
	out := rep.String()
	if !strings.Contains(out, "target selectors:  0") {
		t.Errorf("expected zero counts, got:\n%s", out)
	}
}

func TestCapList(t *testing.T) {
	if got := capList([]string{"a"}, 2); len(got) != 1 {
		t.Errorf("capList short = %v", got)
	}
	if got := capList([]string{"a", "b", "c"}, 2); len(got) != 2 {
		t.Errorf("capList long = %v, want 2 entries", got)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
