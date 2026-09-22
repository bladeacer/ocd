// Package core provides the shared business logic for ocd: ASAR
// extraction, CSS diffing, TLDR analysis, and file export.
package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExportFile marshals data using the supplied marshal function and writes it
// to a file under dir. dir may be empty (uses the current working directory),
// an existing directory, or a path to a directory that does not yet exist (it
// is created). filename is the base name without an extension; the extension
// is derived from format (toml, json, or yaml).
//
// It returns the full path of the written file.
//
// This is a generalised helper shared by the stat, tldr, and check commands
// so that every command exports through one code path.
func ExportFile(marshal func() ([]byte, error), dir, filename, format string) (string, error) {
	dir = expandDir(dir)
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("working directory: %w", err)
		}
		dir = wd
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create output dir %q: %w", dir, err)
	}

	ext := formatToExtension(format)
	fullPath := filepath.Join(dir, filename+ext)
	if strings.TrimSpace(format) == "" {
		fullPath = filepath.Join(dir, filename)
	}

	data, err := marshal()
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", format, err)
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", fullPath, err)
	}
	return fullPath, nil
}

// expandDir expands a leading `~/` and environment variables in a directory
// path, mirroring the behaviour of the legacy expandPath helper.
func expandDir(dir string) string {
	dir = os.ExpandEnv(dir)
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			dir = filepath.Join(home, dir[2:])
		}
	}
	return dir
}

func formatToExtension(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return ".json"
	case "yaml", "yml":
		return ".yaml"
	case "toml":
		return ".toml"
	default:
		return ""
	}
}
