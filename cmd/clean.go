package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
	"github.com/bladeacer/ocd/internal/core"
)

func NewCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [label]",
		Short: "Wipe cached metadata and extracted CSS files",
		Long: `Remove cached data. With a label argument, remove only that
version's extracted CSS. Without arguments, wipe everything.

Examples:
  ocd clean               # wipe all caches and extracted CSS
  ocd clean my-insider    # remove only the "my-insider" import`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				label := args[0]
				dir := filepath.Join(core.CSSDir, label)
				if err := os.RemoveAll(dir); err != nil {
					return fmt.Errorf("remove %s: %w", dir, err)
				}
				fmt.Printf("Removed cached CSS for %q\n", label)
				return nil
			}

			c, err := cache.New(0)
			if err != nil {
				return fmt.Errorf("cache init: %w", err)
			}
			if err := c.Clear(); err != nil {
				return fmt.Errorf("clear cache: %w", err)
			}
			if err := os.RemoveAll(".obsidian_cache/css"); err != nil {
				return fmt.Errorf("remove css dir: %w", err)
			}
			fmt.Println("Cache and extracted CSS cleared.")
			return nil
		},
	}

	return cmd
}
