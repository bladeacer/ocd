package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bladeacer/obsi-css-diff/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:   "obsi-css-diff",
		Short: "Track Obsidian versions, extract app.css, and diff CSS changes",
		Long: `obsi-css-diff is a TUI tool for tracking Obsidian versions,
extracting app.css from Obsidian releases on GitHub,
and computing CSS diffs between versions.`,
		Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
	}

	root.AddCommand(cmd.NewInteractCmd())
	root.AddCommand(cmd.NewExtractCmd())
	root.AddCommand(cmd.NewDiffCmd())
	root.AddCommand(cmd.NewCleanCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
