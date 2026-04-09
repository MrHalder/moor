package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ashutosh/moor/internal/formatter"
	"github.com/ashutosh/moor/internal/scanner"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	listAll   bool
	listProto string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List listening ports",
	Long:  "Show all ports currently in use with process info.",
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "include non-LISTEN states (ESTABLISHED, TIME_WAIT, etc.)")
	listCmd.Flags().StringVar(&listProto, "proto", "", "filter by protocol (tcp or udp)")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Interactive TUI if terminal and no JSON/piped output
	if !jsonOutput && term.IsTerminal(int(os.Stdout.Fd())) {
		return runWatch(cmd, args)
	}

	return runListStatic(cmd, args)
}

// runListStatic outputs the static table (piped / JSON mode).
func runListStatic(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	s := scanner.NewScanner()

	var result *scanner.ScanResult
	var err error

	if listAll {
		result, err = s.ListAll(ctx)
	} else {
		result, err = s.ListListening(ctx)
	}
	if err != nil {
		return fmt.Errorf("scanning ports: %w", err)
	}

	// Filter by protocol if specified
	if listProto != "" {
		filtered := make([]scanner.PortInfo, 0)
		for _, p := range result.Ports {
			if strings.EqualFold(p.Protocol, listProto) {
				filtered = append(filtered, p)
			}
		}
		result = &scanner.ScanResult{
			Ports:          filtered,
			NeedsElevation: result.NeedsElevation,
		}
	}

	if jsonOutput {
		out, err := formatter.FormatJSON(result)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	fmt.Print(formatter.FormatTable(result, noColor))
	return nil
}
