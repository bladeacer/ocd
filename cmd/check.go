package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/core"
)

func NewCheckCmd() *cobra.Command {
	var format string
	var output string
	var silent bool

	cmd := &cobra.Command{
		Use:   "check <version> <theme.css>",
		Short: "Check a theme's CSS variables against a target Obsidian version",
		Long: `Extract the CSS variables from a target Obsidian version and compare
them against a local theme file. The report lists variables that are missing
from the theme (present in the target) and variables that the theme defines
that are not in the target.

The report is printed to stdout by default. Use --output to also write it to
a file. Use --silent to suppress the on-screen report.

Examples:
  ocd check 1.12.7 ./my-theme.css
  ocd check 1.12.7 ./my-theme.css --format json --output ~/reports
  ocd check 1.12.7 ./my-theme.css --silent --output ~/reports`,
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
			themeVars, err := core.ReadCSSVariables(themePath)
			if err != nil {
				return err
			}

			report := core.CompareVariables(version, themePath, targetVars, themeVars)

			if !silent {
				fmt.Print(report.String())
			}

			if output != "" {
				fname := fmt.Sprintf("ocd-check-%s", version)
				exported, err := core.ExportFile(func() ([]byte, error) {
					return marshalReport(report, format)
				}, output, fname, format)
				if err != nil {
					return fmt.Errorf("export check: %w", err)
				}
				fmt.Printf("Exported: %s\n", exported)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "toml", "Export format: toml (default), json, or yaml")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory (supports ~, $HOME, $XDG_CONFIG_HOME)")
	cmd.Flags().BoolVarP(&silent, "silent", "s", false, "Suppress stdout report")
	return cmd
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
