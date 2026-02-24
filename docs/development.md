# Idra — Local Development Guide

## Prerequisites

**Option A — Go installed locally:**
- **Go 1.23+** — [Download](https://go.dev/dl/)
- **Make** (optional) — comes with Xcode CLI tools on macOS, `build-essential` on Linux, or `choco install make` on Windows
- **Git** — for version tagging

**Option B — Docker only (no Go needed):**
- **Docker** with Docker Compose — [Download](https://docs.docker.com/get-docker/)

> If you don't have Go installed, see [docs/isolation.md](isolation.md) for the full Docker-based workflow. Quick start:
> ```bash
> docker compose run --rm builder   # builds the binary into your project folder
> ./idra run                        # run it natively — no Go needed after this
> ```

Verify your setup:

```bash
# Option A
go version    # should print go1.23 or later

# Option B
docker --version
```

---

## Getting Started

### 1. Clone and install dependencies

```bash
git clone <repo-url> idra
cd idra
go mod download
```

### 2. Build

```bash
# With Make
make build

# Without Make
CGO_ENABLED=0 go build -o idra ./cmd/idra
```

This produces the `idra` binary (or `idra.exe` on Windows) in the project root.

### 3. Run in dev mode (foreground)

```bash
# With Make
make dev

# Without Make
./idra run
```

This will:
- Load (or create) config at `~/.idra/config.json`
- Generate a bearer token if one doesn't exist
- Start the HTTP server on `127.0.0.1:8080`
- Open your browser automatically
- Print the address to the terminal
- Stay in the foreground — `Ctrl+C` to stop

You should see output like:

```
time=2026-02-24T12:00:00.000+01:00 level=INFO msg="config loaded" path=/home/you/.idra/config.json port=8080
time=2026-02-24T12:00:00.000+01:00 level=INFO msg="starting HTTP server" addr=127.0.0.1:8080
time=2026-02-24T12:00:00.300+01:00 level=INFO msg="opening browser" url=http://127.0.0.1:8080

  Idra is running at http://127.0.0.1:8080
  Press Ctrl+C to stop.
```

---

## Debugging

### View logs

All logs go to stderr with structured key-value pairs. In dev mode (foreground), they print directly to your terminal.

### Test the API with curl

```bash
# Health check (no auth required)
curl http://127.0.0.1:8080/api/v1/health

# Read config (requires bearer token)
# Find your token in ~/.idra/config.json
TOKEN=$(cat ~/.idra/config.json | grep bearer_token | cut -d'"' -f4)

curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8080/api/v1/config

# Read status
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8080/api/v1/status

# Update config (PUT — full replace)
curl -X PUT \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"port": 8080, "auto_open_browser": false}' \
  http://127.0.0.1:8080/api/v1/config

# Partial update (PATCH)
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"auto_open_browser": false}' \
  http://127.0.0.1:8080/api/v1/config
```

On Windows (PowerShell), use:

```powershell
# Health check
Invoke-RestMethod http://127.0.0.1:8080/api/v1/health

# Read config
$token = (Get-Content "$env:LOCALAPPDATA\Idra\config.json" | ConvertFrom-Json).bearer_token
$headers = @{ Authorization = "Bearer $token" }

Invoke-RestMethod http://127.0.0.1:8080/api/v1/config -Headers $headers

# Update config
Invoke-RestMethod http://127.0.0.1:8080/api/v1/config -Method Put -Headers $headers `
  -ContentType "application/json" `
  -Body '{"port": 8080, "auto_open_browser": false}'
```

### Port conflicts

If port 8080 is already in use, Idra automatically tries 7601–7609 and logs a warning:

```
level=WARN msg="port conflict, using fallback" requested=8080 actual=7601
```

The fallback port is saved to config so the next restart uses the same port.

### Reset config

Delete the config file to start fresh — a new one (with a new bearer token) will be created on next run:

```bash
# Linux/macOS
rm ~/.idra/config.json

# Windows
del %LOCALAPPDATA%\Idra\config.json
```

### Disable auto-open browser

If the browser opening on every `idra run` is annoying during development:

```bash
# Edit config directly
# Set "auto_open_browser": false in ~/.idra/config.json

# Or via the API
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"auto_open_browser": false}' \
  http://127.0.0.1:8080/api/v1/config
```

---

## Running Tests

```bash
# With Make
make test

# Without Make
go test ./...

# Verbose
go test -v ./...

# Single package
go test -v ./internal/config/...
```

---

## Cross-Compilation

Build binaries for all supported platforms:

```bash
make cross
```

This produces six binaries in `dist/`:

```
dist/
  idra-linux-amd64
  idra-linux-arm64
  idra-darwin-amd64
  idra-darwin-arm64
  idra-windows-amd64.exe
  idra-windows-arm64.exe
```

To build for a single target manually:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o idra-linux-arm64 ./cmd/idra
```

---

## Service Mode (local testing)

You can test service install/uninstall locally. This registers Idra with your OS service manager.

```bash
# Install as service (may require sudo/admin)
./idra service install

# Start the service
./idra service start

# Check it's running — health endpoint should respond
curl http://127.0.0.1:8080/api/v1/health

# Stop the service
./idra service stop

# Remove the service registration
./idra service uninstall
```

**Note:** On Linux this creates a systemd unit, on macOS a launchd plist, on Windows a Windows Service. You may need elevated privileges to install/uninstall.

---

## Project Layout Reference

```
cmd/idra/main.go               CLI entry point
internal/config/config.go       Config load/save/validate
internal/server/server.go       HTTP server + REST API
internal/service/service.go     OS service integration
internal/platform/paths_*.go    OS-specific paths (build tags)
internal/platform/browser_*.go  OS-specific browser open (build tags)
web/embed.go                    go:embed directive
web/static/                     HTML/CSS/JS served by the binary
scripts/                        One-command installers
docs/                           Documentation
Makefile                        Build targets
```

---

## Common Issues

| Problem | Solution |
|---|---|
| `go: command not found` | Install Go from https://go.dev/dl/ and add it to your PATH |
| Port 8080 already in use | Idra auto-falls back to 7601–7609. Or stop the other process. |
| `permission denied` on service install | Run with `sudo` (Linux/macOS) or as Administrator (Windows) |
| Config file corrupted | Delete `~/.idra/config.json` and restart — a fresh one will be created |
| Browser doesn't open | Set `auto_open_browser: true` in config, or open `http://127.0.0.1:8080` manually |
| `make: command not found` | Use the `go build` commands directly (shown above for each target) |
