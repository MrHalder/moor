package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ContainerPort represents a Docker container's port mapping.
type ContainerPort struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Image         string `json:"image"`
	HostPort      uint16 `json:"host_port"`
	ContainerPort uint16 `json:"container_port"`
	Protocol      string `json:"protocol"`
	State         string `json:"state"`
}

// Client queries Docker for container port mappings.
type Client struct{}

// NewClient creates a Docker client.
func NewClient() *Client {
	return &Client{}
}

// Available returns true if Docker CLI is accessible and the daemon is running.
func (c *Client) Available(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ID}}")
	return cmd.Run() == nil
}

// dockerContainer is the JSON structure from `docker ps --format json`.
type dockerContainer struct {
	ID    string `json:"ID"`
	Names string `json:"Names"`
	Image string `json:"Image"`
	Ports string `json:"Ports"`
	State string `json:"State"`
}

// ListPortMappings returns all container port mappings.
func (c *Client) ListPortMappings(ctx context.Context) ([]ContainerPort, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w", err)
	}

	var ports []ContainerPort
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		var container dockerContainer
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			continue
		}

		parsed := parsePorts(container)
		ports = append(ports, parsed...)
	}

	return ports, nil
}

// FindByPort returns containers using a specific host port.
func (c *Client) FindByPort(ctx context.Context, port uint16) ([]ContainerPort, error) {
	all, err := c.ListPortMappings(ctx)
	if err != nil {
		return nil, err
	}

	var matched []ContainerPort
	for _, p := range all {
		if p.HostPort == port {
			matched = append(matched, p)
		}
	}
	return matched, nil
}

// parsePorts extracts port mappings from a container's Ports string.
// Format examples:
//   - "0.0.0.0:8080->80/tcp"
//   - "0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp"
//   - "80/tcp" (no host mapping)
func parsePorts(c dockerContainer) []ContainerPort {
	if c.Ports == "" {
		return nil
	}

	var ports []ContainerPort
	for _, mapping := range strings.Split(c.Ports, ", ") {
		mapping = strings.TrimSpace(mapping)
		if mapping == "" {
			continue
		}

		// Only care about host-mapped ports (contains "->")
		if !strings.Contains(mapping, "->") {
			continue
		}

		// "0.0.0.0:8080->80/tcp"
		parts := strings.SplitN(mapping, "->", 2)
		if len(parts) != 2 {
			continue
		}

		hostPart := parts[0]  // "0.0.0.0:8080"
		contPart := parts[1]  // "80/tcp"

		// Extract host port
		hostPort := extractPort(hostPart)
		if hostPort == 0 {
			continue
		}

		// Extract container port and protocol
		contPort, proto := extractPortAndProto(contPart)

		ports = append(ports, ContainerPort{
			ContainerID:   c.ID,
			ContainerName: strings.TrimPrefix(c.Names, "/"),
			Image:         c.Image,
			HostPort:      hostPort,
			ContainerPort: contPort,
			Protocol:      proto,
			State:         c.State,
		})
	}

	return ports
}

func extractPort(s string) uint16 {
	// "0.0.0.0:8080" or ":::8080" (IPv6)
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return 0
	}
	n, err := strconv.ParseUint(s[idx+1:], 10, 16)
	if err != nil {
		return 0
	}
	return uint16(n)
}

func extractPortAndProto(s string) (uint16, string) {
	// "80/tcp"
	parts := strings.SplitN(s, "/", 2)
	proto := "tcp"
	if len(parts) == 2 {
		proto = parts[1]
	}
	n, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return 0, proto
	}
	return uint16(n), proto
}
