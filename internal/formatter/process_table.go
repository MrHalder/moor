package formatter

import (
	"fmt"
	"strings"

	"github.com/MrHalder/moor/internal/process"
)

// FormatProcessTable renders process entries as a numbered CLI table for interactive selection.
func FormatProcessTable(entries []process.ProcessInfo, noColor bool) string {
	if len(entries) == 0 {
		return "No matching processes found."
	}

	st := newStyles(noColor)
	plain := st.dim // reuse existing style instead of allocating per-row

	cols := []string{"#", "PID", "PROCESS", "USER", "COMMAND"}
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c)
	}

	rows := make([][]string, 0, len(entries))
	for i, e := range entries {
		cmd := SanitizeDisplay(truncateCommand(e.CommandLine, 60))
		name := SanitizeDisplay(e.Name)
		user := SanitizeDisplay(e.User)

		row := []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("%d", e.PID),
			name,
			user,
			cmd,
		}
		for j, cell := range row {
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
		rows = append(rows, row)
	}

	var sb strings.Builder

	header := formatRow(cols, widths)
	sb.WriteString(st.header.Render(header))
	sb.WriteString("\n")

	for _, row := range rows {
		line := formatRow(row, widths)
		sb.WriteString(plain.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}

// truncateCommand shortens a command line string to maxLen runes, appending "..." if truncated.
func truncateCommand(cmd string, maxLen int) string {
	if maxLen <= 0 {
		return cmd
	}
	runes := []rune(cmd)
	if len(runes) <= maxLen {
		return cmd
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}
