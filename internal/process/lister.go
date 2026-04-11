package process

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
)

// minUserPID is the lowest PID we consider a user process.
// PIDs below this are kernel/system daemons on macOS (launchd, syslogd, etc.).
const minUserPID int32 = 100

// Lister enumerates running processes on the system.
type Lister interface {
	// List returns all user processes, sorted by name then PID.
	List(ctx context.Context) ([]ProcessInfo, error)
}

// DefaultLister implements Lister using gopsutil.
type DefaultLister struct{}

// NewLister creates a new DefaultLister.
func NewLister() *DefaultLister {
	return &DefaultLister{}
}

func (l *DefaultLister) List(ctx context.Context) ([]ProcessInfo, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing processes: %w", err)
	}

	entries := make([]ProcessInfo, 0, len(procs))
	for _, p := range procs {
		pid := p.Pid
		if pid < minUserPID {
			continue
		}

		name, _ := p.NameWithContext(ctx)
		if name == "" {
			continue
		}

		user, _ := p.UsernameWithContext(ctx)
		cmdline, _ := p.CmdlineWithContext(ctx)

		entries = append(entries, ProcessInfo{
			PID:         pid,
			Name:        name,
			User:        user,
			CommandLine: cmdline,
		})
	}

	sort.Stable(sortByNameThenPID(entries))

	return entries, nil
}

// sortByNameThenPID implements sort.Interface for ProcessInfo slices.
type sortByNameThenPID []ProcessInfo

func (s sortByNameThenPID) Len() int      { return len(s) }
func (s sortByNameThenPID) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortByNameThenPID) Less(i, j int) bool {
	ni := strings.ToLower(s[i].Name)
	nj := strings.ToLower(s[j].Name)
	if ni != nj {
		return ni < nj
	}
	return s[i].PID < s[j].PID
}

// FilterByName returns entries whose name contains the pattern (case-insensitive).
func FilterByName(entries []ProcessInfo, pattern string, exact bool) []ProcessInfo {
	filtered := make([]ProcessInfo, 0, len(entries))
	lowerPattern := strings.ToLower(pattern)

	for _, e := range entries {
		lowerName := strings.ToLower(e.Name)
		if exact {
			if lowerName == lowerPattern {
				filtered = append(filtered, e)
			}
		} else {
			if strings.Contains(lowerName, lowerPattern) {
				filtered = append(filtered, e)
			}
		}
	}
	return filtered
}

// FilterByCommandLine returns entries whose full command line contains the pattern (case-insensitive).
func FilterByCommandLine(entries []ProcessInfo, pattern string) []ProcessInfo {
	filtered := make([]ProcessInfo, 0, len(entries))
	lowerPattern := strings.ToLower(pattern)

	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.CommandLine), lowerPattern) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
