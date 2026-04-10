package scanner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"syscall"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// PortScanner enumerates network connections on the system.
type PortScanner interface {
	// ListAll returns all network connections.
	ListAll(ctx context.Context) (*ScanResult, error)

	// ListListening returns only ports in LISTEN state.
	ListListening(ctx context.Context) (*ScanResult, error)

	// FindByPort returns connections on a specific port.
	FindByPort(ctx context.Context, port uint16) (*ScanResult, error)
}

// GopsutilScanner implements PortScanner using gopsutil.
type GopsutilScanner struct {
	mu        sync.Mutex
	procCache map[int32]*cachedProc
}

type cachedProc struct {
	name        string
	user        string
	commandLine string
}

// NewScanner creates a new GopsutilScanner.
func NewScanner() *GopsutilScanner {
	return &GopsutilScanner{
		procCache: make(map[int32]*cachedProc),
	}
}

func (s *GopsutilScanner) ListAll(ctx context.Context) (*ScanResult, error) {
	return s.scan(ctx, "")
}

func (s *GopsutilScanner) ListListening(ctx context.Context) (*ScanResult, error) {
	return s.scan(ctx, "LISTEN")
}

func (s *GopsutilScanner) FindByPort(ctx context.Context, port uint16) (*ScanResult, error) {
	result, err := s.scan(ctx, "")
	if err != nil {
		return nil, err
	}

	filtered := make([]PortInfo, 0)
	for _, p := range result.Ports {
		if p.LocalPort == port {
			filtered = append(filtered, p)
		}
	}

	return &ScanResult{
		Ports:          filtered,
		NeedsElevation: result.NeedsElevation,
	}, nil
}

func (s *GopsutilScanner) scan(ctx context.Context, stateFilter string) (*ScanResult, error) {
	connections, err := net.ConnectionsWithContext(ctx, "inet")
	if err != nil {
		return nil, fmt.Errorf("scanning connections: %w", err)
	}

	s.mu.Lock()
	s.procCache = make(map[int32]*cachedProc)
	s.mu.Unlock()

	needsElevation := false
	ports := make([]PortInfo, 0, len(connections))

	for _, conn := range connections {
		state := connectionState(conn.Status)

		if stateFilter != "" && !strings.EqualFold(state, stateFilter) {
			continue
		}

		if conn.Laddr.Port > 65535 || conn.Raddr.Port > 65535 {
			continue
		}

		info := PortInfo{
			Protocol:   protocolString(conn.Type),
			LocalAddr:  conn.Laddr.IP,
			LocalPort:  uint16(conn.Laddr.Port),
			RemoteAddr: conn.Raddr.IP,
			RemotePort: uint16(conn.Raddr.Port),
			State:      state,
			PID:        conn.Pid,
		}

		if conn.Pid > 0 {
			proc := s.lookupProcess(conn.Pid)
			if proc != nil {
				info.ProcessName = proc.name
				info.User = proc.user
				info.CommandLine = proc.commandLine
			} else {
				// Process exists but we can't read its info — likely a permissions issue
				needsElevation = true
			}
		} else if state == "LISTEN" {
			needsElevation = true
		}

		ports = append(ports, info)
	}

	return &ScanResult{
		Ports:          ports,
		NeedsElevation: needsElevation,
	}, nil
}

func (s *GopsutilScanner) lookupProcess(pid int32) *cachedProc {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cached, ok := s.procCache[pid]; ok {
		return cached
	}

	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil
	}

	name, _ := proc.Name()
	user, _ := proc.Username()
	cmdline, _ := proc.Cmdline()

	cached := &cachedProc{
		name:        name,
		user:        user,
		commandLine: cmdline,
	}

	s.procCache[pid] = cached
	return cached
}

func protocolString(connType uint32) string {
	switch connType {
	case syscall.SOCK_STREAM:
		return "tcp"
	case syscall.SOCK_DGRAM:
		return "udp"
	default:
		return fmt.Sprintf("unknown(%d)", connType)
	}
}

func connectionState(status string) string {
	if status == "" {
		return "NONE"
	}
	return strings.ToUpper(status)
}
