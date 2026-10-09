package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/sources"
)

func NewOriginCmd() *cobra.Command {
	var format string
	var output string
	var refresh bool
	var cacheDays int

	cmd := &cobra.Command{
		Use:   "origin <selector|variable>",
		Short: "Find the earliest public desktop Obsidian version where a selector or variable was introduced",
		Long: `Search app.css across public desktop Obsidian versions and report
the earliest version where the given selector or CSS variable was first
introduced.

The command caches app.css for every public desktop version on demand, so
the answer covers the whole public history instead of the few versions that
happen to be cached already. The first run downloads every release and takes
some time. Later runs reuse the cache.

Cached app.css is reused for 14 days. After that it is downloaded again. Use
--cache-days to change the age, --refresh to download everything again now,
or 'ocd clean <version>' to drop a single entry.

Targets starting with -- are treated as CSS variables; all others are
treated as selectors. When the target itself starts with --, use the --
separator to pass it as an argument:

  ocd origin -- --my-var       # variable mode (auto-detected)
  ocd origin .messageBar       # selector mode

Use --format to export the result to TOML, JSON, or YAML, and --output to
specify a target directory. File export defaults to the current working
directory.

Examples:
  ocd origin .messageBar
  ocd origin -- --my-var
  ocd origin .my-selector --format json --output ~/reports
  ocd origin .my-selector --refresh`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgErr := config.Resolve()
			if cfgErr != nil {
				return cfgErr
			}
			if !cmd.Flags().Changed("format") && cfg.OriginFormat != "" {
				format = cfg.OriginFormat
			}
			if !cmd.Flags().Changed("output") && cfg.OriginDir != "" {
				output = cfg.OriginDir
			}
			if !cmd.Flags().Changed("cache-days") {
				cacheDays = cfg.CacheDaysOrDefault(core.DefaultCacheDays)
			}
			if format == "" {
				format = "toml"
			}

			target := args[0]
			isVariable := strings.HasPrefix(target, "--")

			c, err := cache.New(metadataTTL(cacheDays))
			if err != nil {
				return fmt.Errorf("cache init: %w", err)
			}

			f := sources.NewFetcher(c)
			fetchResult := f.FetchAll(refresh)

			var versions []string
			if len(fetchResult.RSS) > 0 {
				versions = core.PublicDesktopVersions(fetchResult.RSS)
			}
			cachedVersions, listErr := core.ListCachedVersions()
			if listErr == nil {
				versions = core.MergeVersions(versions, cachedVersions)
			}
			if len(versions) == 0 {
				return fmt.Errorf("no Obsidian versions found: %w", fetchResult.Error)
			}

			// Cache every version so the search covers the whole public
			// history, not only the versions that happen to be cached.
			sweep := core.EnsureAllCSS(core.SweepOptions{
				Versions:  versions,
				CacheDays: cacheDays,
				Force:     refresh,
				Log:       os.Stderr,
			})
			fmt.Fprintf(os.Stderr, "Cache: %s\n", sweep.String())

			originResult := core.FindOrigin(target, isVariable, versions)
			fmt.Print(originResult.String())

			exportDir := output
			if exportDir == "" {
				wd, wdErr := os.Getwd()
				if wdErr != nil {
					exportDir = "."
				} else {
					exportDir = wd
				}
			}
			fname := fmt.Sprintf("ocd-origin-%s", sanitizeFilename(target))
			fullPath, err := core.ExportFile(func() ([]byte, error) {
				return marshalOrigin(originResult, format)
			}, exportDir, fname, format)
			if err != nil {
				return fmt.Errorf("export origin: %w", err)
			}
			fmt.Printf("Exported: %s\n", fullPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "toml", "Export format: toml (default), json, or yaml")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory (supports ~, $HOME, $XDG_CONFIG_HOME). Defaults to cwd.")
	cmd.Flags().BoolVarP(&refresh, "refresh", "r", false, "Force refresh metadata cache and download every app.css again")
	cmd.Flags().IntVar(&cacheDays, "cache-days", core.DefaultCacheDays, "Days a cached app.css stays fresh before it is downloaded again. 0 disables expiry.")
	return cmd
}

// metadataTTL turns the cache age in days into a lifetime for the version
// metadata cache. Zero or less disables expiry for that cache too.
func metadataTTL(cacheDays int) time.Duration {
	if cacheDays <= 0 {
		return 0
	}
	return time.Duration(cacheDays) * 24 * time.Hour
}

func marshalOrigin(r *core.OriginResult, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return r.MarshalJSON()
	case "yaml", "yml":
		return r.MarshalYAML()
	default:
		return r.MarshalTOML()
	}
}

func sanitizeFilename(s string) string {
	s = strings.TrimLeft(s, ".-#")
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteRune('-')
		}
	}
	result := b.String()
	if result == "" {
		result = "untitled"
	}
	return result
}
