package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bladeacer/obsi-css-diff/internal/cache"
	"github.com/bladeacer/obsi-css-diff/internal/core"
	"github.com/bladeacer/obsi-css-diff/internal/sources"
	"github.com/bladeacer/obsi-css-diff/internal/tui"
)

func NewInteractCmd() *cobra.Command {
	var forceRefresh bool

	cmd := &cobra.Command{
		Use:   "interact",
		Short: "Launch the interactive TUI to browse and select Obsidian versions",
		Long: `Launch an interactive terminal UI to browse Obsidian versions,
filter by type (desktop/mobile), search, and select a version
for CSS extraction and diffing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := cache.New(0)
			f := sources.NewFetcher(c)

			result := f.FetchAll(forceRefresh)
			if result.Error != nil {
				return fmt.Errorf("fetch data: %w", result.Error)
			}

			app := tui.New(result)
			selected, err := app.Run()
			if err != nil {
				return fmt.Errorf("tui error: %w", err)
			}
			if selected.Version == "" {
				fmt.Println("Selection cancelled.")
				return nil
			}

			fmt.Printf("Extracting app.css for v%s...\n", selected.Version)
			path, err := core.ExtractCSS(selected.Version)
			if err != nil {
				return fmt.Errorf("extract v%s: %w", selected.Version, err)
			}
			fmt.Printf("Saved to: %s\n", path)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&forceRefresh, "refresh", "r", false, "Force refresh metadata cache")
	return cmd
}
