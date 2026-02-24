# Idra Project - Software Architect Memory

## Project Overview
Idra is a cross-platform Go orchestrator for a fleet of polyglot AI agents. Runs as an OS service (systemd/launchd/Windows SCM) with localhost web UI.

## Tech Stack
- **Language**: Go 1.23, CGO_ENABLED=0 static binaries
- **Module path**: `idra` (local, not URL-based yet)
- **Service mgmt**: github.com/kardianos/service
- **Web UI**: Vanilla HTML/CSS/JS embedded via go:embed
- **Config**: JSON at ~/.idra/config.json (Linux/macOS) or %LOCALAPPDATA%\Idra\ (Windows)
- **Logging**: log/slog (stdlib)
- **Auth**: Auto-generated bearer token, localhost-only binding

## Key Directories
- `cmd/idra/main.go` - CLI entry point (run, service, version, help)
- `internal/config/` - Config load/save/validate with atomic writes
- `internal/server/` - HTTP server + REST API (:8080, fallback 7601-7609)
- `internal/service/` - OS service integration via kardianos/service
- `internal/platform/` - OS-specific code via build tags (paths, browser open)
- `web/static/` - Embedded frontend assets
- `scripts/` - install.sh + install.ps1

## Agent Architecture (Designed Feb 2026)
- **Contract**: proto/agent.proto (gRPC, protobuf) - AgentService with Execute (server-streaming), Ping, Describe RPCs
- **Startup protocol**: Agent binds to port 0, prints JSON line to stdout: `{"protocol":"idra-agent-v1","port":N,"name":"..."}`
- **Agents dir**: agents/python-summarizer/, agents/ts-sentiment/
- **Go orchestrator**: internal/agent/ (registry.go, process.go, grpcclient.go, types.go)
- **Each agent**: manifest.json, self-contained deps, own Dockerfile
- **External API stays REST**; gRPC is internal orchestrator-to-agent only

## Conventions
- No external deps unless justified (decisions.md documents each choice)
- Build tags for platform-specific code (not runtime.GOOS switches)
- Atomic config writes (write .tmp then rename)
- Graceful shutdown with 10s timeout on SIGINT/SIGTERM
