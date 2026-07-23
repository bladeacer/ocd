package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bladeacer/ocd/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:           "ocd",
		Short:         "Track Obsidian versions, extract app.css, and diff CSS changes",
		Long:          `ocd is a TUI tool for tracking Obsidian versions,
extracting app.css from Obsidian releases on GitHub,
and computing CSS diffs between versions.`,
		Version:       fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		SilenceErrors: true,
		SilenceUsage:  true,
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
