package process

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

const defaultGracePeriod = 3 * time.Second
const pollInterval = 100 * time.Millisecond

// Manager handles process lifecycle operations.
type Manager interface {
	// Kill sends SIGTERM, waits gracePeriod, then SIGKILL if still alive.
	// If force is true, sends SIGKILL immediately.
	Kill(ctx context.Context, pid int32, force bool) error

	// IsAlive checks if a process is still running.
	IsAlive(pid int32) bool

	// Info returns process details.
	Info(pid int32) (*ProcessInfo, error)
}

// ProcessInfo holds metadata about a running process.
type ProcessInfo struct {
	PID         int32  `json:"pid"`
	Name        string `json:"name"`
	User        string `json:"user"`
	CommandLine string `json:"command_line"`
}

// DefaultManager implements Manager using syscalls.
type DefaultManager struct {
	GracePeriod time.Duration
}

// NewManager creates a DefaultManager with default settings.
func NewManager() *DefaultManager {
	return &DefaultManager{
		GracePeriod: defaultGracePeriod,
	}
}

func (m *DefaultManager) Kill(ctx context.Context, pid int32, force bool) error {
	if pid <= 0 {
		return fmt.Errorf("invalid PID %d: must be positive", pid)
	}

	if !m.IsAlive(pid) {
		return fmt.Errorf("process %d is not running", pid)
	}

	if force {
		if err := syscall.Kill(int(pid), syscall.SIGKILL); err != nil {
			return fmt.Errorf("sending SIGKILL to %d: %w", pid, err)
		}
		return nil
	}

	// Graceful: SIGTERM first
	if err := syscall.Kill(int(pid), syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	// Poll until dead or grace period expires
	deadline := time.After(m.GracePeriod)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			// Grace period expired, force kill
			if m.IsAlive(pid) {
				return syscall.Kill(int(pid), syscall.SIGKILL)
			}
			return nil
		case <-ticker.C:
			if !m.IsAlive(pid) {
				return nil
			}
		}
	}
}

func (m *DefaultManager) IsAlive(pid int32) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(int(pid), 0)
	return err == nil
}

func (m *DefaultManager) Info(pid int32) (*ProcessInfo, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid PID %d: must be positive", pid)
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("process %d not found: %w", pid, err)
	}

	name, _ := proc.Name()
	user, _ := proc.Username()
	cmdline, _ := proc.Cmdline()

	return &ProcessInfo{
		PID:         pid,
		Name:        name,
		User:        user,
		CommandLine: cmdline,
	}, nil
}
