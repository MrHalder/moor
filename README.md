# moor

> Moor your ports — terminal port management for macOS

A fast, interactive CLI + TUI tool for managing network ports, killing stale processes, reserving ports for projects, and more. Written in Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/MrHalder/moor.svg)](https://pkg.go.dev/github.com/MrHalder/moor)

## Install

### Go (recommended)

```bash
go install github.com/MrHalder/moor@latest
```

Requires Go 1.25.8+. Binary lands in `$GOPATH/bin` (usually `~/go/bin`).

### From release

```bash
# macOS (Apple Silicon)
curl -L https://github.com/MrHalder/moor/releases/latest/download/moor -o moor
chmod +x moor
sudo mv moor /usr/local/bin/
```

### From source

```bash
git clone https://github.com/MrHalder/moor.git
cd moor
make install
```

## Usage

```bash
moor                    # interactive TUI (default)
moor list               # interactive port list (static if piped)
moor list --json        # JSON output
moor find <port>        # what's using this port?
moor kill <port>        # kill process on a port
moor kill <port> -f     # force kill (SIGKILL)
moor pkill              # list all processes, pick which to kill
moor pkill node         # find & kill processes matching "node"
moor pkill -F server.js # match against full command line
moor watch              # real-time TUI dashboard
moor reserve <port> <project>   # reserve a port
moor reservations       # list all reservations
moor doctor             # health check
moor forward <from> <to>        # port forwarding
moor config show        # view config
```

## Interactive Mode

Run `moor` or `moor list` in a terminal for the interactive experience:

- **Arrow keys** to navigate ports
- **Enter** to open action menu (kill, details, force kill, etc.)
- **/** to filter
- **s** to cycle sort column
- **a** to toggle all connections / LISTEN only
- **q** to quit

## Features

- **List & Find** — See all listening ports with process name, PID, user
- **Kill** — Graceful kill (SIGTERM -> SIGKILL) or force kill by port
- **Pkill** — Find and kill processes by name with interactive selection
- **Interactive TUI** — Full-screen dashboard with auto-refresh, filtering, sorting
- **Port Reservations** — Assign ports to projects, detect conflicts
- **Doctor** — Health check for port conflicts, stale processes, zombies
- **Port Forwarding** — Forward local port A to local port B
- **Docker Integration** — Show Docker container port mappings inline
- **.env Integration** — Auto-reserve ports from `.env` files
- **JSON Output** — All read commands support `--json`
- **Sudo-aware** — Works without sudo (partial view), full view with sudo

## Config

Config lives at `~/.config/moor/config.yaml` (XDG compliant on macOS at `~/Library/Application Support/moor/config.yaml`).

```yaml
settings:
  refresh_interval_seconds: 2
  grace_period_seconds: 3
  show_docker: true
  default_output: table

reservations:
  - port: 3000
    project: frontend
    description: React dev server
  - port: 8080
    project: api-server
```

## Tech Stack

- **Go** with Cobra CLI
- **Bubble Tea** + Lipgloss for TUI
- **gopsutil** for port scanning
- macOS only (for now)

## License

MIT
