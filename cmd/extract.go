package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/core"
)

func NewExtractCmd() *cobra.Command {
	var fromFile string

	cmd := &cobra.Command{
		Use:   "extract <version|label>",
		Short: "Download and extract app.css from an Obsidian release, or import from a local file",
		Long: `Download the Obsidian ASAR bundle for a given version from GitHub releases
and extract app.css, or import a local .asar/.css file with --from-file.

Usage:
  ocd extract 1.12.7                          # download from GitHub
  ocd extract --from-file ./obsidian.asar      # import local asar (label from filename)
  ocd extract --from-file ./obsidian.asar my-label  # import with explicit label
  ocd extract --from-file ./custom.css my-theme     # import local css`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromFile != "" {
				label := ""
				if len(args) == 1 {
					label = args[0]
				} else {
					label = strings.TrimSuffix(filepath.Base(fromFile), filepath.Ext(fromFile))
				}
				path, err := core.ImportFile(fromFile, label)
				if err != nil {
					return fmt.Errorf("import %q: %w", fromFile, err)
				}
				fmt.Printf("Imported app.css from %q to: %s\n", fromFile, path)
				return nil
			}

			if len(args) != 1 {
				return fmt.Errorf("usage: extract <version> or extract --from-file <path> [label]")
			}
			version := args[0]
			path, err := core.ExtractCSS(version)
			if err != nil {
				return fmt.Errorf("extract v%s: %w", version, err)
			}
			fmt.Printf("Extracted app.css for v%s to: %s\n", version, path)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromFile, "from-file", "", "Import a local .asar or .css file instead of downloading from GitHub")
	return cmd
}
