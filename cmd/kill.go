package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"time"

	"github.com/MrHalder/moor/internal/config"
	"github.com/MrHalder/moor/internal/formatter"
	"github.com/MrHalder/moor/internal/process"
	"github.com/MrHalder/moor/internal/scanner"
	"github.com/spf13/cobra"
)

var (
	killForce bool
	killYes   bool
)

var killCmd = &cobra.Command{
	Use:   "kill <port>",
	Short: "Kill the process using a port",
	Long:  "Send SIGTERM (or SIGKILL with -f) to the process occupying a port.",
	Args:  cobra.ExactArgs(1),
	RunE:  runKill,
}

func init() {
	killCmd.Flags().BoolVarP(&killForce, "force", "f", false, "force kill with SIGKILL (no grace period)")
	killCmd.Flags().BoolVarP(&killYes, "yes", "y", false, "skip confirmation prompt")
	rootCmd.AddCommand(killCmd)
}

func runKill(cmd *cobra.Command, args []string) error {
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

	// Find LISTEN connections first, fall back to any connection
	targets := filterByState(result.Ports, "LISTEN")
	if len(targets) == 0 {
		targets = result.Ports
	}

	if len(targets) == 0 {
		fmt.Printf("No process found on port %d\n", port)
		if result.NeedsElevation {
			fmt.Println("Try running with sudo for complete results.")
		}
		return nil
	}

	// Deduplicate by PID
	seen := make(map[int32]bool)
	unique := make([]scanner.PortInfo, 0)
	for _, t := range targets {
		if t.PID > 0 && !seen[t.PID] {
			seen[t.PID] = true
			unique = append(unique, t)
		}
	}

	if len(unique) == 0 {
		fmt.Printf("Found connections on port %d but could not identify process (try sudo)\n", port)
		return nil
	}

	mgr := process.NewManager()
	if cfg, err := config.Load(); err == nil {
		mgr.GracePeriod = time.Duration(cfg.Settings.GracePeriodSecs) * time.Second
	}

	for _, target := range unique {
		name := formatter.SanitizeDisplay(target.ProcessName)
		if name == "" {
			name = "unknown"
		}

		if !killYes {
			fmt.Printf("Kill process '%s' (PID %d) on port %d? [y/N] ", name, target.PID, port)
			reader := bufio.NewReader(os.Stdin)
			answer, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return fmt.Errorf("reading input: %w", err)
			}
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Skipped.")
				continue
			}
		}

		method := "SIGTERM"
		if killForce {
			method = "SIGKILL"
		}

		fmt.Printf("Sending %s to '%s' (PID %d)...\n", method, name, target.PID)

		if err := mgr.Kill(ctx, target.PID, killForce); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to kill PID %d: %v\n", target.PID, err)
			continue
		}

		fmt.Printf("Killed '%s' (PID %d) on port %d\n", name, target.PID, port)
	}

	return nil
}

func filterByState(ports []scanner.PortInfo, state string) []scanner.PortInfo {
	filtered := make([]scanner.PortInfo, 0)
	for _, p := range ports {
		if strings.EqualFold(p.State, state) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
