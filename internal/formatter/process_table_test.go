package formatter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MrHalder/moor/internal/process"
)

func TestFormatProcessTableEmpty(t *testing.T) {
	got := FormatProcessTable(nil, true)
	if got != "No matching processes found." {
		t.Errorf("expected empty message, got %q", got)
	}
}

func TestFormatProcessTableHeader(t *testing.T) {
	entries := []process.ProcessInfo{
		{PID: 1234, Name: "node", User: "ash", CommandLine: "node server.js"},
	}

	got := FormatProcessTable(entries, true) // noColor for predictable output
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (header + 1 row), got %d", len(lines))
	}

	header := lines[0]
	for _, col := range []string{"#", "PID", "PROCESS", "USER", "COMMAND"} {
		if !strings.Contains(header, col) {
			t.Errorf("header missing column %q: %s", col, header)
		}
	}
}

func TestFormatProcessTableRows(t *testing.T) {
	entries := []process.ProcessInfo{
		{PID: 1234, Name: "node", User: "ash", CommandLine: "node server.js"},
		{PID: 5678, Name: "python3", User: "ash", CommandLine: "python3 app.py"},
	}

	got := FormatProcessTable(entries, true)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	// header + 2 data rows
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	if !strings.Contains(lines[1], "1234") || !strings.Contains(lines[1], "node") {
		t.Errorf("row 1 missing expected data: %s", lines[1])
	}

	if !strings.Contains(lines[2], "5678") || !strings.Contains(lines[2], "python3") {
		t.Errorf("row 2 missing expected data: %s", lines[2])
	}
}

func TestFormatProcessTableNumbering(t *testing.T) {
	entries := make([]process.ProcessInfo, 3)
	for i := range entries {
		entries[i] = process.ProcessInfo{
			PID:  int32(1000 + i),
			Name: "proc",
			User: "user",
		}
	}

	got := FormatProcessTable(entries, true)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")

	// Skip header (line 0), check numbering in data rows
	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) == 0 {
			continue
		}
		wantNum := fmt.Sprintf("%d", i) // rows are 1-indexed, i starts at 1
		if fields[0] != wantNum {
			t.Errorf("row %d: expected number %q, got %q", i, wantNum, fields[0])
		}
	}
}

func TestFormatProcessTableSanitizesDisplay(t *testing.T) {
	entries := []process.ProcessInfo{
		{PID: 100, Name: "bad\x1b[31mname", User: "user", CommandLine: "cmd\x07line"},
	}

	got := FormatProcessTable(entries, true)

	if strings.Contains(got, "\x1b") {
		t.Error("output contains unescaped ANSI sequence")
	}
	if strings.Contains(got, "\x07") {
		t.Error("output contains BEL character")
	}
}

func TestTruncateCommand(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		maxLen int
		want   string
	}{
		{"short enough", "node server.js", 60, "node server.js"},
		{"exactly at limit", "abc", 3, "abc"},
		{"needs truncation", "a very long command line string here", 10, "a very ..."}, // 7 runes + "..."
		{"empty", "", 60, ""},
		{"zero max", "hello", 0, "hello"},
		{"maxLen is 3", "hello", 3, "hel"},
		{"maxLen is 2", "hello", 2, "he"},
		{"maxLen is 1", "hello", 1, "h"},
		{"unicode safe", "\u00d1o\u00f1o hello world", 6, "\u00d1o\u00f1..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateCommand(tt.cmd, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateCommand(%q, %d) = %q, want %q", tt.cmd, tt.maxLen, got, tt.want)
			}
		})
	}
}
