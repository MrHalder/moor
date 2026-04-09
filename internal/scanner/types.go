package scanner

// PortInfo represents a single network connection with its associated process.
type PortInfo struct {
	Protocol    string         `json:"protocol"`
	LocalAddr   string         `json:"local_addr"`
	LocalPort   uint16         `json:"local_port"`
	RemoteAddr  string         `json:"remote_addr,omitempty"`
	RemotePort  uint16         `json:"remote_port,omitempty"`
	State       string         `json:"state"`
	PID         int32          `json:"pid"`
	ProcessName string         `json:"process_name"`
	User        string         `json:"user,omitempty"`
	CommandLine string         `json:"command_line,omitempty"`
	DockerInfo  *DockerPortInfo `json:"docker_info,omitempty"`
}

// DockerPortInfo holds container-specific metadata for a port.
type DockerPortInfo struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Image         string `json:"image"`
}

// ScanResult holds the outcome of a port scan.
type ScanResult struct {
	Ports          []PortInfo `json:"ports"`
	NeedsElevation bool       `json:"needs_elevation"`
}
