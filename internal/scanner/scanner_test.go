package scanner

import (
	"context"
	"testing"
)

func TestNewScanner(t *testing.T) {
	s := NewScanner()
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
	if s.procCache == nil {
		t.Fatal("expected non-nil procCache")
	}
}

func TestListAll(t *testing.T) {
	s := NewScanner()
	result, err := s.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// We should have at least some connections on any running system
	if len(result.Ports) == 0 {
		t.Log("warning: no connections found (might need elevated privileges)")
	}
}

func TestListListening(t *testing.T) {
	s := NewScanner()
	result, err := s.ListListening(context.Background())
	if err != nil {
		t.Fatalf("ListListening failed: %v", err)
	}
	for _, p := range result.Ports {
		if p.State != "LISTEN" {
			t.Errorf("expected LISTEN state, got %s for port %d", p.State, p.LocalPort)
		}
	}
}

func TestFindByPort(t *testing.T) {
	s := NewScanner()
	// Port 0 should return no results
	result, err := s.FindByPort(context.Background(), 0)
	if err != nil {
		t.Fatalf("FindByPort failed: %v", err)
	}
	if len(result.Ports) != 0 {
		t.Errorf("expected 0 results for port 0, got %d", len(result.Ports))
	}
}

func TestProtocolString(t *testing.T) {
	tests := []struct {
		input uint32
		want  string
	}{
		{1, "tcp"},
		{2, "udp"},
		{99, "unknown(99)"},
	}
	for _, tt := range tests {
		got := protocolString(tt.input)
		if got != tt.want {
			t.Errorf("protocolString(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConnectionState(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "NONE"},
		{"listen", "LISTEN"},
		{"ESTABLISHED", "ESTABLISHED"},
		{"time_wait", "TIME_WAIT"},
	}
	for _, tt := range tests {
		got := connectionState(tt.input)
		if got != tt.want {
			t.Errorf("connectionState(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
