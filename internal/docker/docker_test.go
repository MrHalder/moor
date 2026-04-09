package docker

import (
	"testing"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name      string
		container dockerContainer
		wantCount int
		wantHost  uint16
		wantCont  uint16
		wantProto string
	}{
		{
			name: "simple mapping",
			container: dockerContainer{
				ID:    "abc123",
				Names: "my-nginx",
				Image: "nginx:latest",
				Ports: "0.0.0.0:8080->80/tcp",
				State: "running",
			},
			wantCount: 1,
			wantHost:  8080,
			wantCont:  80,
			wantProto: "tcp",
		},
		{
			name: "multiple mappings",
			container: dockerContainer{
				ID:    "def456",
				Names: "my-app",
				Image: "myapp:v1",
				Ports: "0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp",
				State: "running",
			},
			wantCount: 2,
		},
		{
			name: "no host mapping",
			container: dockerContainer{
				ID:    "ghi789",
				Names: "internal",
				Image: "redis:7",
				Ports: "6379/tcp",
				State: "running",
			},
			wantCount: 0,
		},
		{
			name: "empty ports",
			container: dockerContainer{
				ID:    "jkl012",
				Names: "worker",
				Image: "worker:latest",
				Ports: "",
				State: "running",
			},
			wantCount: 0,
		},
		{
			name: "ipv6 mapping",
			container: dockerContainer{
				ID:    "mno345",
				Names: "ipv6-app",
				Image: "app:latest",
				Ports: ":::9090->9090/tcp",
				State: "running",
			},
			wantCount: 1,
			wantHost:  9090,
			wantCont:  9090,
			wantProto: "tcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports := parsePorts(tt.container)
			if len(ports) != tt.wantCount {
				t.Errorf("expected %d ports, got %d", tt.wantCount, len(ports))
				return
			}
			if tt.wantCount > 0 {
				p := ports[0]
				if tt.wantHost > 0 && p.HostPort != tt.wantHost {
					t.Errorf("expected host port %d, got %d", tt.wantHost, p.HostPort)
				}
				if tt.wantCont > 0 && p.ContainerPort != tt.wantCont {
					t.Errorf("expected container port %d, got %d", tt.wantCont, p.ContainerPort)
				}
				if tt.wantProto != "" && p.Protocol != tt.wantProto {
					t.Errorf("expected proto %s, got %s", tt.wantProto, p.Protocol)
				}
				if p.ContainerName != tt.container.Names {
					t.Errorf("expected name %s, got %s", tt.container.Names, p.ContainerName)
				}
			}
		})
	}
}

func TestExtractPort(t *testing.T) {
	tests := []struct {
		input string
		want  uint16
	}{
		{"0.0.0.0:8080", 8080},
		{":::9090", 9090},
		{"127.0.0.1:3000", 3000},
		{"noport", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := extractPort(tt.input)
		if got != tt.want {
			t.Errorf("extractPort(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestExtractPortAndProto(t *testing.T) {
	tests := []struct {
		input     string
		wantPort  uint16
		wantProto string
	}{
		{"80/tcp", 80, "tcp"},
		{"443/tcp", 443, "tcp"},
		{"53/udp", 53, "udp"},
		{"9090", 9090, "tcp"}, // default proto
	}
	for _, tt := range tests {
		port, proto := extractPortAndProto(tt.input)
		if port != tt.wantPort {
			t.Errorf("extractPortAndProto(%q) port = %d, want %d", tt.input, port, tt.wantPort)
		}
		if proto != tt.wantProto {
			t.Errorf("extractPortAndProto(%q) proto = %s, want %s", tt.input, proto, tt.wantProto)
		}
	}
}
