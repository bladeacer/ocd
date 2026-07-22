package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractCSSCached(t *testing.T) {
	dir := t.TempDir()
	cssDir := filepath.Join(dir, "1.0.0")
	if err := os.MkdirAll(cssDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cssFile := filepath.Join(cssDir, "app.css")
	if err := os.WriteFile(cssFile, []byte("body{}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	orig := extractCSSDir
	extractCSSDir = dir
	defer func() { extractCSSDir = orig }()

	path, err := ExtractCSS("1.0.0")
	if err != nil {
		t.Fatalf("ExtractCSS error: %v", err)
	}
	if path != cssFile {
		t.Errorf("expected %s, got %s", cssFile, path)
	}
}

func TestExtractCSSDirCreation(t *testing.T) {
	orig := extractCSSDir
	extractCSSDir = t.TempDir()
	defer func() { extractCSSDir = orig }()

	_, err := ExtractCSS("999.999.999")
	if err == nil {
		t.Fatal("expected error for nonexistent version")
	}
}
