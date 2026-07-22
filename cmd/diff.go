package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/sources"
	"github.com/bladeacer/ocd/internal/tui"
)

func NewDiffCmd() *cobra.Command {
	var forceRefresh bool
	var interactive bool

	cmd := &cobra.Command{
		Use:   "diff [version-a] [version-b]",
		Short: "Show CSS diff between two Obsidian versions",
		Long: `Display a unified diff of app.css between two Obsidian versions.
Both versions must have been extracted first via 'extract'.

If no arguments are provided, or --pick is used, an interactive
version picker is launched.`,
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

			result := core.DiffCSS(versionA, versionB)
			if result.Error != nil {
				return fmt.Errorf("diff: %w", result.Error)
			}

			return tui.RunDiffViewer(result)
		},
	}

	cmd.Flags().BoolVarP(&forceRefresh, "refresh", "r", false, "Force refresh metadata cache")
	cmd.Flags().BoolVarP(&interactive, "pick", "p", false, "Launch interactive version picker")
	return cmd
}
