package core

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func buildMinimalASARGz(t *testing.T) []byte {
	asarHeader := struct {
		Files map[string]struct {
			Offset string `json:"offset"`
			Size   int    `json:"size"`
		} `json:"files"`
	}{
		Files: map[string]struct {
			Offset string `json:"offset"`
			Size   int    `json:"size"`
		}{
			"app.css": {Offset: "0", Size: 6},
		},
	}
	headerJSON, err := json.Marshal(asarHeader)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var asarData []byte
	asarData = append(asarData, make([]byte, 12)...)
	sizeBytes := make([]byte, 4)
	sizeBytes[0] = byte(len(headerJSON))
	sizeBytes[1] = byte(len(headerJSON) >> 8)
	sizeBytes[2] = byte(len(headerJSON) >> 16)
	sizeBytes[3] = byte(len(headerJSON) >> 24)
	asarData = append(asarData, sizeBytes...)
	asarData = append(asarData, headerJSON...)
	asarData = append(asarData, []byte("body{}")...)

	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(asarData); err != nil {
		t.Fatalf("gzip.Write: %v", err)
	}
	gz.Close()
	return []byte(buf.String())
}

func TestExtractCSSDirCreation(t *testing.T) {
	orig := extractCSSDir
	extractCSSDir = t.TempDir()
	defer func() { extractCSSDir = orig }()

	origURL := asarReleaseURL
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	asarReleaseURL = ts.URL
	defer func() { asarReleaseURL = origURL }()

	_, err := ExtractCSS("999.999.999")
	if err == nil {
		t.Fatal("expected error for nonexistent version")
	}
}

func TestExtractCSSHTTPError(t *testing.T) {
	orig := extractCSSDir
	extractCSSDir = t.TempDir()
	defer func() { extractCSSDir = orig }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	origURL := asarReleaseURL
	asarReleaseURL = ts.URL
	defer func() { asarReleaseURL = origURL }()

	_, err := ExtractCSS("1.0.0")
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 error, got %v", err)
	}
}

func TestExtractCSSBadGzip(t *testing.T) {
	orig := extractCSSDir
	extractCSSDir = t.TempDir()
	defer func() { extractCSSDir = orig }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not gzip data"))
	}))
	defer ts.Close()

	origURL := asarReleaseURL
	asarReleaseURL = ts.URL
	defer func() { asarReleaseURL = origURL }()

	_, err := ExtractCSS("1.0.0")
	if err == nil {
		t.Fatal("expected error for bad gzip")
	}
}

func TestExtractCSSDecompressError(t *testing.T) {
	orig := extractCSSDir
	extractCSSDir = t.TempDir()
	defer func() { extractCSSDir = orig }()

	origClient := httpClient
	origURL := asarReleaseURL
	defer func() { httpClient = origClient; asarReleaseURL = origURL }()

	// gzip with wrong CRC
	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte("test"))
	gz.Close()
	gzipped := []byte(buf.String())
	gzipped[len(gzipped)-1] ^= 0xFF

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(buf.String()))
	}))
	defer ts.Close()
	asarReleaseURL = ts.URL
	httpClient = ts.Client()

	_, err := ExtractCSS("1.0.0")
	if err == nil {
		t.Fatal("expected error for corrupted gzip")
	}
}
