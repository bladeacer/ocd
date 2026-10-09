package core

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var CSSDir = ".obsidian_cache/css"

var asarReleaseURL = "https://github.com/obsidianmd/obsidian-releases/releases/download/v%s/obsidian-%s.asar.gz"

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ErrNoRelease reports that GitHub has no ASAR release for a version.
// Callers use errors.Is to tell a missing release apart from a network
// failure, so a sweep can skip such versions instead of retrying them.
var ErrNoRelease = errors.New("no asar release")

func ImportFile(srcPath, label string) (string, error) {
	destDir := filepath.Join(CSSDir, label)
	destFile := filepath.Join(destDir, "app.css")

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	switch ext {
	case ".asar":
		if err := extractCSSAtomic(srcPath, destFile); err != nil {
			return "", fmt.Errorf("extract asar %q: %w", srcPath, err)
		}
	case ".css":
		if err := copyFileAtomic(srcPath, destFile); err != nil {
			return "", fmt.Errorf("copy css %q: %w", srcPath, err)
		}
	default:
		return "", fmt.Errorf("unsupported file extension %q (expected .asar or .css)", ext)
	}

	return destFile, nil
}

func ExtractCSS(version string) (string, error) {
	return extractCSS(version, false)
}

// ExtractCSSForce downloads and extracts app.css even when a cached copy
// already exists. Use it to renew a cache entry that has expired.
func ExtractCSSForce(version string) (string, error) {
	return extractCSS(version, true)
}

// extractCSS downloads the ASAR bundle for version and writes app.css into the
// CSS cache. Unless force is set, an existing cached copy is returned as is.
func extractCSS(version string, force bool) (string, error) {
	destDir := filepath.Join(CSSDir, version)
	destFile := filepath.Join(destDir, "app.css")

	if !force {
		if _, err := os.Stat(destFile); err == nil {
			return destFile, nil
		}
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	url := fmt.Sprintf(asarReleaseURL, version, version)

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("download asar.gz for v%s: %w", version, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no asar release for v%s (not found on GitHub): %w", version, ErrNoRelease)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download asar.gz for v%s: HTTP %d", version, resp.StatusCode)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("decompress asar.gz for v%s: %w", version, err)
	}
	defer func() { _ = gzReader.Close() }()

	tmpAsar, err := os.CreateTemp("", "obsidian-*.asar")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpAsar.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := io.Copy(tmpAsar, gzReader); err != nil {
		_ = tmpAsar.Close()
		return "", fmt.Errorf("write asar for v%s: %w", version, err)
	}
	_ = tmpAsar.Close()

	if err := extractCSSAtomic(tmpPath, destFile); err != nil {
		return "", fmt.Errorf("extract app.css for v%s: %w", version, err)
	}

	return destFile, nil
}

// CSSPath returns the cache path of app.css for a version.
func CSSPath(version string) string {
	return filepath.Join(CSSDir, version, "app.css")
}

// CSSCached reports whether app.css for a version is present in the cache.
func CSSCached(version string) bool {
	_, err := os.Stat(CSSPath(version))
	return err == nil
}

// CSSModTime returns the time when the cached app.css was last written.
// Callers use it to decide whether a cache entry has expired.
func CSSModTime(version string) (time.Time, error) {
	info, err := os.Stat(CSSPath(version))
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// CSSFresh reports whether the cached app.css for a version is still fresh.
// A ttl of zero or less means entries never expire.
func CSSFresh(version string, ttl time.Duration) bool {
	if ttl <= 0 {
		return CSSCached(version)
	}
	mod, err := CSSModTime(version)
	if err != nil {
		return false
	}
	return time.Since(mod) <= ttl
}

// extractCSSAtomic writes app.css to a temporary file in the destination
// directory and renames it into place. A failed or interrupted extraction
// therefore never leaves a partial app.css in the cache.
func extractCSSAtomic(asarPath, destPath string) error {
	tmpPath, cleanup, err := tempPathIn(destPath)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := extractAppCSSFromASAR(asarPath, tmpPath); err != nil {
		return err
	}
	return os.Rename(tmpPath, destPath)
}

// copyFileAtomic copies src to dst through a temporary file in the
// destination directory, then renames it into place.
func copyFileAtomic(src, dst string) error {
	tmpPath, cleanup, err := tempPathIn(dst)
	if err != nil {
		return err
	}
	defer cleanup()

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	d, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		_ = d.Close()
		return err
	}
	if err := d.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, dst)
}

// tempPathIn creates a temporary file next to dest and returns its path with
// a cleanup function that removes it.
func tempPathIn(dest string) (string, func(), error) {
	f, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".tmp-*")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp file: %w", err)
	}
	path := f.Name()
	if err := f.Close(); err != nil {
		return "", func() { _ = os.Remove(path) }, fmt.Errorf("close temp file: %w", err)
	}
	return path, func() { _ = os.Remove(path) }, nil
}
