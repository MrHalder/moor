package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	content := `# Application config
PORT=3000
API_PORT=8080
VITE_PORT="5173"
DB_PORT='5432'
NOT_A_PORT=hello
HOST=localhost
DATABASE_URL=postgres://localhost:5432/mydb
INVALID_PORT=99999
ZERO_PORT=0
`
	tmpFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ports, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := map[string]uint16{
		"PORT":      3000,
		"API_PORT":  8080,
		"VITE_PORT": 5173,
		"DB_PORT":   5432,
	}

	if len(ports) != len(expected) {
		t.Errorf("expected %d ports, got %d", len(expected), len(ports))
		for _, p := range ports {
			t.Logf("  found: %s=%d", p.Key, p.Value)
		}
	}

	for _, p := range ports {
		want, ok := expected[p.Key]
		if !ok {
			t.Errorf("unexpected port key: %s", p.Key)
			continue
		}
		if p.Value != want {
			t.Errorf("key %s: expected %d, got %d", p.Key, want, p.Value)
		}
	}
}

func TestParseEmptyFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(tmpFile, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	ports, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(ports) != 0 {
		t.Errorf("expected 0 ports, got %d", len(ports))
	}
}

func TestParseMissingFile(t *testing.T) {
	_, err := Parse("/nonexistent/.env")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestScanDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create .env
	env1 := "PORT=3000\n"
	os.WriteFile(filepath.Join(dir, ".env"), []byte(env1), 0o644)

	// Create .env.local
	env2 := "API_PORT=8080\n"
	os.WriteFile(filepath.Join(dir, ".env.local"), []byte(env2), 0o644)

	// Create non-env file (should be ignored)
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("port: 9090"), 0o644)

	ports, err := ScanDirectory(dir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	if len(ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(ports))
	}
}

func TestIsPortKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"PORT", true},
		{"port", true},
		{"API_PORT", true},
		{"VITE_PORT", true},
		{"DB_PORT", true},
		{"HOST", false},
		{"DATABASE_URL", false},
		{"REPORT_DIR", true}, // contains PORT — acceptable false positive
	}
	for _, tt := range tests {
		got := isPortKey(tt.key)
		if got != tt.want {
			t.Errorf("isPortKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}
