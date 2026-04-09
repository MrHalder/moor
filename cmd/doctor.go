package cmd

import (
	"context"
	"fmt"

	"github.com/ashutosh/moor/internal/config"
	"github.com/ashutosh/moor/internal/scanner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check port health: conflicts, stale processes, issues",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	pass := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	fail := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	header := lipgloss.NewStyle().Bold(true)

	if noColor {
		pass = lipgloss.NewStyle()
		warn = lipgloss.NewStyle()
		fail = lipgloss.NewStyle()
		header = lipgloss.NewStyle()
	}

	issues := 0

	fmt.Println(header.Render("moor doctor"))
	fmt.Println()

	// 1. Port scanning
	fmt.Println(header.Render("Port Scan"))
	s := scanner.NewScanner()
	result, err := s.ListListening(context.Background())
	if err != nil {
		fmt.Println(fail.Render(fmt.Sprintf("  FAIL  Cannot scan ports: %v", err)))
		issues++
	} else {
		fmt.Println(pass.Render(fmt.Sprintf("  PASS  Found %d listening ports", len(result.Ports))))
		if result.NeedsElevation {
			fmt.Println(warn.Render("  WARN  Limited visibility — run with sudo for complete results"))
			issues++
		}
	}
	fmt.Println()

	// 2. Config
	fmt.Println(header.Render("Configuration"))
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(fail.Render(fmt.Sprintf("  FAIL  Config error: %v", err)))
		fmt.Println(warn.Render(fmt.Sprintf("  TIP   Run 'moor config reset' to fix: %s", config.ConfigPath())))
		issues++
	} else {
		fmt.Println(pass.Render(fmt.Sprintf("  PASS  Config OK (%s)", config.ConfigPath())))
	}
	fmt.Println()

	// 3. Reservations
	if err == nil && len(cfg.Reservations) > 0 {
		fmt.Println(header.Render("Reservations"))

		portMap := make(map[uint16]scanner.PortInfo)
		if result != nil {
			for _, p := range result.Ports {
				portMap[p.LocalPort] = p
			}
		}

		// Check for duplicate ports
		seen := make(map[uint16]bool)
		for _, r := range cfg.Reservations {
			if seen[r.Port] {
				fmt.Println(fail.Render(fmt.Sprintf("  FAIL  Duplicate reservation for port %d", r.Port)))
				issues++
			}
			seen[r.Port] = true
		}

		// Check each reservation
		for _, r := range cfg.Reservations {
			if p, ok := portMap[r.Port]; ok {
				fmt.Println(pass.Render(fmt.Sprintf("  PASS  Port %d (%s) — in use by '%s' (PID %d)",
					r.Port, r.Project, p.ProcessName, p.PID)))
			} else {
				fmt.Println(warn.Render(fmt.Sprintf("  WARN  Port %d (%s) — not currently in use",
					r.Port, r.Project)))
			}
		}
		fmt.Println()
	}

	// 4. Stale/zombie detection
	if result != nil {
		fmt.Println(header.Render("Process Health"))
		zombies := 0
		for _, p := range result.Ports {
			if p.PID > 0 && p.ProcessName == "" {
				zombies++
			}
		}
		if zombies > 0 {
			fmt.Println(warn.Render(fmt.Sprintf("  WARN  %d port(s) held by unidentifiable processes", zombies)))
			issues++
		} else {
			fmt.Println(pass.Render("  PASS  All processes identifiable"))
		}

		// Check for ports < 1024 (privileged)
		privileged := 0
		for _, p := range result.Ports {
			if p.LocalPort < 1024 {
				privileged++
			}
		}
		if privileged > 0 {
			fmt.Println(warn.Render(fmt.Sprintf("  INFO  %d privileged port(s) < 1024 in use", privileged)))
		}
		fmt.Println()
	}

	// Summary
	fmt.Println(header.Render("Summary"))
	if issues == 0 {
		fmt.Println(pass.Render("  All checks passed!"))
	} else {
		fmt.Println(warn.Render(fmt.Sprintf("  %d issue(s) found", issues)))
	}

	return nil
}
