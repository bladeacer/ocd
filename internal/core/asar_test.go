package core

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type asarFile struct {
	name string
	data string
}

func writeASAR(t *testing.T, files []asarFile) string {
	t.Helper()

	header := asarHeader{Files: make(map[string]asarEntry)}
	offset := 0
	for _, f := range files {
		header.Files[f.name] = asarEntry{
			Offset: fmt.Sprintf("%d", offset),
			Size:   len(f.data),
		}
		offset += len(f.data)
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	dir := t.TempDir()
	asarPath := filepath.Join(dir, "test.asar")
	f, err := os.Create(asarPath)
	if err != nil {
		t.Fatalf("create asar: %v", err)
	}
	defer f.Close()

	if _, err := f.Write(make([]byte, 12)); err != nil {
		t.Fatalf("write padding: %v", err)
	}
	if err := binary.Write(f, binary.LittleEndian, uint32(len(headerJSON))); err != nil {
		t.Fatalf("write json size: %v", err)
	}
	if _, err := f.Write(headerJSON); err != nil {
		t.Fatalf("write header json: %v", err)
	}
	for _, fe := range files {
		if _, err := f.WriteString(fe.data); err != nil {
			t.Fatalf("write file data: %v", err)
		}
	}

	return asarPath
}

func TestExtractAppCSSFromASAR(t *testing.T) {
	asarPath := writeASAR(t, []asarFile{{"app.css", "Hello, World!"}})
	destPath := filepath.Join(filepath.Dir(asarPath), "app.css")

	if err := extractAppCSSFromASAR(asarPath, destPath); err != nil {
		t.Fatalf("extractAppCSSFromASAR error: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("read extracted css: %v", err)
	}
	if string(data) != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", string(data))
	}
}

func TestExtractAppCSSFromASARMultiFile(t *testing.T) {
	asarPath := writeASAR(t, []asarFile{
		{"app.css", "body { color: red; }"},
		{"other", "junk"},
	})
	destPath := filepath.Join(filepath.Dir(asarPath), "app.css")

	if err := extractAppCSSFromASAR(asarPath, destPath); err != nil {
		t.Fatalf("extractAppCSSFromASAR error: %v", err)
	}

	data, _ := os.ReadFile(destPath)
	if string(data) != "body { color: red; }" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestExtractAppCSSFromASARNotFound(t *testing.T) {
	asarPath := writeASAR(t, []asarFile{{"other.css", "x"}})
	destPath := filepath.Join(filepath.Dir(asarPath), "app.css")

	err := extractAppCSSFromASAR(asarPath, destPath)
	if err == nil {
		t.Fatal("expected error for missing app.css")
	}
}

func TestExtractAppCSSFromASARLargeContent(t *testing.T) {
	data := strings.Repeat("x", 10000)
	asarPath := writeASAR(t, []asarFile{{"app.css", data}})
	destPath := filepath.Join(filepath.Dir(asarPath), "app.css")

	if err := extractAppCSSFromASAR(asarPath, destPath); err != nil {
		t.Fatalf("extractAppCSSFromASAR error: %v", err)
	}

	result, _ := os.ReadFile(destPath)
	if len(result) != 10000 {
		t.Errorf("expected 10000 bytes, got %d", len(result))
	}
}

func TestExtractAppCSSFromASAROffsetOrder(t *testing.T) {
	asarPath := writeASAR(t, []asarFile{
		{"a", "first"},
		{"app.css", "target"},
		{"b", "last"},
	})
	destPath := filepath.Join(filepath.Dir(asarPath), "app.css")

	if err := extractAppCSSFromASAR(asarPath, destPath); err != nil {
		t.Fatalf("extractAppCSSFromASAR error: %v", err)
	}

	data, _ := os.ReadFile(destPath)
	if string(data) != "target" {
		t.Errorf("expected 'target', got '%s'", string(data))
	}
}

func TestExtractAppCSSFromASAREmptyContent(t *testing.T) {
	asarPath := writeASAR(t, []asarFile{{"app.css", ""}})
	destPath := filepath.Join(filepath.Dir(asarPath), "app.css")

	if err := extractAppCSSFromASAR(asarPath, destPath); err != nil {
		t.Fatalf("extractAppCSSFromASAR error: %v", err)
	}
	data, _ := os.ReadFile(destPath)
	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

func TestExtractAppCSSFromASARTruncated(t *testing.T) {
	dir := t.TempDir()
	asarPath := filepath.Join(dir, "truncated.asar")
	f, err := os.Create(asarPath)
	if err != nil {
		t.Fatalf("create asar: %v", err)
	}
	_, _ = f.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	f.Close()

	destPath := filepath.Join(dir, "app.css")
	err = extractAppCSSFromASAR(asarPath, destPath)
	if err == nil {
		t.Fatal("expected error for truncated asar")
	}
}
