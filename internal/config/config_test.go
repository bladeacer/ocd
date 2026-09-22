package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveNoFiles(t *testing.T) {
	cfg, err := ResolveIn(func() (string, error) {
		return t.TempDir(), nil
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.TLDR != nil {
		t.Error("expected nil TLDR with no config files")
	}
}

func TestResolveGlobalOnly(t *testing.T) {
	wd := t.TempDir()
	cfgDir := filepath.Join(t.TempDir(), "ocd")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(`tldr = true
tldr_format = "json"
tldr_dir = "~/reports"
pick = false
refresh = true
stat_format = "yaml"
stat_dir = "~/stats"
output_dir = "~/out"
check_format = "toml"
check_dir = "~/check"
`), 0644); err != nil {
		t.Fatal(err)
	}

	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)
	os.Setenv("XDG_CONFIG_HOME", filepath.Dir(cfgDir))

	cfg, err := ResolveIn(func() (string, error) { return wd, nil })
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.TLDR == nil || !*cfg.TLDR {
		t.Errorf("expected tldr=true, got %v", cfg.TLDR)
	}
	if cfg.TLDRFormat != "json" {
		t.Errorf("expected tldr_format=json, got %q", cfg.TLDRFormat)
	}
	if cfg.TLDRDir != "~/reports" {
		t.Errorf("expected tldr_dir=~/reports, got %q", cfg.TLDRDir)
	}
	if cfg.Pick == nil || *cfg.Pick {
		t.Errorf("expected pick=false, got %v", cfg.Pick)
	}
	if cfg.Refresh == nil || !*cfg.Refresh {
		t.Errorf("expected refresh=true, got %v", cfg.Refresh)
	}
	if cfg.StatFormat != "yaml" {
		t.Errorf("expected stat_format=yaml, got %q", cfg.StatFormat)
	}
	if cfg.StatDir != "~/stats" {
		t.Errorf("expected stat_dir=~/stats, got %q", cfg.StatDir)
	}
	if cfg.OutputDir != "~/out" {
		t.Errorf("expected output_dir=~/out, got %q", cfg.OutputDir)
	}
	if cfg.CheckFormat != "toml" {
		t.Errorf("expected check_format=toml, got %q", cfg.CheckFormat)
	}
	if cfg.CheckDir != "~/check" {
		t.Errorf("expected check_dir=~/check, got %q", cfg.CheckDir)
	}
}

func TestResolveLocalOverridesGlobal(t *testing.T) {
	wd := t.TempDir()
	globalDir := filepath.Join(t.TempDir(), "ocd")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "config.toml"), []byte(`tldr = true
tldr_format = "json"
tldr_dir = "~/global"
stat_format = "yaml"
`), 0644); err != nil {
		t.Fatal(err)
	}
	localPath := filepath.Join(wd, ".ocd.toml")
	localContent := `tldr = false
tldr_format = "toml"
tldr_dir = "~/local"
`
	if err := os.WriteFile(localPath, []byte(localContent), 0644); err != nil {
		t.Fatal(err)
	}

	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)
	os.Setenv("XDG_CONFIG_HOME", filepath.Dir(globalDir))

	cfg, err := ResolveIn(func() (string, error) { return wd, nil })
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.TLDR == nil || *cfg.TLDR {
		t.Errorf("expected local tldr=false to override global, got %v", cfg.TLDR)
	}
	if cfg.TLDRFormat != "toml" {
		t.Errorf("expected local tldr_format=toml, got %q", cfg.TLDRFormat)
	}
	if cfg.TLDRDir != "~/local" {
		t.Errorf("expected local tldr_dir=~/local, got %q", cfg.TLDRDir)
	}
	// Fields not set locally fall back to global.
	if cfg.StatFormat != "yaml" {
		t.Errorf("expected global stat_format=yaml fallback, got %q", cfg.StatFormat)
	}
}

func TestResolveXDGHomeFallback(t *testing.T) {
	wd := t.TempDir()
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".config", "ocd")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := os.WriteFile(cfgPath, []byte(`pick = true`), 0644); err != nil {
		t.Fatal(err)
	}

	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()
	os.Setenv("XDG_CONFIG_HOME", "")
	os.Setenv("HOME", home)

	cfg, err := ResolveIn(func() (string, error) { return wd, nil })
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.Pick == nil || !*cfg.Pick {
		t.Errorf("expected pick=true from ~/.config fallback, got %v", cfg.Pick)
	}
}

func TestResolveBadGlobal(t *testing.T) {
	wd := t.TempDir()
	orig := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", orig)
	os.Setenv("XDG_CONFIG_HOME", "/nonexistent-dir-ocd-test")

	_, err := ResolveIn(func() (string, error) { return wd, nil })
	if err != nil {
		t.Fatalf("Resolve should not error on missing global: %v", err)
	}
}

func TestConfigString(t *testing.T) {
	cfg := &Config{}
	cfg.TLDR = boolPtr(true)
	cfg.TLDRFormat = "json"
	s := cfg.String()
	if s == "" {
		t.Error("expected non-empty String()")
	}
}

func boolPtr(b bool) *bool { return &b }
