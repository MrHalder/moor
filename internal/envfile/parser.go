package envfile

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const maxEnvFileSize = 10 << 20 // 10 MB

// EnvPort represents a port value found in a .env file.
type EnvPort struct {
	FilePath string `json:"file_path"`
	Key      string `json:"key"`
	Value    uint16 `json:"value"`
}

// Parse reads a .env file and extracts port-like values.
// Looks for keys containing "PORT" (case-insensitive).
func Parse(path string) ([]EnvPort, error) {
	// Check file size before reading
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if info.Size() > maxEnvFileSize {
		return nil, fmt.Errorf("file %s too large (%d bytes, max %d)", path, info.Size(), maxEnvFileSize)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	var ports []EnvPort
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and blank lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		value := strings.TrimSpace(line[eqIdx+1:])

		// Remove surrounding quotes
		value = strings.Trim(value, `"'`)

		// Check if key looks port-related
		if !isPortKey(key) {
			continue
		}

		// Try to parse as port number
		port, err := strconv.ParseUint(value, 10, 16)
		if err != nil || port == 0 || port > 65535 {
			continue
		}

		ports = append(ports, EnvPort{
			FilePath: absPath,
			Key:      key,
			Value:    uint16(port),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	return ports, nil
}

// ScanDirectory finds .env files in a directory and extracts ports.
// Only processes regular files — symlinks and special files are skipped.
func ScanDirectory(dir string) ([]EnvPort, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir, err)
	}

	var allPorts []EnvPort
	var warnings []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Skip symlinks and non-regular files to prevent FIFO hangs and symlink attacks
		if entry.Type()&fs.ModeType != 0 {
			continue
		}

		name := entry.Name()
		if name == ".env" || strings.HasPrefix(name, ".env.") {
			ports, err := Parse(filepath.Join(dir, name))
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("skipping %s: %v", name, err))
				continue
			}
			allPorts = append(allPorts, ports...)
		}
	}

	if len(warnings) > 0 {
		fmt.Fprintf(os.Stderr, "moor: %s\n", strings.Join(warnings, "; "))
	}

	return allPorts, nil
}

// isPortKey checks if an env var key is likely a port configuration.
func isPortKey(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "PORT")
}
