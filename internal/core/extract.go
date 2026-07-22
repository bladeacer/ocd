package core

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var extractCSSDir = ".obsidian_cache/css"

const asarReleaseURL = "https://github.com/obsidianmd/obsidian-releases/releases/download/v%s/obsidian-%s.asar.gz"

var httpClient = &http.Client{Timeout: 30 * time.Second}

func ExtractCSS(version string) (string, error) {
	destDir := filepath.Join(extractCSSDir, version)
	destFile := filepath.Join(destDir, "app.css")

	if _, err := os.Stat(destFile); err == nil {
		return destFile, nil
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create cache dir: %w", err)
	}

	url := fmt.Sprintf(asarReleaseURL, version, version)

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("download asar.gz for v%s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no asar release for v%s (not found on GitHub)", version)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download asar.gz for v%s: HTTP %d", version, resp.StatusCode)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("decompress asar.gz for v%s: %w", version, err)
	}
	defer gzReader.Close()

	tmpAsar, err := os.CreateTemp("", "obsidian-*.asar")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpAsar.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpAsar, gzReader); err != nil {
		tmpAsar.Close()
		return "", fmt.Errorf("write asar for v%s: %w", version, err)
	}
	tmpAsar.Close()

	if err := extractAppCSSFromASAR(tmpPath, destFile); err != nil {
		return "", fmt.Errorf("extract app.css for v%s: %w", version, err)
	}

	return destFile, nil
}
