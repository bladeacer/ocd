package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/sources"
)

func NewCheckCmd() *cobra.Command {
	var format string
	var output string
	var silent bool
	var compatMode string
	var compatSweep bool
	var cacheDays int

	cmd := &cobra.Command{
		Use:   "check <version> <theme.css>",
		Short: "Check a theme's CSS variables against a target Obsidian version",
		Long: `Extract the CSS variables from a target Obsidian version and compare
them against a local theme. The report lists variables that are missing
from the theme (present in the target) and variables that the theme defines
that are not in the target.

The theme may be a single stylesheet or a folder. For a folder, the built CSS
is used when present, because that is what Obsidian loads. When the folder
holds no CSS, its SCSS sources are read instead, so a theme repository can be
checked directly. Only CSS custom properties are compared; Sass variables
such as $name are ignored, because Obsidian never sees them.

When --compat-mode is set (strict or relaxed), also run a compatibility
check on the theme's CSS variables against the target version. By default
the check searches the app.css versions already cached, so run 'ocd origin'
or use --compat-sweep to cache every public desktop version first. In strict
mode, an incompatible result yields a non-zero exit code.

The report is printed to stdout by default. Use --output to write it
to a specific directory. Use --silent to suppress the on-screen report.
File export defaults to the current working directory.

Examples:
  ocd check 1.12.7 ./my-theme.css
  ocd check 1.12.7 ~/.config/obsidian/themes/flexcyon   # a theme folder
  ocd check 1.12.7 ./my-theme.scss                       # SCSS sources
  ocd check 1.12.7 ./my-theme.css --format json --output ~/reports
  ocd check 1.12.7 ./my-theme.css --silent --output ~/reports
  ocd check 1.12.7 ./my-theme.css --compat-mode strict
  ocd check 1.12.7 ./my-theme.css --compat-mode strict --compat-sweep`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			themePath := args[1]

			cfg, cfgErr := config.Resolve()
			if cfgErr != nil {
				return cfgErr
			}

			// CLI flag overrides config overrides default.
			if format == "" && cfg.CheckFormat != "" {
				format = cfg.CheckFormat
			}
			if format == "" {
				format = "toml"
			}
			if output == "" {
				output = cfg.CheckDir
			}
			if compatMode == "" && cfg.CheckCompatMode != "" {
				compatMode = cfg.CheckCompatMode
			}
			if !cmd.Flags().Changed("cache-days") {
				cacheDays = cfg.CacheDaysOrDefault(core.DefaultCacheDays)
			}

			if _, err := os.Stat(themePath); err != nil {
				return fmt.Errorf("theme %q: %w", themePath, err)
			}

			path, err := core.ExtractCSS(version)
			if err != nil {
				return fmt.Errorf("extract v%s: %w", version, err)
			}
			css, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read app.css for %s: %w", version, err)
			}

			targetVars := core.ExtractCSSVariables(string(css))
			theme, err := core.ReadTheme(themePath)
			if err != nil {
				return fmt.Errorf("read theme %q: %w", themePath, err)
			}
			if len(theme.Vars) == 0 {
				return fmt.Errorf("no CSS custom properties found in %q: give a .css file, a folder holding theme.css, or a folder of SCSS sources", themePath)
			}
			themeVars := theme.Vars
			if !silent {
				fmt.Fprintf(os.Stderr, "Theme: %d variables from %d %s file(s) in %s\n",
					len(themeVars), len(theme.Files), themeKind(theme.Kind), themePath)
			}

			report := core.CompareVariables(version, themePath, targetVars, themeVars)

			cachedVersions, _ := core.ListCachedVersions()

			if compatMode != "" {
				compatVersions := cachedVersions
				if compatSweep {
					compatVersions = sweepAllVersions(cfg, cacheDays, false)
				}
				if len(compatVersions) > 0 {
					themeVarNames := core.VariableNames(themeVars)
					mode := core.CompatModeRelaxed
					if strings.EqualFold(compatMode, "strict") {
						mode = core.CompatModeStrict
					}
					compatReport := core.CheckCompatibility(themeVarNames, true, version, mode, compatVersions)
					if !silent {
						fmt.Print(compatReport.String())
					}
					if mode == core.CompatModeStrict && !compatReport.Compatible {
						return fmt.Errorf("compatibility check failed in strict mode")
					}
				} else if compatSweep {
					return fmt.Errorf("no cached versions available for the compatibility check")
				}
			}

			if !silent {
				fmt.Print(report.String())
			}

			// Default to cwd if no output dir specified.
			if output == "" {
				wd, wdErr := os.Getwd()
				if wdErr != nil {
					output = "."
				} else {
					output = wd
				}
			}

			fname := fmt.Sprintf("ocd-check-%s", version)
			exported, err := core.ExportFile(func() ([]byte, error) {
				return marshalReport(report, format)
			}, output, fname, format)
			if err != nil {
				return fmt.Errorf("export check: %w", err)
			}
			fmt.Printf("Exported: %s\n", exported)

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "toml", "Export format: toml (default), json, or yaml")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory (supports ~, $HOME, $XDG_CONFIG_HOME). Defaults to cwd.")
	cmd.Flags().BoolVarP(&silent, "silent", "s", false, "Suppress stdout report (file export still occurs)")
	cmd.Flags().StringVar(&compatMode, "compat-mode", "", "Compatibility mode: strict or relaxed")
	cmd.Flags().BoolVar(&compatSweep, "compat-sweep", false, "Cache app.css for every public desktop version before the compatibility check, so the check covers the whole public history")
	cmd.Flags().IntVar(&cacheDays, "cache-days", core.DefaultCacheDays, "Days a cached app.css stays fresh before it is downloaded again. 0 disables expiry.")
	return cmd
}

// themeKind names the kind of source a theme was read from, for output.
func themeKind(kind string) string {
	switch kind {
	case "css":
		return "css"
	case "scss":
		return "scss"
	default:
		return "source"
	}
}

// sweepAllVersions fetches the public desktop version list and makes sure
// app.css is cached for every version. It returns the versions that can be
// searched after the sweep.
func sweepAllVersions(cfg *config.Config, cacheDays int, force bool) []string {
	c, err := cache.New(metadataTTL(cacheDays))
	if err != nil {
		fmt.Fprintf(os.Stderr, "cache init: %v\n", err)
		return nil
	}

	fetchResult := sources.NewFetcher(c).FetchAll(force)

	var versions []string
	if len(fetchResult.RSS) > 0 {
		versions = core.PublicDesktopVersions(fetchResult.RSS)
	}
	if cached, listErr := core.ListCachedVersions(); listErr == nil {
		versions = core.MergeVersions(versions, cached)
	}
	if len(versions) == 0 {
		return nil
	}

	sweep := core.EnsureAllCSS(core.SweepOptions{
		Versions:  versions,
		CacheDays: cacheDays,
		Force:     force,
		Log:       os.Stderr,
	})
	fmt.Fprintf(os.Stderr, "Cache: %s\n", sweep.String())

	searchable, listErr := core.ListCachedVersions()
	if listErr != nil {
		return nil
	}
	return searchable
}

func marshalReport(r *core.VariableReport, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return r.MarshalJSON()
	case "yaml", "yml":
		return r.MarshalYAML()
	default:
		return r.MarshalTOML()
	}
}
