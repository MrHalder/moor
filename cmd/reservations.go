package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MrHalder/moor/internal/config"
	"github.com/MrHalder/moor/internal/scanner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var reservationsCheck bool

var reservationsCmd = &cobra.Command{
	Use:   "reservations",
	Short: "List port reservations and check for conflicts",
	RunE:  runReservations,
}

func init() {
	reservationsCmd.Flags().BoolVar(&reservationsCheck, "check", false, "only show conflicts")
	rootCmd.AddCommand(reservationsCmd)
}

func runReservations(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Reservations) == 0 {
		fmt.Println("No reservations. Use 'moor reserve <port> <project>' to add one.")
		return nil
	}

	// Scan current ports for conflict detection
	s := scanner.NewScanner()
	result, err := s.ListListening(context.Background())
	if err != nil {
		return fmt.Errorf("scanning ports: %w", err)
	}

	portMap := make(map[uint16]scanner.PortInfo)
	for _, p := range result.Ports {
		portMap[p.LocalPort] = p
	}

	if jsonOutput {
		return printReservationsJSON(cfg, portMap)
	}

	return printReservationsTable(cfg, portMap)
}

type reservationStatus struct {
	Reservation config.Reservation `json:"reservation"`
	Status      string             `json:"status"`
	ActualProc  string             `json:"actual_process,omitempty"`
	ActualPID   int32              `json:"actual_pid,omitempty"`
}

func printReservationsJSON(cfg config.Config, portMap map[uint16]scanner.PortInfo) error {
	statuses := make([]reservationStatus, 0, len(cfg.Reservations))
	for _, r := range cfg.Reservations {
		rs := reservationStatus{Reservation: r}
		if p, ok := portMap[r.Port]; ok {
			rs.ActualProc = p.ProcessName
			rs.ActualPID = p.PID
			rs.Status = "in_use"
		} else {
			rs.Status = "free"
		}
		statuses = append(statuses, rs)
	}
	data, err := json.MarshalIndent(statuses, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printReservationsTable(cfg config.Config, portMap map[uint16]scanner.PortInfo) error {
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	conflictStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	freeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))

	if noColor {
		okStyle = lipgloss.NewStyle()
		conflictStyle = lipgloss.NewStyle()
		freeStyle = lipgloss.NewStyle()
		headerStyle = lipgloss.NewStyle()
	}

	header := fmt.Sprintf("%-6s  %-20s  %-30s  %-10s  %-20s", "PORT", "PROJECT", "DESCRIPTION", "STATUS", "ACTUAL PROCESS")
	fmt.Println(headerStyle.Render(header))

	for _, r := range cfg.Reservations {
		status := "free"
		actualProc := "-"
		style := freeStyle

		if p, ok := portMap[r.Port]; ok {
			actualProc = fmt.Sprintf("%s (PID %d)", p.ProcessName, p.PID)
			// Check if it matches the expected project
			if strings.Contains(strings.ToLower(p.ProcessName), strings.ToLower(r.Project)) {
				status = "ok"
				style = okStyle
			} else {
				status = "CONFLICT"
				style = conflictStyle
			}
		}

		if reservationsCheck && status != "CONFLICT" {
			continue
		}

		row := fmt.Sprintf("%-6d  %-20s  %-30s  %-10s  %-20s",
			r.Port,
			truncateStr(r.Project, 20),
			truncateStr(r.Description, 30),
			status,
			truncateStr(actualProc, 20),
		)
		fmt.Println(style.Render(row))
	}

	return nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
