package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/sources"
	"github.com/bladeacer/ocd/internal/tui"
)

func exportTLDR(t *core.TLDRResult, path, format string) error {
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = t.MarshalJSON()
	case "yaml":
		data, err = t.MarshalYAML()
	default:
		data, err = t.MarshalTOML()
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func printTLDR(t *core.TLDRResult) {
	fmt.Println(t.String())
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

	cmd := &cobra.Command{
		Use:   "diff [version-a] [version-b]",
		Short: "Show CSS diff between two Obsidian versions",
		Long: `Display a unified diff of app.css between two Obsidian versions.
Versions are auto-extracted if not already cached.

If no arguments are provided, or --pick is used, an interactive
version picker is launched.

Use --tldr to print a summary of CSS changes and export to file.`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var versionA, versionB string

			if len(args) == 2 {
				versionA = args[0]
				versionB = args[1]
			} else if interactive || len(args) == 0 {
				c := cache.New(0)
				f := sources.NewFetcher(c)

				var err error
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
				printTLDR(tldrResult)
				fname := fmt.Sprintf("ocd-tldr-%s-%s.%s", versionA, versionB, tldrFormat)
				fname = strings.ReplaceAll(fname, ".", "_")
				if err := exportTLDR(tldrResult, fname, tldrFormat); err != nil {
					return fmt.Errorf("export tldr: %w", err)
				}
				fmt.Printf("TLDR exported to %s\n", fname)
				return nil
			}

			return tui.RunDiffViewer(result)
		},
	}

	cmd.Flags().BoolVarP(&forceRefresh, "refresh", "r", false, "Force refresh metadata cache")
	cmd.Flags().BoolVarP(&interactive, "pick", "p", false, "Launch interactive version picker")
	cmd.Flags().BoolVar(&tldr, "tldr", false, "Print TLDR analysis and export to file")
	cmd.Flags().StringVar(&tldrFormat, "tldr-format", "toml", "Export format: toml (default), json, or yaml")
	return cmd
}
