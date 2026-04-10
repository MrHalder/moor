package formatter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MrHalder/moor/internal/scanner"
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	header      lipgloss.Style
	listen      lipgloss.Style
	established lipgloss.Style
	dim         lipgloss.Style
	warn        lipgloss.Style
}

func newStyles(noColor bool) styles {
	if noColor {
		plain := lipgloss.NewStyle()
		return styles{plain, plain, plain, plain, plain}
	}
	return styles{
		header: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")),
		listen: lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
		established: lipgloss.NewStyle().Foreground(lipgloss.Color("75")),
		dim:    lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		warn:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
	}
}

// FormatTable renders port info as a styled CLI table.
func FormatTable(result *scanner.ScanResult, noColor bool) string {
	st := newStyles(noColor)

	ports := make([]scanner.PortInfo, len(result.Ports))
	copy(ports, result.Ports)
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].LocalPort != ports[j].LocalPort {
			return ports[i].LocalPort < ports[j].LocalPort
		}
		return ports[i].Protocol < ports[j].Protocol
	})

	if len(ports) == 0 {
		msg := "No ports found."
		if result.NeedsElevation {
			msg += " Run with sudo for complete results."
		}
		return msg
	}

	// Calculate column widths
	cols := []string{"PROTO", "ADDRESS", "PORT", "PID", "PROCESS", "USER", "STATE"}
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c)
	}

	rows := make([][]string, 0, len(ports))
	for _, p := range ports {
		row := []string{
			p.Protocol,
			p.LocalAddr,
			fmt.Sprintf("%d", p.LocalPort),
			pidString(p.PID),
			p.ProcessName,
			p.User,
			p.State,
		}
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
		rows = append(rows, row)
	}

	var sb strings.Builder

	// Header
	header := formatRow(cols, widths)
	sb.WriteString(st.header.Render(header))
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		line := formatRow(row, widths)
		state := row[6]
		styled := styleByState(st, line, state)
		sb.WriteString(styled)
		sb.WriteString("\n")
	}

	if result.NeedsElevation {
		sb.WriteString("\n")
		sb.WriteString(st.warn.Render("⚠ Limited view — run with sudo for all processes"))
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
	}
	return strings.Join(parts, "  ")
}

func styleByState(st styles, line, state string) string {
	switch strings.ToUpper(state) {
	case "LISTEN":
		return st.listen.Render(line)
	case "ESTABLISHED":
		return st.established.Render(line)
	case "TIME_WAIT", "CLOSE_WAIT", "FIN_WAIT1", "FIN_WAIT2":
		return st.dim.Render(line)
	default:
		return line
	}
}

func pidString(pid int32) string {
	if pid <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d", pid)
}
