package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/core"
)

func NewStatCmd() *cobra.Command {
	var format string
	var output string

	cmd := &cobra.Command{
		Use:   "stat <version>",
		Short: "Show CSS composition stats for a single version",
		Long: `Analyze a single version's app.css and print statistics:
selectors, CSS variables, color usage, etc.

Results are printed to stdout and optionally exported as
TOML, JSON, or YAML.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]

			css, err := core.ExtractCSS(version)
			if err != nil {
				return fmt.Errorf("extract %s: %w", version, err)
			}

			result := core.AnalyzeCSS(css)
			result.VersionA = version

			fmt.Print(result.String())

			exportPath := output
			if exportPath == "" {
				exportPath, _ = os.Getwd()
			} else {
				exportPath = expandPath(exportPath)
			}
			fname := fmt.Sprintf("ocd-stat-%s.%s", version, format)
			fullPath := filepath.Join(exportPath, fname)
			if err := exportTLDR(result, fullPath, format); err != nil {
				return fmt.Errorf("export stat: %w", err)
			}
			fmt.Printf("Exported: %s\n", fullPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "toml", "Export format: toml (default), json, or yaml")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory (supports ~, $HOME, $XDG_CONFIG_HOME)")
	return cmd
}
