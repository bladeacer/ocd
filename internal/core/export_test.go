package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportFile(t *testing.T) {
	dir := t.TempDir()

	// Test TOML export.
	path, err := ExportFile(func() ([]byte, error) { return []byte("key = \"value\"\n"), nil }, dir, "test", "toml")
	if err != nil {
		t.Fatalf("ExportFile: %v", err)
	}
	if !strings.HasSuffix(path, ".toml") {
		t.Errorf("expected .toml extension, got %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "key") {
		t.Errorf("expected file to contain key, got %s", string(data))
	}

	// Test JSON export.
	path, err = ExportFile(func() ([]byte, error) { return []byte("{}"), nil }, dir, "test", "json")
	if err != nil {
		t.Fatalf("ExportFile: %v", err)
	}
	if !strings.HasSuffix(path, ".json") {
		t.Errorf("expected .json extension, got %s", path)
	}

	// Test YAML export.
	path, err = ExportFile(func() ([]byte, error) { return []byte("key: value\n"), nil }, dir, "test", "yaml")
	if err != nil {
		t.Fatalf("ExportFile: %v", err)
	}
	if !strings.HasSuffix(path, ".yaml") {
		t.Errorf("expected .yaml extension, got %s", path)
	}

	// Test directory creation.
	nestedDir := filepath.Join(dir, "nested", "deep")
	path, err = ExportFile(func() ([]byte, error) { return []byte("x"), nil }, nestedDir, "file", "toml")
	if err != nil {
		t.Fatalf("ExportFile with nested dir: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to exist in nested dir")
	}

	// Test empty format (no extension).
	path, err = ExportFile(func() ([]byte, error) { return []byte("x"), nil }, dir, "file", "")
	if err != nil {
		t.Fatalf("ExportFile with empty format: %v", err)
	}
	if strings.Contains(path, ".") {
		t.Errorf("expected no extension with empty format, got %s", path)
	}
}

func TestExportFileMarshalError(t *testing.T) {
	_, err := ExportFile(func() ([]byte, error) { return nil, os.ErrInvalid }, "", "test", "toml")
	if err == nil {
		t.Error("expected error from marshal function")
	}
}

func TestExpandDir(t *testing.T) {
	home, _ := os.UserHomeDir()

	// Test ~/ expansion.
	result := expandDir("~/testdir")
	expected := filepath.Join(home, "testdir")
	if result != expected {
		t.Errorf("expandDir(\"~/testdir\") = %q, want %q", result, expected)
	}

	// Test env var expansion.
	os.Setenv("OCD_TEST_DIR", "/tmp/ocd-test")
	defer os.Unsetenv("OCD_TEST_DIR")
	result = expandDir("$OCD_TEST_DIR")
	if result != "/tmp/ocd-test" {
		t.Errorf("expandDir with env var = %q, want /tmp/ocd-test", result)
	}

	// Test no expansion needed.
	result = expandDir("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("expandDir(\"/absolute/path\") = %q, want /absolute/path", result)
	}
}

func TestFormatToExtension(t *testing.T) {
	tests := []struct {
		format string
		want   string
	}{
		{"toml", ".toml"},
		{"json", ".json"},
		{"yaml", ".yaml"},
		{"yml", ".yaml"},
		{"", ""},
		{"unknown", ""},
		{" TOML ", ".toml"},
		{"JSON", ".json"},
	}
	for _, tt := range tests {
		got := formatToExtension(tt.format)
		if got != tt.want {
			t.Errorf("formatToExtension(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}
