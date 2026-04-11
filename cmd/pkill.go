package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MrHalder/moor/internal/config"
	"github.com/MrHalder/moor/internal/formatter"
	"github.com/MrHalder/moor/internal/process"
	"github.com/spf13/cobra"
)

// bulkKillThreshold is the number of processes above which "all" requires
// explicit confirmation, even when -y is set.
const bulkKillThreshold = 10

// pkillTimeout is the maximum time allowed for the entire pkill operation.
const pkillTimeout = 30 * time.Second

var (
	pkillForce bool
	pkillYes   bool
	pkillExact bool
	pkillFull  bool
)

var pkillCmd = &cobra.Command{
	Use:   "pkill [pattern]",
	Short: "Find and kill processes by name",
	Long: `List system processes, optionally filtered by name pattern, and interactively kill selected ones.

Without a pattern, all user processes are shown. With a pattern, only matching
processes are listed. Use --exact for exact name matching, or -F to match
against the full command line.

Examples:
  moor pkill              # list all processes, pick which to kill
  moor pkill node         # list processes matching "node"
  moor pkill -f node      # force kill (SIGKILL) matching processes
  moor pkill --exact node # exact name match only
  moor pkill -F server.js # match against full command line`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPkill,
}

func init() {
	pkillCmd.Flags().BoolVarP(&pkillForce, "force", "f", false, "force kill with SIGKILL (no grace period)")
	pkillCmd.Flags().BoolVarP(&pkillYes, "yes", "y", false, "skip confirmation prompt")
	pkillCmd.Flags().BoolVar(&pkillExact, "exact", false, "require exact process name match")
	pkillCmd.Flags().BoolVarP(&pkillFull, "full", "F", false, "match against full command line")
	rootCmd.AddCommand(pkillCmd)
}

func runPkill(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), pkillTimeout)
	defer cancel()

	lister := process.NewLister()
	reader := bufio.NewReader(os.Stdin)

	entries, err := lister.List(ctx)
	if err != nil {
		return fmt.Errorf("listing processes: %w", err)
	}

	// Filter if a pattern was provided
	if len(args) == 1 {
		pattern := args[0]
		if pkillFull {
			entries = process.FilterByCommandLine(entries, pattern)
		} else {
			entries = process.FilterByName(entries, pattern, pkillExact)
		}
	}

	if len(entries) == 0 {
		fmt.Println("No matching processes found.")
		return nil
	}

	// Display the process table
	fmt.Print(formatter.FormatProcessTable(entries, noColor))
	fmt.Println()

	// Ask user to select processes
	selected, err := promptSelection(reader, entries)
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		fmt.Println("No processes selected.")
		return nil
	}

	// Set up the process manager
	mgr := process.NewManager()
	if cfg, loadErr := config.Load(); loadErr == nil {
		mgr.GracePeriod = time.Duration(cfg.Settings.GracePeriodSecs) * time.Second
	}

	// Kill each selected process with individual confirmation
	for _, entry := range selected {
		name := formatter.SanitizeDisplay(entry.Name)
		if name == "" {
			name = "unknown"
		}

		// TOCTOU mitigation: verify the process identity before killing
		if info, infoErr := mgr.Info(entry.PID); infoErr != nil {
			fmt.Fprintf(os.Stderr, "PID %d is no longer running, skipping.\n", entry.PID)
			continue
		} else if info.Name != entry.Name {
			fmt.Fprintf(os.Stderr, "PID %d was recycled (was %q, now %q), skipping.\n",
				entry.PID, entry.Name, info.Name)
			continue
		}

		if !pkillYes {
			fmt.Printf("Kill process '%s' (PID %d)? [y/N] ", name, entry.PID)
			answer, readErr := reader.ReadString('\n')
			if readErr != nil && readErr != io.EOF {
				return fmt.Errorf("reading input: %w", readErr)
			}
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Println("Skipped.")
				continue
			}
		}

		method := "SIGTERM"
		if pkillForce {
			method = "SIGKILL"
		}

		fmt.Printf("Sending %s to '%s' (PID %d)...\n", method, name, entry.PID)

		if killErr := mgr.Kill(ctx, entry.PID, pkillForce); killErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to kill PID %d: %v\n", entry.PID, killErr)
			continue
		}

		fmt.Printf("Killed '%s' (PID %d)\n", name, entry.PID)
	}

	return nil
}

// promptSelection asks the user to enter process numbers from the displayed table.
// Returns the selected ProcessInfo slice.
func promptSelection(reader *bufio.Reader, entries []process.ProcessInfo) ([]process.ProcessInfo, error) {
	fmt.Print("Enter process numbers to kill (e.g., 1,3,5 or 'all'): ")
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	if strings.EqualFold(line, "all") {
		// Safety guard: require explicit confirmation for large bulk kills
		if len(entries) > bulkKillThreshold {
			fmt.Printf("Warning: this will target %d processes. Type 'confirm' to proceed: ", len(entries))
			confirm, confirmErr := reader.ReadString('\n')
			if confirmErr != nil && confirmErr != io.EOF {
				return nil, fmt.Errorf("reading confirmation: %w", confirmErr)
			}
			if strings.TrimSpace(strings.ToLower(confirm)) != "confirm" {
				fmt.Println("Cancelled.")
				return nil, nil
			}
		}
		return entries, nil
	}

	parts := strings.Split(line, ",")
	seen := make(map[int]bool)
	selected := make([]process.ProcessInfo, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		num, parseErr := strconv.Atoi(part)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Invalid number: %s (skipping)\n", part)
			continue
		}

		idx := num - 1 // table is 1-indexed
		if idx < 0 || idx >= len(entries) {
			fmt.Fprintf(os.Stderr, "Out of range: %d (skipping)\n", num)
			continue
		}

		if !seen[idx] {
			seen[idx] = true
			selected = append(selected, entries[idx])
		}
	}

	return selected, nil
}
