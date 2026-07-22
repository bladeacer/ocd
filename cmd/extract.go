package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/core"
)

func NewExtractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract <version>",
		Short: "Download and extract app.css from an Obsidian release",
		Long: `Download the Obsidian ASAR bundle for a given version from GitHub releases
and extract app.css. Example: ocd extract 1.12.7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := args[0]
			path, err := core.ExtractCSS(version)
			if err != nil {
				return fmt.Errorf("extract v%s: %w", version, err)
			}
			fmt.Printf("Extracted app.css for v%s to: %s\n", version, path)
			return nil
		},
	}

	return cmd
}
