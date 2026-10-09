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
	"time"
)

func TestBuildMinimalASARGzIsValid(t *testing.T) {
	data := buildMinimalASARGz(t)
	if len(data) == 0 {
		t.Fatal("expected non-empty gzip data")
	}
	r, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	_ = r.Close()
}

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

	orig := CSSDir
	CSSDir = dir
	defer func() { CSSDir = orig }()

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
	_ = gz.Close()
	return []byte(buf.String())
}

func TestExtractCSSDirCreation(t *testing.T) {
	orig := CSSDir
	CSSDir = t.TempDir()
	defer func() { CSSDir = orig }()

	origURL := asarReleaseURL
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	defer func() { asarReleaseURL = origURL }()

	_, err := ExtractCSS("999.999.999")
	if err == nil {
		t.Fatal("expected error for nonexistent version")
	}
}

func TestExtractCSSHTTPError(t *testing.T) {
	orig := CSSDir
	CSSDir = t.TempDir()
	defer func() { CSSDir = orig }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	origURL := asarReleaseURL
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
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
	orig := CSSDir
	CSSDir = t.TempDir()
	defer func() { CSSDir = orig }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not gzip data"))
	}))
	defer ts.Close()

	origURL := asarReleaseURL
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	defer func() { asarReleaseURL = origURL }()

	_, err := ExtractCSS("1.0.0")
	if err == nil {
		t.Fatal("expected error for bad gzip")
	}
}

func TestExtractCSSDecompressError(t *testing.T) {
	orig := CSSDir
	CSSDir = t.TempDir()
	defer func() { CSSDir = orig }()

	origClient := httpClient
	origURL := asarReleaseURL
	defer func() { httpClient = origClient; asarReleaseURL = origURL }()

	// gzip with wrong CRC
	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("test"))
	_ = gz.Close()
	gzipped := []byte(buf.String())
	gzipped[len(gzipped)-1] ^= 0xFF

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(buf.String()))
	}))
	defer ts.Close()
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()

	_, err := ExtractCSS("1.0.0")
	if err == nil {
		t.Fatal("expected error for corrupted gzip")
	}
}

func TestCSSCachedAndModTime(t *testing.T) {
	useTempCSSDir(t)

	if CSSCached("1.0.0") {
		t.Error("CSSCached should be false for a missing entry")
	}
	if _, err := CSSModTime("1.0.0"); err == nil {
		t.Error("CSSModTime should fail for a missing entry")
	}
	if err := os.MkdirAll(filepath.Dir(CSSPath("1.0.0")), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(CSSPath("1.0.0"), []byte("body{}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !CSSCached("1.0.0") {
		t.Error("CSSCached should be true once app.css exists")
	}
	mod, err := CSSModTime("1.0.0")
	if err != nil {
		t.Fatalf("CSSModTime: %v", err)
	}
	if time.Since(mod) > time.Minute {
		t.Errorf("mod time looks wrong: %v", mod)
	}
}

func TestCSSFresh(t *testing.T) {
	useTempCSSDir(t)

	if CSSFresh("1.0.0", 24*time.Hour) {
		t.Error("a missing entry is never fresh")
	}
	if err := os.MkdirAll(filepath.Dir(CSSPath("1.0.0")), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(CSSPath("1.0.0"), []byte("body{}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if !CSSFresh("1.0.0", time.Hour) {
		t.Error("a new entry should be fresh within the window")
	}
	if !CSSFresh("1.0.0", 0) {
		t.Error("a zero ttl means entries never expire")
	}

	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(CSSPath("1.0.0"), old, old); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	if CSSFresh("1.0.0", time.Hour) {
		t.Error("an entry older than the window should not be fresh")
	}
	if !CSSFresh("1.0.0", 0) {
		t.Error("a zero ttl means an aged entry is still used")
	}
}

func TestExtractCSSForceReplacesCachedFile(t *testing.T) {
	useTempCSSDir(t)
	orig := CSSDir
	CSSDir = t.TempDir()
	defer func() { CSSDir = orig }()

	if err := os.MkdirAll(filepath.Dir(CSSPath("1.0.0")), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(CSSPath("1.0.0"), []byte("stale{}"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	origURL := asarReleaseURL
	origClient := httpClient
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buildMinimalASARGz(t))
	}))
	defer ts.Close()
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()
	defer func() { asarReleaseURL = origURL; httpClient = origClient }()

	// Without force the cached copy wins.
	if _, err := ExtractCSS("1.0.0"); err != nil {
		t.Fatalf("ExtractCSS: %v", err)
	}
	data, err := os.ReadFile(CSSPath("1.0.0"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "stale{}" {
		t.Fatalf("ExtractCSS replaced the cache: %q", string(data))
	}

	if _, err := ExtractCSSForce("1.0.0"); err != nil {
		t.Fatalf("ExtractCSSForce: %v", err)
	}
	data, err = os.ReadFile(CSSPath("1.0.0"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) == "stale{}" {
		t.Error("ExtractCSSForce did not replace the cached file")
	}
}

func TestExtractCSSLeavesNoPartialFileOnFailure(t *testing.T) {
	useTempCSSDir(t)

	origURL := asarReleaseURL
	origClient := httpClient
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Valid gzip header, but the payload is not a usable ASAR archive.
		_, _ = w.Write([]byte("not an asar"))
	}))
	defer ts.Close()
	asarReleaseURL = ts.URL + "/v%s/obsidian-%s.asar.gz"
	httpClient = ts.Client()
	defer func() { asarReleaseURL = origURL; httpClient = origClient }()

	if _, err := ExtractCSS("1.0.0"); err == nil {
		t.Fatal("expected an extraction error")
	}
	if CSSCached("1.0.0") {
		t.Error("a failed extraction must not leave an app.css in the cache")
	}

	entries, err := os.ReadDir(filepath.Dir(CSSPath("1.0.0")))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temporary file %q was left behind", e.Name())
		}
	}
}
