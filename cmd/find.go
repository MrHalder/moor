package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/MrHalder/moor/internal/formatter"
	"github.com/MrHalder/moor/internal/scanner"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find <port>",
	Short: "Find what process is using a port",
	Args:  cobra.ExactArgs(1),
	RunE:  runFind,
}

func init() {
	rootCmd.AddCommand(findCmd)
}

func runFind(cmd *cobra.Command, args []string) error {
	port, err := parsePort(args[0])
	if err != nil {
		return err
	}

	ctx := context.Background()
	s := scanner.NewScanner()

	result, err := s.FindByPort(ctx, port)
	if err != nil {
		return fmt.Errorf("scanning port %d: %w", port, err)
	}

	if len(result.Ports) == 0 {
		fmt.Printf("No process found on port %d\n", port)
		if result.NeedsElevation {
			fmt.Println("Try running with sudo for complete results.")
		}
		return nil
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

func parsePort(s string) (uint16, error) {
	n, err := strconv.ParseUint(s, 10, 16)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("invalid port: %q (must be 1-65535)", s)
	}
	return uint16(n), nil
}
