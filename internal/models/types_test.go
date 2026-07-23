package models

import (
	"errors"
	"testing"
	"time"
)

func TestVersionTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		actual   VersionType
		expected string
	}{
		{"Desktop", Desktop, "Desktop"},
		{"Mobile", Mobile, "Mobile"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.actual) != tt.expected {
				t.Errorf("got %q, want %q", tt.actual, tt.expected)
			}
		})
	}
}

func TestRSSVersionStruct(t *testing.T) {
	v := RSSVersion{
		Version:   "1.0",
		Type:      Desktop,
		Date:      "2025-01-01",
		Electron:  "28.0.0",
		Title:     "Release 1.0",
		IsEarly:   false,
		Chromium:  "120.0",
		DockerTag: "v1.0",
	}

	if v.Version != "1.0" {
		t.Errorf("Version = %q, want %q", v.Version, "1.0")
	}
	if v.Type != Desktop {
		t.Errorf("Type = %v, want %v", v.Type, Desktop)
	}
	if v.Date != "2025-01-01" {
		t.Errorf("Date = %q, want %q", v.Date, "2025-01-01")
	}
	if v.Electron != "28.0.0" {
		t.Errorf("Electron = %q, want %q", v.Electron, "28.0.0")
	}
	if v.Title != "Release 1.0" {
		t.Errorf("Title = %q, want %q", v.Title, "Release 1.0")
	}
	if v.IsEarly != false {
		t.Errorf("IsEarly = %v, want %v", v.IsEarly, false)
	}
	if v.Chromium != "120.0" {
		t.Errorf("Chromium = %q, want %q", v.Chromium, "120.0")
	}
	if v.DockerTag != "v1.0" {
		t.Errorf("DockerTag = %q, want %q", v.DockerTag, "v1.0")
	}
}

func TestDockerTagStruct(t *testing.T) {
	d := DockerTag{
		Version:     "2.0",
		Tag:         "latest",
		LastUpdated: "2025-06-01",
	}

	if d.Version != "2.0" {
		t.Errorf("Version = %q, want %q", d.Version, "2.0")
	}
	if d.Tag != "latest" {
		t.Errorf("Tag = %q, want %q", d.Tag, "latest")
	}
	if d.LastUpdated != "2025-06-01" {
		t.Errorf("LastUpdated = %q, want %q", d.LastUpdated, "2025-06-01")
	}
}

func TestCacheEntryStruct(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	e := CacheEntry{
		Data:      []byte("test data"),
		Timestamp: now,
	}

	if string(e.Data) != "test data" {
		t.Errorf("Data = %q, want %q", string(e.Data), "test data")
	}
	if !e.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", e.Timestamp, now)
	}
}

func TestFetchResultStruct(t *testing.T) {
	tests := []struct {
		name       string
		result     FetchResult
		wantError  bool
		errorMatch string
	}{
		{
			name: "without error",
			result: FetchResult{
				RSS:      []RSSVersion{{Version: "1.0"}},
				Docker:   []DockerTag{{Tag: "latest"}},
				Electron: ElectronMap{"28": "28.0.0"},
				Error:    nil,
			},
			wantError: false,
		},
		{
			name: "with error",
			result: FetchResult{
				RSS:      nil,
				Docker:   nil,
				Electron: nil,
				Error:    errors.New("fetch failed"),
			},
			wantError:  true,
			errorMatch: "fetch failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantError {
				if tt.result.Error == nil {
					t.Fatal("expected error, got nil")
				}
				if !contains(tt.result.Error.Error(), tt.errorMatch) {
					t.Errorf("error = %q, want it to contain %q", tt.result.Error.Error(), tt.errorMatch)
				}
			} else {
				if tt.result.Error != nil {
					t.Errorf("unexpected error: %v", tt.result.Error)
				}
				if len(tt.result.RSS) != 1 {
					t.Errorf("len(RSS) = %d, want %d", len(tt.result.RSS), 1)
				}
				if len(tt.result.Docker) != 1 {
					t.Errorf("len(Docker) = %d, want %d", len(tt.result.Docker), 1)
				}
				if tt.result.Electron == nil {
					t.Error("Electron map is nil")
				}
			}
		})
	}
}

func TestExtractResultStruct(t *testing.T) {
	tests := []struct {
		name       string
		result     ExtractResult
		wantError  bool
		errorMatch string
	}{
		{
			name: "without error",
			result: ExtractResult{
				Version: "1.0",
				Path:    "/tmp/extract",
				Error:   nil,
			},
			wantError: false,
		},
		{
			name: "with error",
			result: ExtractResult{
				Version: "",
				Path:    "",
				Error:   errors.New("extraction failed"),
			},
			wantError:  true,
			errorMatch: "extraction failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantError {
				if tt.result.Error == nil {
					t.Fatal("expected error, got nil")
				}
				if !contains(tt.result.Error.Error(), tt.errorMatch) {
					t.Errorf("error = %q, want it to contain %q", tt.result.Error.Error(), tt.errorMatch)
				}
			} else {
				if tt.result.Error != nil {
					t.Errorf("unexpected error: %v", tt.result.Error)
				}
				if tt.result.Version != "1.0" {
					t.Errorf("Version = %q, want %q", tt.result.Version, "1.0")
				}
				if tt.result.Path != "/tmp/extract" {
					t.Errorf("Path = %q, want %q", tt.result.Path, "/tmp/extract")
				}
			}
		})
	}
}

func TestDiffResultStruct(t *testing.T) {
	tests := []struct {
		name       string
		result     DiffResult
		wantError  bool
		errorMatch string
	}{
		{
			name: "has diff without error",
			result: DiffResult{
				VersionA: "1.0",
				VersionB: "2.0",
				Diff:     "line1\nline2",
				HasDiff:  true,
				Error:    nil,
			},
			wantError: false,
		},
		{
			name: "no diff without error",
			result: DiffResult{
				VersionA: "1.0",
				VersionB: "1.0",
				Diff:     "",
				HasDiff:  false,
				Error:    nil,
			},
			wantError: false,
		},
		{
			name: "with error",
			result: DiffResult{
				VersionA: "",
				VersionB: "",
				Diff:     "",
				HasDiff:  false,
				Error:    errors.New("diff failed"),
			},
			wantError:  true,
			errorMatch: "diff failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantError {
				if tt.result.Error == nil {
					t.Fatal("expected error, got nil")
				}
				if !contains(tt.result.Error.Error(), tt.errorMatch) {
					t.Errorf("error = %q, want it to contain %q", tt.result.Error.Error(), tt.errorMatch)
				}
			} else {
				if tt.result.Error != nil {
					t.Errorf("unexpected error: %v", tt.result.Error)
				}
				_ = tt.result.VersionA
				_ = tt.result.VersionB
				_ = tt.result.Diff
				_ = tt.result.HasDiff
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
