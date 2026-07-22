package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		v    string
		want []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"0.0.0", []int{0, 0, 0}},
		{"10.20.30", []int{10, 20, 30}},
		{"1.2", []int{1, 2, 0}},
		{"abc", []int{0, 0, 0}},
		{"1.2.3.4", []int{1, 2, 3}},
	}
	for _, tt := range tests {
		got := parseVersion(tt.v)
		if len(got) != 3 {
			t.Errorf("parseVersion(%q) len = %d, want 3", tt.v, len(got))
			continue
		}
		for i := 0; i < 3; i++ {
			if got[i] != tt.want[i] {
				t.Errorf("parseVersion(%q)[%d] = %d, want %d", tt.v, i, got[i], tt.want[i])
			}
		}
	}
}

func TestAtoi(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"123", 123},
		{"0", 0},
		{"abc", 0},
		{"12a34", 1234},
		{"", 0},
		{"999", 999},
	}
	for _, tt := range tests {
		got := atoi(tt.s)
		if got != tt.want {
			t.Errorf("atoi(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b []int
		want int
	}{
		{[]int{1, 0, 0}, []int{1, 0, 0}, 0},
		{[]int{2, 0, 0}, []int{1, 0, 0}, 1},
		{[]int{1, 0, 0}, []int{2, 0, 0}, -1},
		{[]int{1, 5, 0}, []int{1, 3, 0}, 2},
		{[]int{1, 0, 5}, []int{1, 0, 3}, 2},
	}
	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSortPriority(t *testing.T) {
	tests := []struct {
		row  table.Row
		want int
	}{
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", "[x]"}, 0},
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "Found", "v28", "120", ""}, 1},
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "N/A", "v28", "120", ""}, 2},
		{table.Row{"0", "1.0.0", "Desktop", "2024-01-01", "Missing", "v28", "120", ""}, 3},
		{table.Row{"0", "1.0.0", "Mobile", "2024-01-01", "", "v28", "120", ""}, 4},
	}
	for _, tt := range tests {
		got := sortPriority(tt.row)
		if got != tt.want {
			t.Errorf("sortPriority(%v) = %d, want %d", tt.row, got, tt.want)
		}
	}
}

func TestFmtElectron(t *testing.T) {
	tests := []struct {
		v    string
		want string
	}{
		{"28.0.0", "v28.0.0"},
		{"", "---"},
		{"1.0.0", "v1.0.0"},
	}
	for _, tt := range tests {
		got := fmtElectron(tt.v)
		if got != tt.want {
			t.Errorf("fmtElectron(%q) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestTableStyles(t *testing.T) {
	s := tableStyles()
	_ = s.Header
	_ = s.Selected
}

func TestFmtStatus(t *testing.T) {
	active := fmtStatus("M", true)
	inactive := fmtStatus("M", false)
	if active == "" {
		t.Error("expected non-empty active status")
	}
	if inactive == "" {
		t.Error("expected non-empty inactive status")
	}
	if active == inactive {
		t.Log("note: active and inactive styles may render differently with ANSI")
	}
}

//nolint:unused
func newTestSearch() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "test"
	ti.CharLimit = 50
	ti.Width = 20
	return ti
}
