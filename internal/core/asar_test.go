package core

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAppCSSFromASAR(t *testing.T) {
	dir := t.TempDir()
	asarPath := filepath.Join(dir, "test.asar")
	destPath := filepath.Join(dir, "app.css")

	f, err := os.Create(asarPath)
	if err != nil {
		t.Fatalf("create asar: %v", err)
	}

	header := asarHeader{
		Files: map[string]asarEntry{
			"app.css": {Offset: "0", Size: 13},
		},
	}
	headerJSON, _ := json.Marshal(header)

	magic := []byte{0x04, 0x1A, 0x2B, 0x3C}
	f.Write(magic)
	binary.Write(f, binary.LittleEndian, uint32(len(headerJSON)))
	f.Write(headerJSON)
	f.WriteString("Hello, World!")
	f.Close()

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

func TestExtractAppCSSFromASARNotFound(t *testing.T) {
	dir := t.TempDir()
	asarPath := filepath.Join(dir, "test.asar")
	destPath := filepath.Join(dir, "app.css")

	f, _ := os.Create(asarPath)
	header := asarHeader{Files: map[string]asarEntry{}}
	headerJSON, _ := json.Marshal(header)

	magic := []byte{0x04, 0x1A, 0x2B, 0x3C}
	f.Write(magic)
	binary.Write(f, binary.LittleEndian, uint32(len(headerJSON)))
	f.Write(headerJSON)
	f.Close()

	err := extractAppCSSFromASAR(asarPath, destPath)
	if err == nil {
		t.Fatal("expected error for missing app.css")
	}
}
