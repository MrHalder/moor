package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	jsonOutput bool
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:   "moor",
	Short: "Moor your ports — terminal port management for macOS",
	Long:  "moor is a CLI + TUI tool for managing network ports, processes, and reservations on macOS.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If stdout is a terminal and no JSON flag, launch TUI
		if term.IsTerminal(int(os.Stdout.Fd())) && !jsonOutput {
			return runWatch(cmd, args)
		}
		// Piped or JSON mode: fall back to list
		return runList(cmd, args)
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable color output")
}

// Execute runs the root command.
func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}
	return nil
}
