package sources

import (
	"testing"

	"github.com/bladeacer/ocd/internal/models"
)

func TestFillElectron(t *testing.T) {
	versions := []models.RSSVersion{
		{Version: "1.1.0", Electron: ""},
		{Version: "1.0.3", Electron: "28.0.0"},
		{Version: "1.0.2", Electron: ""},
		{Version: "1.0.1", Electron: ""},
		{Version: "1.0.0", Electron: "27.0.0"},
	}

	result := fillElectron(versions)

	tests := []struct {
		version  string
		expected string
	}{
		{"1.1.0", "28.0.0"},
		{"1.0.3", "28.0.0"},
		{"1.0.2", "27.0.0"},
		{"1.0.1", "27.0.0"},
		{"1.0.0", "27.0.0"},
	}

	for _, tc := range tests {
		found := false
		for _, v := range result {
			if v.Version == tc.version {
				if v.Electron != tc.expected {
					t.Errorf("version %s: expected electron %s, got %s", tc.version, tc.expected, v.Electron)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("version %s not found in result", tc.version)
		}
	}
}

func TestFillElectronEmpty(t *testing.T) {
	versions := []models.RSSVersion{
		{Version: "1.0.0", Electron: ""},
	}

	result := fillElectron(versions)
	if len(result) != 1 {
		t.Fatalf("expected 1 version, got %d", len(result))
	}
	if result[0].Electron != "13.0.0" {
		t.Errorf("expected fallback 13.0.0, got %s", result[0].Electron)
	}
}

func TestFillElectronAllSet(t *testing.T) {
	versions := []models.RSSVersion{
		{Version: "1.0.0", Electron: "25.0.0"},
		{Version: "1.0.1", Electron: "26.0.0"},
	}

	result := fillElectron(versions)
	if len(result) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(result))
	}
	if result[0].Electron != "25.0.0" {
		t.Errorf("expected 25.0.0, got %s", result[0].Electron)
	}
	if result[1].Electron != "26.0.0" {
		t.Errorf("expected 26.0.0, got %s", result[1].Electron)
	}
}
