package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStringShowsEverySetField(t *testing.T) {
	yes := true
	days := 7
	keys := DiffKeys{PrevHunk: []string{"p"}}

	c := &Config{
		TLDR:            &yes,
		TLDRFormat:      "json",
		TLDRDir:         "~/t",
		Pick:            &yes,
		Refresh:         &yes,
		DiffKeys:        keys,
		StatFormat:      "yaml",
		StatDir:         "~/s",
		OriginFormat:    "json",
		OriginDir:       "~/o",
		OutputDir:       "~/out",
		CheckFormat:     "toml",
		CheckDir:        "~/c",
		CheckCompatMode: "strict",
		CacheDays:       &days,
	}

	got := c.String()
	for _, want := range []string{
		"tldr=true", `tldr_format="json"`, "tldr_dir=", "pick=true", "refresh=true",
		"diff_keys=", "stat_format=", "stat_dir=", "origin_format=", "origin_dir=",
		"output_dir=", "check_format=", "check_dir=", "check_compat_mode=",
		"cache_days=7",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("String() missing %q, got %s", want, got)
		}
	}
}

func TestStringWithNothingSet(t *testing.T) {
	if got := (&Config{}).String(); got != "config{}" {
		t.Errorf("String() = %q, want config{}", got)
	}
}

func TestMergeFilesSkipsMissingPaths(t *testing.T) {
	cfg, err := mergeFiles("", filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("mergeFiles: %v", err)
	}
	if cfg == nil {
		t.Fatal("mergeFiles must return a config")
	}
}

func TestMergeFilesReadsEveryFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.toml")
	b := filepath.Join(dir, "b.toml")
	if err := os.WriteFile(a, []byte("cache_days = 3\nstat_format = \"json\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("cache_days = 9\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := mergeFiles(a, b)
	if err != nil {
		t.Fatalf("mergeFiles: %v", err)
	}
	if cfg.CacheDaysOrDefault(14) != 9 {
		t.Errorf("later file should win, got %d", cfg.CacheDaysOrDefault(14))
	}
	if cfg.StatFormat != "json" {
		t.Errorf("stat_format = %q", cfg.StatFormat)
	}
}

func TestMergeFilesBadTOML(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(bad, []byte("this is not = = toml"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := mergeFiles(bad); err == nil {
		t.Error("expected a decode error")
	}
}

func TestResolveInGetwdError(t *testing.T) {
	if _, err := ResolveIn(func() (string, error) { return "", os.ErrPermission }); err == nil {
		t.Error("expected an error when the working directory cannot be read")
	}
}

func TestResolveInMergesGlobalAndLocal(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, "ocd")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte("cache_days = 2\nstat_dir = \"~/global\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	wd := t.TempDir()
	if err := os.WriteFile(filepath.Join(wd, ".ocd.toml"), []byte("cache_days = 5\n"), 0644); err != nil {
		t.Fatal(err)
	}

	orig := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", home)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", orig) }()

	cfg, err := ResolveIn(func() (string, error) { return wd, nil })
	if err != nil {
		t.Fatalf("ResolveIn: %v", err)
	}
	if got := cfg.CacheDaysOrDefault(14); got != 5 {
		t.Errorf("cache_days = %d, want the local value 5", got)
	}
	if cfg.StatDir != "~/global" {
		t.Errorf("stat_dir = %q, want the global value to survive", cfg.StatDir)
	}
}
