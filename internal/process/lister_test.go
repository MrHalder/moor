package process

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultListerList(t *testing.T) {
	l := NewLister()
	entries, err := l.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one process")
	}

	// Verify no kernel/system processes (PID < minUserPID) are included
	for _, e := range entries {
		if e.PID < minUserPID {
			t.Errorf("unexpected system process PID %d in results", e.PID)
		}
	}
}

func TestDefaultListerListSorted(t *testing.T) {
	l := NewLister()
	entries, err := l.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	for i := 1; i < len(entries); i++ {
		prevName := strings.ToLower(entries[i-1].Name)
		currName := strings.ToLower(entries[i].Name)
		if prevName > currName {
			t.Errorf("entries not sorted: %q (PID %d) came before %q (PID %d)",
				entries[i-1].Name, entries[i-1].PID, entries[i].Name, entries[i].PID)
		}
		if prevName == currName && entries[i-1].PID > entries[i].PID {
			t.Errorf("same-name entries not sorted by PID: %d came before %d for process %q",
				entries[i-1].PID, entries[i].PID, entries[i-1].Name)
		}
	}
}

func TestFilterByName(t *testing.T) {
	entries := []ProcessInfo{
		{PID: 1000, Name: "node", CommandLine: "node server.js"},
		{PID: 1001, Name: "nodemon", CommandLine: "nodemon --watch src"},
		{PID: 1002, Name: "python3", CommandLine: "python3 app.py"},
		{PID: 1003, Name: "Node", CommandLine: "Node worker.js"},
	}

	tests := []struct {
		name    string
		pattern string
		exact   bool
		want    int
	}{
		{"substring match", "node", false, 3},         // node, nodemon, Node
		{"exact match", "node", true, 2},              // node, Node (case-insensitive)
		{"no match", "ruby", false, 0},                // nothing
		{"case insensitive", "NODE", false, 3},        // node, nodemon, Node
		{"exact case insensitive", "NODE", true, 2},   // node, Node
		{"partial match", "python", false, 1},         // python3
		{"exact no partial", "python", true, 0},       // python3 doesn't match exactly
		{"empty pattern", "", false, 4},               // matches all
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByName(entries, tt.pattern, tt.exact)
			if len(got) != tt.want {
				t.Errorf("FilterByName(%q, exact=%v) returned %d entries, want %d",
					tt.pattern, tt.exact, len(got), tt.want)
			}
		})
	}
}

func TestFilterByCommandLine(t *testing.T) {
	entries := []ProcessInfo{
		{PID: 1000, Name: "node", CommandLine: "node server.js"},
		{PID: 1001, Name: "node", CommandLine: "node worker.js"},
		{PID: 1002, Name: "python3", CommandLine: "python3 app.py"},
	}

	tests := []struct {
		name    string
		pattern string
		want    int
	}{
		{"match one", "server.js", 1},
		{"match by runtime", "node", 2},
		{"match all js", ".js", 2},
		{"case insensitive", "SERVER.JS", 1},
		{"no match", "ruby", 0},
		{"partial path", "app.py", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByCommandLine(entries, tt.pattern)
			if len(got) != tt.want {
				t.Errorf("FilterByCommandLine(%q) returned %d entries, want %d",
					tt.pattern, len(got), tt.want)
			}
		})
	}
}

func TestFilterByNameImmutability(t *testing.T) {
	entries := []ProcessInfo{
		{PID: 1000, Name: "node"},
		{PID: 1001, Name: "python3"},
	}

	original := make([]ProcessInfo, len(entries))
	copy(original, entries)

	_ = FilterByName(entries, "node", false)

	if len(entries) != len(original) {
		t.Error("FilterByName mutated the original slice length")
	}
	for i := range entries {
		if entries[i] != original[i] {
			t.Error("FilterByName mutated the original slice entries")
		}
	}
}

func TestFilterByCommandLineImmutability(t *testing.T) {
	entries := []ProcessInfo{
		{PID: 1000, Name: "node", CommandLine: "node server.js"},
		{PID: 1001, Name: "python3", CommandLine: "python3 app.py"},
	}

	original := make([]ProcessInfo, len(entries))
	copy(original, entries)

	_ = FilterByCommandLine(entries, "node")

	if len(entries) != len(original) {
		t.Error("FilterByCommandLine mutated the original slice length")
	}
	for i := range entries {
		if entries[i] != original[i] {
			t.Error("FilterByCommandLine mutated the original slice entries")
		}
	}
}
