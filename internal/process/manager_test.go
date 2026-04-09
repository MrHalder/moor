package process

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m.GracePeriod != defaultGracePeriod {
		t.Errorf("expected grace period %v, got %v", defaultGracePeriod, m.GracePeriod)
	}
}

func TestIsAlive(t *testing.T) {
	m := NewManager()

	// Current process should be alive
	if !m.IsAlive(int32(os.Getpid())) {
		t.Error("expected current process to be alive")
	}

	// PID 0 is special (kernel), skip; use a very high PID that's unlikely to exist
	if m.IsAlive(int32(999999999)) {
		t.Error("expected nonexistent PID to not be alive")
	}
}

func TestInfo(t *testing.T) {
	m := NewManager()
	info, err := m.Info(int32(os.Getpid()))
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info.PID != int32(os.Getpid()) {
		t.Errorf("expected PID %d, got %d", os.Getpid(), info.PID)
	}
	if info.Name == "" {
		t.Error("expected non-empty process name")
	}
}

func TestKillForce(t *testing.T) {
	// Start a sleep subprocess so we can kill it
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	pid := int32(cmd.Process.Pid)

	m := NewManager()
	if !m.IsAlive(pid) {
		t.Fatal("expected subprocess to be alive")
	}

	err := m.Kill(context.Background(), pid, true)
	if err != nil {
		t.Fatalf("force kill failed: %v", err)
	}

	// Reap the child process (zombie won't disappear until waited)
	_ = cmd.Wait()

	if m.IsAlive(pid) {
		t.Error("expected subprocess to be dead after force kill")
	}
}

func TestKillGraceful(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	pid := int32(cmd.Process.Pid)

	m := NewManager()
	m.GracePeriod = 1 * time.Second

	err := m.Kill(context.Background(), pid, false)
	if err != nil {
		t.Fatalf("graceful kill failed: %v", err)
	}

	// Reap the child process
	_ = cmd.Wait()

	if m.IsAlive(pid) {
		t.Error("expected subprocess to be dead after graceful kill")
	}
}

func TestKillAlreadyDead(t *testing.T) {
	m := NewManager()
	err := m.Kill(context.Background(), int32(999999999), false)
	if err == nil {
		t.Error("expected error when killing nonexistent process")
	}
}
