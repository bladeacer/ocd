package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bladeacer/obsi-css-diff/internal/core"
)

func NewDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <version-a> <version-b>",
		Short: "Show CSS diff between two Obsidian versions",
		Long: `Display a unified diff of app.css between two Obsidian versions.
Both versions must have been extracted first via 'extract'.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			versionA := args[0]
			versionB := args[1]

			result := core.DiffCSS(versionA, versionB)
			if result.Error != nil {
				return fmt.Errorf("diff: %w", result.Error)
			}

			fmt.Print(result.Diff)
			return nil
		},
	}

	return cmd
}
