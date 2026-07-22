package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiffCSS(t *testing.T) {
	dir := t.TempDir()
	oldDir := filepath.Join(dir, "1.0.0")
	newDir := filepath.Join(dir, "1.0.1")
	os.MkdirAll(oldDir, 0755)
	os.MkdirAll(newDir, 0755)

	os.WriteFile(filepath.Join(oldDir, "app.css"), []byte("body {\n  color: red;\n}\n"), 0644)
	os.WriteFile(filepath.Join(newDir, "app.css"), []byte("body {\n  color: blue;\n}\n"), 0644)

	origDir := diffCSSDir
	diffCSSDir = dir
	defer func() { diffCSSDir = origDir }()

	result := DiffCSS("1.0.0", "1.0.1")
	if result.Error != nil {
		t.Fatalf("DiffCSS error: %v", result.Error)
	}
	if !result.HasDiff {
		t.Error("expected HasDiff=true")
	}

	if result.Diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestDiffCSSNoDiff(t *testing.T) {
	dir := t.TempDir()
	verDir := filepath.Join(dir, "1.0.0")
	os.MkdirAll(verDir, 0755)
	os.WriteFile(filepath.Join(verDir, "app.css"), []byte("body { color: red; }\n"), 0644)

	origDir := diffCSSDir
	diffCSSDir = dir
	defer func() { diffCSSDir = origDir }()

	result := DiffCSS("1.0.0", "1.0.0")
	if result.Error != nil {
		t.Fatalf("DiffCSS error: %v", result.Error)
	}
	if result.HasDiff {
		t.Error("expected HasDiff=false for same version")
	}
}

func TestDiffCSSMissingFile(t *testing.T) {
	dir := t.TempDir()

	origDir := diffCSSDir
	diffCSSDir = dir
	defer func() { diffCSSDir = origDir }()

	result := DiffCSS("nonexistent", "1.0.0")
	if result.Error == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestDiffCSSVersionFields(t *testing.T) {
	dir := t.TempDir()
	aDir := filepath.Join(dir, "v1")
	bDir := filepath.Join(dir, "v2")
	os.MkdirAll(aDir, 0755)
	os.MkdirAll(bDir, 0755)
	os.WriteFile(filepath.Join(aDir, "app.css"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(bDir, "app.css"), []byte("b"), 0644)

	origDir := diffCSSDir
	diffCSSDir = dir
	defer func() { diffCSSDir = origDir }()

	result := DiffCSS("v1", "v2")
	if result.Error != nil {
		t.Fatalf("DiffCSS error: %v", result.Error)
	}
	if result.VersionA != "v1" {
		t.Errorf("expected VersionA=v1, got %s", result.VersionA)
	}
	if result.VersionB != "v2" {
		t.Errorf("expected VersionB=v2, got %s", result.VersionB)
	}
}
