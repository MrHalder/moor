package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Settings.RefreshIntervalSecs != 2 {
		t.Errorf("expected refresh 2, got %d", cfg.Settings.RefreshIntervalSecs)
	}
	if cfg.Settings.GracePeriodSecs != 3 {
		t.Errorf("expected grace 3, got %d", cfg.Settings.GracePeriodSecs)
	}
	if !cfg.Settings.ShowDocker {
		t.Error("expected ShowDocker true")
	}
	if len(cfg.Reservations) != 0 {
		t.Error("expected no reservations")
	}
}

func TestAddReservation(t *testing.T) {
	cfg := DefaultConfig()
	r := Reservation{Port: 3000, Project: "frontend"}
	updated := cfg.AddReservation(r)

	if len(updated.Reservations) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(updated.Reservations))
	}
	if updated.Reservations[0].Port != 3000 {
		t.Errorf("expected port 3000, got %d", updated.Reservations[0].Port)
	}
	if updated.Reservations[0].CreatedAt == "" {
		t.Error("expected created_at to be set")
	}

	// Original unchanged (immutability)
	if len(cfg.Reservations) != 0 {
		t.Error("original config should be unchanged")
	}
}

func TestAddReservationReplaces(t *testing.T) {
	cfg := DefaultConfig()
	cfg = cfg.AddReservation(Reservation{Port: 3000, Project: "frontend"})
	cfg = cfg.AddReservation(Reservation{Port: 3000, Project: "new-frontend"})

	if len(cfg.Reservations) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(cfg.Reservations))
	}
	if cfg.Reservations[0].Project != "new-frontend" {
		t.Errorf("expected new-frontend, got %s", cfg.Reservations[0].Project)
	}
}

func TestRemoveReservation(t *testing.T) {
	cfg := DefaultConfig()
	cfg = cfg.AddReservation(Reservation{Port: 3000, Project: "a"})
	cfg = cfg.AddReservation(Reservation{Port: 8080, Project: "b"})
	cfg = cfg.RemoveReservation(3000)

	if len(cfg.Reservations) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(cfg.Reservations))
	}
	if cfg.Reservations[0].Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Reservations[0].Port)
	}
}

func TestFindReservation(t *testing.T) {
	cfg := DefaultConfig()
	cfg = cfg.AddReservation(Reservation{Port: 3000, Project: "frontend"})

	found := cfg.FindReservation(3000)
	if found == nil {
		t.Fatal("expected to find reservation")
	}
	if found.Project != "frontend" {
		t.Errorf("expected frontend, got %s", found.Project)
	}

	notFound := cfg.FindReservation(9999)
	if notFound != nil {
		t.Error("expected nil for unknown port")
	}
}

func withTempConfigDir(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	old := overrideDir
	overrideDir = tmpDir
	t.Cleanup(func() { overrideDir = old })
}

func TestSaveAndLoad(t *testing.T) {
	withTempConfigDir(t)

	cfg := DefaultConfig()
	cfg = cfg.AddReservation(Reservation{Port: 3000, Project: "test-app", Description: "dev server"})

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", path)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Reservations) != 1 {
		t.Fatalf("expected 1 reservation, got %d", len(loaded.Reservations))
	}
	if loaded.Reservations[0].Project != "test-app" {
		t.Errorf("expected test-app, got %s", loaded.Reservations[0].Project)
	}
}

func TestLoadMissing(t *testing.T) {
	withTempConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load on missing file should return defaults, got error: %v", err)
	}
	if cfg.Settings.RefreshIntervalSecs != 2 {
		t.Error("expected default settings")
	}
}
