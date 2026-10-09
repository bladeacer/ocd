package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompoundTokens(t *testing.T) {
	classes, ids := compoundTokens(".a.b#c")
	if !classes["a"] || !classes["b"] || !ids["c"] {
		t.Errorf("classes=%v ids=%v", classes, ids)
	}

	classes, ids = compoundTokens("div#main.cls:hover[data-x]")
	if !ids["main"] || !classes["cls"] {
		t.Errorf("pseudo and attribute parts must be ignored: classes=%v ids=%v", classes, ids)
	}

	classes, ids = compoundTokens("plain")
	if len(classes) != 0 || len(ids) != 0 {
		t.Errorf("an element-only compound has no tokens: %v %v", classes, ids)
	}

	classes, ids = compoundTokens(".")
	if len(classes) != 0 || len(ids) != 0 {
		t.Errorf("a lone dot has no class or id: %v %v", classes, ids)
	}
}

func TestTempPathIn(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "app.css")

	path, cleanup, err := tempPathIn(dest)
	if err != nil {
		t.Fatalf("tempPathIn: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("temp file %q should live in %q", path, dir)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("temp file should exist: %v", err)
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("cleanup should remove the temp file")
	}
}

func TestTempPathInMissingDir(t *testing.T) {
	if _, _, err := tempPathIn(filepath.Join(t.TempDir(), "nope", "app.css")); err == nil {
		t.Error("expected an error when the parent directory is missing")
	}
}

func TestCopyFileAtomic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.css")
	dst := filepath.Join(dir, "out.css")
	if err := os.WriteFile(src, []byte("body{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFileAtomic(src, dst); err != nil {
		t.Fatalf("copyFileAtomic: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "body{}" {
		t.Errorf("copied content = %q", data)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp file %q was left behind", e.Name())
		}
	}

	if err := copyFileAtomic(filepath.Join(dir, "missing.css"), dst); err == nil {
		t.Error("expected an error for a missing source")
	}
}

func TestImportFileCSSAndErrors(t *testing.T) {
	dir := t.TempDir()
	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = dir }()
	defer func() { CSSDir = orig }()

	src := filepath.Join(dir, "mytheme.css")
	if err := os.WriteFile(src, []byte("body{--a:1}"), 0644); err != nil {
		t.Fatal(err)
	}

	path, err := ImportFile(src, "my-label")
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "body{--a:1}" {
		t.Errorf("imported content = %q", data)
	}
	if !CSSCached("my-label") {
		t.Error("the import should be cached")
	}

	if _, err := ImportFile(filepath.Join(dir, "mytheme.txt"), "bad"); err == nil {
		t.Error("expected an error for an unsupported extension")
	}
	if _, err := ImportFile(filepath.Join(dir, "missing.css"), "missing"); err == nil {
		t.Error("expected an error for a missing source")
	}
}

func TestImportFileBadASAR(t *testing.T) {
	dir := t.TempDir()
	orig := CSSDir
	CSSDir = filepath.Join(dir, "cache")
	defer func() { CSSDir = orig }()

	src := filepath.Join(dir, "bad.asar")
	if err := os.WriteFile(src, []byte("not an asar"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := ImportFile(src, "bad"); err == nil {
		t.Error("expected an error for a malformed asar")
	}
	if CSSCached("bad") {
		t.Error("a failed import must not leave a cached file")
	}
}

func TestEnsureVersionStatus(t *testing.T) {
	useTempCSSDir(t)

	// A fresh cache entry is reported as cached.
	seedCSSFile(t, "1.0.0", "body{}")
	if status, err := ensureVersion("1.0.0", 0, false); status != SweepCached || err != nil {
		t.Errorf("fresh entry = %v, %v; want cached", status, err)
	}

	// Without a server the download fails, which is not "unavailable".
	orig := asarReleaseURL
	asarReleaseURL = "http://127.0.0.1:1/v%s/obsidian-%s.asar.gz"
	defer func() { asarReleaseURL = orig }()

	if status, _ := ensureVersion("9.9.9", 0, true); status != SweepFailed {
		t.Errorf("unreachable host = %v, want failed", status)
	}
}

func seedCSSFile(t *testing.T, version, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(CSSPath(version)), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(CSSPath(version), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
