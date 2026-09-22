package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/config"
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
TOML, JSON, or YAML to the current working directory by default.
Use --output to specify a target directory.

Configuration: --format and --output defaults may be set in a per-project
config file in the working directory, or in the global config at
$XDG_CONFIG_HOME/ocd/config.toml (falling back to ~/.config/ocd).
Direct command-line flags always take priority over config file values.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgErr := config.Resolve()
			if cfgErr != nil {
				return cfgErr
			}
			// CLI flag overrides config overrides default.
			if !cmd.Flags().Changed("format") && cfg.StatFormat != "" {
				format = cfg.StatFormat
			}
			if !cmd.Flags().Changed("output") && cfg.StatDir != "" {
				output = cfg.StatDir
			}

			version := args[0]

			path, err := core.ExtractCSS(version)
			if err != nil {
				return fmt.Errorf("extract %s: %w", version, err)
			}
			css, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read app.css for %s: %w", version, err)
			}

			result := core.AnalyzeCSS(string(css))
			result.VersionA = version

			fmt.Print(result.StatString())

			exportPath := output
			if exportPath == "" {
				wd, wdErr := os.Getwd()
				if wdErr != nil {
					exportPath = "."
				} else {
					exportPath = wd
				}
			}
			fname := fmt.Sprintf("ocd-stat-%s", version)
			fullPath, err := core.ExportFile(func() ([]byte, error) {
				return marshalStat(result, format)
			}, exportPath, fname, format)
			if err != nil {
				return fmt.Errorf("export stat: %w", err)
			}
			fmt.Printf("\nExported: %s\n", fullPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "toml", "Export format: toml (default), json, or yaml")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory (supports ~, $HOME, $XDG_CONFIG_HOME). Defaults to cwd.")
	return cmd
}

func marshalStat(t *core.TLDRResult, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return t.MarshalJSON()
	case "yaml", "yml":
		return t.MarshalYAML()
	default:
		return t.MarshalTOML()
	}
}
