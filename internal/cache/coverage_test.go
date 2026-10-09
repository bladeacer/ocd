package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClearRemovesEveryEntry(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	if err := s.Set("a", "one"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set("b", "two"); err != nil {
		t.Fatal(err)
	}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected an empty directory, got %d entries", len(entries))
	}
}

func TestClearOnMissingDirectory(t *testing.T) {
	s := &Store{dir: filepath.Join(t.TempDir(), "gone")}
	if err := s.Clear(); err != nil {
		t.Errorf("clearing a missing directory should be a no-op, got %v", err)
	}
}

func TestNewCreatesTheDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "cache")

	orig := CacheDir
	CacheDir = dir
	defer func() { CacheDir = orig }()

	s, err := New(time.Hour)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("New should create %q", dir)
	}
	if s.dir != dir {
		t.Errorf("dir = %q, want %q", s.dir, dir)
	}
}

func TestStaleEntry(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	type data struct {
		Name string `json:"name"`
	}
	if err := s.Set("old", data{Name: "x"}); err != nil {
		t.Fatal(err)
	}

	// Rewrite the entry with an old timestamp.
	path := filepath.Join(dir, "old.json")
	if err := os.WriteFile(path,
		[]byte(`{"data":{"name":"x"},"timestamp":"2000-01-01T00:00:00Z"}`), 0644); err != nil {
		t.Fatal(err)
	}

	shortLived := &Store{dir: dir, ttl: time.Hour}
	var out data
	if err := shortLived.Get("old", &out); err != ErrCacheStale {
		t.Errorf("Get with a one hour ttl = %v, want ErrCacheStale", err)
	}

	forever := &Store{dir: dir}
	if err := forever.Get("old", &out); err != nil {
		t.Errorf("a zero ttl should never go stale, got %v", err)
	}
}

func TestGetOnCorruptEntry(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	s := &Store{dir: dir}
	var out string
	if err := s.Get("bad", &out); err == nil {
		t.Error("expected an error for a corrupt entry")
	}
}

func TestDeleteMissingKey(t *testing.T) {
	s := &Store{dir: t.TempDir()}
	if err := s.Delete("nope"); err != nil {
		t.Errorf("deleting a missing key should be a no-op, got %v", err)
	}
}

func TestSetRejectsUnmarshalableValue(t *testing.T) {
	s := &Store{dir: t.TempDir()}
	if err := s.Set("bad", make(chan int)); err == nil {
		t.Error("expected an error for a value that cannot be marshalled")
	}
}
