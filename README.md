# Idra

A cross-platform background service that runs a fleet of AI Agents, managed through a localhost web UI.

Idra installs with a single command, runs isolated as an OS service, and exposes a local dashboard for configuring and orchestrating your AI agents — no cloud dependency, no exposed ports, everything stays on your machine.

## Features

- **Single binary** — one static executable, zero runtime dependencies
- **Cross-platform** — Linux, macOS, and Windows (amd64 + arm64)
- **Runs as a service** — systemd, launchd, or Windows SCM via a single API
- **Localhost web UI** — configure and monitor from your browser at `http://127.0.0.1:8080`
- **REST API** — programmatic access to all configuration and status endpoints
- **Fully isolated** — binds to localhost only, bearer token auth, dedicated data directory
- **One-command install** — `curl | sh` on Unix, `irm | iex` on Windows

## Quick Start

### Install (Unix)

```bash
curl -fsSL https://raw.githubusercontent.com/HaraldBregu/idra/master/scripts/install.sh | bash
```

### Install (Windows)

```powershell
irm https://raw.githubusercontent.com/HaraldBregu/idra/master/scripts/install.ps1 | iex
```

### Run in dev mode

```bash
# With Go installed
make dev

# With Docker only (no Go needed)
docker compose run --rm builder && ./idra run
```

This starts Idra in the foreground and opens `http://127.0.0.1:8080` in your browser.

## Architecture

```
[Install Script] → downloads binary → [idra service install] → [OS Service Manager]
                                                                        |
                                                                  starts idra
                                                                        |
                                                            [HTTP Server :8080]
                                                            /                  \
                                                  GET /  (Web UI)      /api/v1/* (REST API)
                                                            \                  /
                                                        [Config Store ~/.idra/config.json]
```

## API

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/` | Web UI |
| `GET` | `/api/v1/health` | Health check (no auth required) |
| `GET` | `/api/v1/config` | Read current configuration |
| `PUT` | `/api/v1/config` | Replace configuration |
| `PATCH` | `/api/v1/config` | Partial config update |
| `GET` | `/api/v1/status` | Runtime info (version, uptime, port) |

## CLI

```
idra run                    Run in foreground (dev mode)
idra service install        Install as OS service
idra service start          Start the OS service
idra service stop           Stop the OS service
idra service uninstall      Remove the OS service
idra version                Print version
idra help                   Show help
```

## Documentation

| Document | Description |
|---|---|
| [docs/development.md](docs/development.md) | Local dev setup, building, testing, debugging |
| [docs/isolation.md](docs/isolation.md) | Isolation model and Docker-based workflows |
| [docs/decisions.md](docs/decisions.md) | Architecture decisions and rationale |

## License

MIT
