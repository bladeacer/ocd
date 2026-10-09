package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/config"
	"github.com/bladeacer/ocd/internal/core"
	"github.com/bladeacer/ocd/internal/sources"
	"github.com/bladeacer/ocd/internal/tui"
)

// runInteract launches the interactive browser. Tests replace it so the
// command can be exercised without a terminal.
var runInteract = func(f *sources.Fetcher, force bool) (tui.Selection, error) {
	return tui.New(f, force).Run()
}

func NewInteractCmd() *cobra.Command {
	var forceRefresh bool

	cmd := &cobra.Command{
		Use:   "interact",
		Short: "Launch the interactive TUI to browse and select Obsidian versions",
		Long: `Launch an interactive terminal UI to browse Obsidian versions,
filter by type (desktop/mobile), search, and select a version
for CSS extraction and diffing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, cfgErr := config.Resolve()
			if cfgErr != nil {
				return cfgErr
			}
			c, err := cache.New(metadataTTL(cfg.CacheDaysOrDefault(core.DefaultCacheDays)))
			if err != nil {
				return fmt.Errorf("cache init: %w", err)
			}
			f := sources.NewFetcher(c)

			selected, err := runInteract(f, forceRefresh)
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
