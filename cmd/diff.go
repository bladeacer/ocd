package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/sources"
	"github.com/bladeacer/ocd/internal/tui"
)

func printTLDR(t *core.TLDRResult, exportPath string) {
	fmt.Println(t.String())
	if exportPath != "" {
		fmt.Printf("Exported: %s\n", exportPath)
	}
}

func marshalTLDR(t *core.TLDRResult, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return t.MarshalJSON()
	case "yaml", "yml":
		return t.MarshalYAML()
	default:
		return t.MarshalTOML()
	}
}

func ensureCSS(version string) error {
	p := filepath.Join(".obsidian_cache", "css", version, "app.css")
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	fmt.Printf("Extracting app.css for v%s...\n", version)
	_, err := core.ExtractCSS(version)
	return err
}

func NewDiffCmd() *cobra.Command {
	var forceRefresh bool
	var interactive bool
	var tldr bool
	var tldrFormat string
	var tldrOutput string

	cmd := &cobra.Command{
		Use:   "diff [version-a] [version-b]",
		Short: "Show CSS diff between two Obsidian versions",
		Long: `Display a unified diff of app.css between two Obsidian versions.
Versions are auto-extracted if not already cached.

If no arguments are provided, or --pick is used, an interactive
version picker is launched.

Use --tldr to print a summary of CSS changes and export to file.

Configuration: defaults for the flags below may be set in a per-project
config file in the working directory, or in the global config at
$XDG_CONFIG_HOME/ocd/config.toml (falling back to ~/.config/ocd).
Direct command-line flags always take priority over config file values.`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgErr := config.Resolve()
			if cfgErr != nil {
				return cfgErr
			}
			// CLI flag overrides config overrides default.
			if !cmd.Flags().Changed("tldr") && cfg.TLDR != nil {
				tldr = *cfg.TLDR
			}
			if !cmd.Flags().Changed("tldr-format") && cfg.TLDRFormat != "" {
				tldrFormat = cfg.TLDRFormat
			}
			if !cmd.Flags().Changed("tldr-output") && cfg.TLDRDir != "" {
				tldrOutput = cfg.TLDRDir
			}
			if !cmd.Flags().Changed("pick") && cfg.Pick != nil {
				interactive = *cfg.Pick
			}
			if !cmd.Flags().Changed("refresh") && cfg.Refresh != nil {
				forceRefresh = *cfg.Refresh
			}

			var versionA, versionB string

			if len(args) == 2 {
				versionA = args[0]
				versionB = args[1]
			} else if interactive || len(args) == 0 {
				c, err := cache.New(0)
				if err != nil {
					return fmt.Errorf("cache init: %w", err)
				}
				f := sources.NewFetcher(c)

				versionA, versionB, err = tui.PickVersions(f, forceRefresh)
				if err != nil {
					return fmt.Errorf("picker: %w", err)
				}
				if versionA == "" || versionB == "" {
					fmt.Println("Selection cancelled.")
					return nil
				}
			} else {
				return fmt.Errorf("usage: diff <version-a> <version-b> or diff --pick")
			}

			if err := ensureCSS(versionA); err != nil {
				return fmt.Errorf("extract %s: %w", versionA, err)
			}
			if err := ensureCSS(versionB); err != nil {
				return fmt.Errorf("extract %s: %w", versionB, err)
			}

			result := core.DiffCSS(versionA, versionB)
			if result.Error != nil {
				return fmt.Errorf("diff: %w", result.Error)
			}

			if tldr {
				tldrResult := core.AnalyzeDiff(result.Diff)
				tldrResult.VersionA = versionA
				tldrResult.VersionB = versionB
				tldrResult.SemverBump = core.SemverBump(versionA, versionB)
				exportPath := tldrOutput
				if exportPath == "" {
					wd, wdErr := os.Getwd()
					if wdErr != nil {
						exportPath = "."
					} else {
						exportPath = wd
					}
				}
				fname := fmt.Sprintf("ocd-tldr-%s-%s", versionA, versionB)
				fullPath, err := core.ExportFile(func() ([]byte, error) {
					return marshalTLDR(tldrResult, tldrFormat)
				}, exportPath, fname, tldrFormat)
				if err != nil {
					return fmt.Errorf("export tldr: %w", err)
				}
				printTLDR(tldrResult, fullPath)
				return nil
			}

			return tui.RunDiffViewer(result)
		},
	}

	cmd.Flags().BoolVarP(&forceRefresh, "refresh", "r", false, "Force refresh metadata cache")
	cmd.Flags().BoolVarP(&interactive, "pick", "p", false, "Launch interactive version picker")
	cmd.Flags().BoolVar(&tldr, "tldr", false, "Print TLDR analysis and export to file")
	cmd.Flags().StringVar(&tldrFormat, "tldr-format", "toml", "Export format: toml (default), json, or yaml")
	cmd.Flags().StringVar(&tldrOutput, "tldr-output", "", "Output directory (supports ~, $HOME, $XDG_CONFIG_HOME)")
	return cmd
}
