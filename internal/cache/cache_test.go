package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	type Data struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	input := Data{Name: "test", Age: 42}
	if err := s.Set("test_key", input); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	var output Data
	if err := s.Get("test_key", &output); err != nil {
		t.Fatalf("Get error: %v", err)
	}

	if output.Name != "test" || output.Age != 42 {
		t.Errorf("got %+v, expected {test 42}", output)
	}
}

func TestCacheMiss(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	var v string
	err := s.Get("nonexistent", &v)
	if err != ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss, got %v", err)
	}
}

func TestCacheStale(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir, ttl: 1 * time.Millisecond}

	if err := s.Set("stale_test", "value"); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	time.Sleep(5 * time.Millisecond)

	var v string
	err := s.Get("stale_test", &v)
	if err != ErrCacheStale {
		t.Fatalf("expected ErrCacheStale, got %v", err)
	}
}

func TestCacheDelete(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	if err := s.Set("del_test", "value"); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	if err := s.Delete("del_test"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}

	var v string
	if err := s.Get("del_test", &v); err != ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestCacheDeleteMissing(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	if err := s.Delete("never_existed"); err != nil {
		t.Fatalf("Delete missing should not error: %v", err)
	}
}

func TestCacheClear(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	if err := s.Set("a", 1); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	if err := s.Set("b", 2); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear error: %v", err)
	}

	var v int
	if err := s.Get("a", &v); err != ErrCacheMiss {
		t.Errorf("expected miss after clear, got %v", err)
	}
}

func TestStoreStruct(t *testing.T) {
	origDir := CacheDir
	CacheDir = t.TempDir()
	defer func() { CacheDir = origDir }()

	s := New(0)
	if s.ttl != 0 {
		t.Errorf("expected ttl=0, got %v", s.ttl)
	}
	if _, err := os.Stat(s.dir); os.IsNotExist(err) {
		t.Error("cache dir was not created")
	}
}

func TestCacheClearEmptyDir(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	if err := s.Clear(); err != nil {
		t.Fatalf("Clear on empty dir: %v", err)
	}
}

func TestCacheGetInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{invalid"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	var v string
	err := s.Get("bad", &v)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCacheGetNonExistentFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	s := &Store{dir: dir}

	var v string
	err := s.Get("test", &v)
	if err != ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss, got %v", err)
	}
}

func TestCacheGetCorruptEntry(t *testing.T) {
	dir := t.TempDir()
	s := &Store{dir: dir}

	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	var v string
	err := s.Get("corrupt", &v)
	if err == nil {
		t.Fatal("expected error for corrupt entry")
	}
}

func TestPath(t *testing.T) {
	s := &Store{dir: "/tmp/cache"}
	p := s.path("foo")
	expected := filepath.Join("/tmp/cache", "foo.json")
	if p != expected {
		t.Errorf("expected %s, got %s", expected, p)
	}
}
