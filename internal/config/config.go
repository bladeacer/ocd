// Package config provides OS-aware configuration file resolution for ocd.
//
// Config files are TOML files that let users override the default value of
// any command-line flag. Resolution order, from highest to lowest priority:
//
//  1. Direct command-line flag value.
//  2. Per-project (working directory) config file: `.ocd.toml` in the
//     current working directory.
//  3. Global config file: `$XDG_CONFIG_HOME/ocd/config.toml`, falling back
//     to `~/.config/ocd/config.toml` when `$XDG_CONFIG_HOME` is unset or
//     empty.
//
// Only fields that are explicitly set in a config file participate in the
// merge; unset fields keep their command default. This means a global config
// can set a default for a flag without overriding a per-project value, and a
// command-line flag always wins.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds the user-overridable defaults for ocd commands.
//
// Every field maps to a command-line flag. A zero value means "not set in
// config"; commands should treat that as "use the flag default".
type Config struct {
	// Diff command defaults.
	TLDR       *bool  `toml:"tldr,omitempty"`
	TLDRFormat string `toml:"tldr_format,omitempty"`
	TLDRDir    string `toml:"tldr_dir,omitempty"`
	Pick       *bool  `toml:"pick,omitempty"`
	Refresh    *bool  `toml:"refresh,omitempty"`

	// Stat command defaults.
	StatFormat string `toml:"stat_format,omitempty"`
	StatDir    string `toml:"stat_dir,omitempty"`

	// Common output defaults.
	OutputDir string `toml:"output_dir,omitempty"`

	// Check command defaults.
	CheckFormat string `toml:"check_format,omitempty"`
	CheckDir    string `toml:"check_dir,omitempty"`
}

// Resolve finds and loads the effective configuration for the current working
// directory. It merges the global config with the per-project config. When
// both are absent it returns an empty, non-nil *Config.
func Resolve() (*Config, error) {
	return ResolveIn(os.Getwd)
}

// ResolveIn is like Resolve but allows the working-directory lookup to be
// overridden (useful for tests).
func ResolveIn(getwd func() (string, error)) (*Config, error) {
	wd, err := getwd()
	if err != nil {
		return nil, fmt.Errorf("working directory: %w", err)
	}

	globalPath, err := globalConfigPath()
	if err != nil {
		return nil, err
	}

	localPath := filepath.Join(wd, ".ocd.toml")

	// Process files from lowest to highest priority so that later files
	// override earlier ones. Direct command-line flags are applied by the
	// caller after Resolve returns.
	return mergeFiles(globalPath, localPath)
}

// globalConfigPath returns the OS-specific location of the global config
// file. It honours `$XDG_CONFIG_HOME` and falls back to `~/.config/ocd`.
func globalConfigPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "ocd", "config.toml"), nil
}

func mergeFiles(paths ...string) (*Config, error) {
	out := &Config{}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat config %q: %w", p, err)
		}
		var c Config
		if _, err := toml.DecodeFile(p, &c); err != nil {
			return nil, fmt.Errorf("decode config %q: %w", p, err)
		}
		mergeInto(out, &c)
	}
	return out, nil
}

// mergeInto copies non-zero fields from src into dst. Pointer fields are
// copied by value so a later file can override an earlier one.
func mergeInto(dst, src *Config) {
	if src.TLDR != nil {
		v := *src.TLDR
		dst.TLDR = &v
	}
	if src.TLDRFormat != "" {
		dst.TLDRFormat = src.TLDRFormat
	}
	if src.TLDRDir != "" {
		dst.TLDRDir = src.TLDRDir
	}
	if src.Pick != nil {
		v := *src.Pick
		dst.Pick = &v
	}
	if src.Refresh != nil {
		v := *src.Refresh
		dst.Refresh = &v
	}
	if src.StatFormat != "" {
		dst.StatFormat = src.StatFormat
	}
	if src.StatDir != "" {
		dst.StatDir = src.StatDir
	}
	if src.OutputDir != "" {
		dst.OutputDir = src.OutputDir
	}
	if src.CheckFormat != "" {
		dst.CheckFormat = src.CheckFormat
	}
	if src.CheckDir != "" {
		dst.CheckDir = src.CheckDir
	}
}

// String returns a human-readable description of the config sources that
// were consulted, used for diagnostics.
func (c *Config) String() string {
	var b strings.Builder
	b.WriteString("config{")
	if c.TLDR != nil {
		fmt.Fprintf(&b, " tldr=%v", *c.TLDR)
	}
	if c.TLDRFormat != "" {
		fmt.Fprintf(&b, " tldr_format=%q", c.TLDRFormat)
	}
	if c.TLDRDir != "" {
		fmt.Fprintf(&b, " tldr_dir=%q", c.TLDRDir)
	}
	if c.Pick != nil {
		fmt.Fprintf(&b, " pick=%v", *c.Pick)
	}
	if c.Refresh != nil {
		fmt.Fprintf(&b, " refresh=%v", *c.Refresh)
	}
	if c.StatFormat != "" {
		fmt.Fprintf(&b, " stat_format=%q", c.StatFormat)
	}
	if c.StatDir != "" {
		fmt.Fprintf(&b, " stat_dir=%q", c.StatDir)
	}
	if c.OutputDir != "" {
		fmt.Fprintf(&b, " output_dir=%q", c.OutputDir)
	}
	if c.CheckFormat != "" {
		fmt.Fprintf(&b, " check_format=%q", c.CheckFormat)
	}
	if c.CheckDir != "" {
		fmt.Fprintf(&b, " check_dir=%q", c.CheckDir)
	}
	b.WriteString("}")
	return b.String()
}
