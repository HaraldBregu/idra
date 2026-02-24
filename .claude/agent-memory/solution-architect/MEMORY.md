# Idra Project - Solution Architect Memory

## Project Overview
Idra is a Go-based orchestrator managing a fleet of AI agents (Python, TypeScript, Go).
- Module: `idra`, Go 1.23
- Dependency: `github.com/kardianos/service` for OS service integration
- Config stored at platform-specific paths (Windows: `%LOCALAPPDATA%/Idra`)

## Architecture
- Go HTTP server with REST API + embedded web UI
- Agent fleet: agents are subprocesses communicating via gRPC
- Idra = gRPC client, each Agent = gRPC server (agents expose capabilities)
- Port assignment: orchestrator assigns from pool (9100-9199) via `IDRA_GRPC_PORT` env var
- Discovery: file-based manifests at `{DataDir}/agents/{name}/agent.json`

## Key File Locations
- `cmd/idra/main.go` - Entry point, CLI commands (run, service, version)
- `internal/server/server.go` - HTTP server, routes, auth middleware
- `internal/config/config.go` - Config with mutex-protected read/write, auto-generated bearer token
- `internal/service/service.go` - OS service integration via kardianos/service
- `internal/platform/` - Platform-specific paths and browser opening
- `web/embed.go` - Embedded static files
- `proto/agent/v1/agent.proto` - gRPC contract (designed, not yet created)
- `internal/agent/` - Agent fleet layer (designed, not yet created)

## Design Decisions (Agent Fleet)
- Proto: `bytes` for input/output (JSON by convention, proto-stable)
- Manifest for spawn config, Handshake RPC for runtime capabilities
- Supervisor pattern with max restart count, then FAILED state
- REST-to-gRPC bridge: SSE for streaming, JSON for unary
- Shutdown order: agents first, then HTTP server

## Patterns
- Config uses sync.RWMutex with atomic file writes (tmp + rename)
- Server uses port fallback (tries 8080, then 7601-7609)
- Auth: Bearer token + same-origin Referer check for web UI
