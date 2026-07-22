package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/internal/cache"
)

func NewCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Wipe all cached metadata and extracted CSS files",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := cache.New(0)
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
