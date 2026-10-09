package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bladeacer/ocd/internal/core"
)

func useTempCSSDir(t *testing.T) {
	t.Helper()
	orig := core.CSSDir
	core.CSSDir = t.TempDir()
	t.Cleanup(func() { core.CSSDir = orig })
}

func seedCSS(t *testing.T, version, body string) {
	t.Helper()
	path := core.CSSPath(version)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestExtractReportsAlreadyCached(t *testing.T) {
	useTempCSSDir(t)
	seedCSS(t, "1.6.3", "body{}")

	cmd := NewExtractCmd()
	cmd.SetArgs([]string{"1.6.3"})
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stdout)

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute: %v", err)
		}
	})

	if !strings.Contains(out, "Already cached") {
		t.Errorf("expected an already cached message, got %q", out)
	}
	if !strings.Contains(out, "--refresh") {
		t.Errorf("expected a hint about --refresh, got %q", out)
	}
	if !strings.Contains(out, "ocd clean 1.6.3") {
		t.Errorf("expected a hint about ocd clean, got %q", out)
	}
}

func TestExtractRefreshFlagExists(t *testing.T) {
	c := NewExtractCmd()
	refresh := c.Flag("refresh")
	if refresh == nil {
		t.Fatal("expected --refresh flag")
	}
	if refresh.DefValue != "false" {
		t.Errorf("expected refresh default false, got %s", refresh.DefValue)
	}
}

func TestOriginCacheDaysFlag(t *testing.T) {
	c := NewOriginCmd()
	flag := c.Flag("cache-days")
	if flag == nil {
		t.Fatal("expected --cache-days flag")
	}
	if flag.DefValue != "14" {
		t.Errorf("expected cache-days default 14, got %s", flag.DefValue)
	}
}

func TestCheckCompatSweepFlag(t *testing.T) {
	c := NewCheckCmd()
	flag := c.Flag("compat-sweep")
	if flag == nil {
		t.Fatal("expected --compat-sweep flag")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected compat-sweep default false, got %s", flag.DefValue)
	}
	if c.Flag("cache-days") == nil {
		t.Fatal("expected --cache-days flag on check")
	}
}

func TestMetadataTTL(t *testing.T) {
	if got := metadataTTL(0); got != 0 {
		t.Errorf("metadataTTL(0) = %v, want 0", got)
	}
	if got := metadataTTL(-5); got != 0 {
		t.Errorf("metadataTTL(-5) = %v, want 0", got)
	}
	want := 14 * 24 * time.Hour
	if got := metadataTTL(14); got != want {
		t.Errorf("metadataTTL(14) = %v, want %v", got, want)
	}
}
