# moor

Terminal-based port management tool for macOS. Written in Go.

## Build & Run

```bash
make build        # builds ./moor binary
make test         # runs all tests with race detector
make lint         # go vet
make install      # copies to /usr/local/bin/moor
```

## Architecture

- `cmd/` — Cobra CLI commands
- `internal/scanner/` — Port scanning via gopsutil
- `internal/process/` — Process kill/signal management + system-wide process listing
- `internal/formatter/` — CLI table + JSON output
- `internal/config/` — YAML config in ~/.config/moor/
- `internal/tui/` — Bubble Tea interactive dashboard
- `internal/forward/` — TCP port forwarding
- `internal/docker/` — Docker container port mappings
- `internal/envfile/` — .env file port extraction

## Conventions

- Interfaces for testability (PortScanner, ProcessManager, etc.)
- Immutable data: return new structs, never mutate
- Graceful degradation: no sudo = partial view, no Docker = skip enrichment
- All public functions documented
