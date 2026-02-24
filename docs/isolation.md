# Idra — Isolation Model

Idra is designed to be fully isolated at every layer: build, runtime, and data. This document explains how each layer works and why.

---

## Overview

```
┌─────────────────────────────────────────────────────┐
│                   HOST MACHINE                      │
│                                                     │
│  ┌───────────────────────────────────────────────┐  │
│  │              Idra Binary (static)             │  │
│  │  ┌─────────┐ ┌──────────┐ ┌───────────────┐  │  │
│  │  │ Web UI  │ │ HTTP API │ │ Config Engine │  │  │
│  │  │(embedded)│ │(localhost)│ │  (atomic I/O) │  │  │
│  │  └─────────┘ └──────────┘ └───────────────┘  │  │
│  └──────────────────┬────────────────────────────┘  │
│                     │ binds 127.0.0.1 ONLY          │
│                     ▼                               │
│  ┌──────────────────────────┐                       │
│  │  ~/.idra/                │  0700 permissions     │
│  │    config.json (0600)    │  user-owned only      │
│  │    (bearer token inside) │                       │
│  └──────────────────────────┘                       │
│                                                     │
│  Network boundary: NOTHING leaves localhost         │
└─────────────────────────────────────────────────────┘
```

---

## 1. Build Isolation — No Go Required

You do **not** need Go installed on your machine. The build happens inside a container.

### Option A: Docker build (produces a local binary)

```bash
# Build the binary using Docker — outputs ./idra to your project folder
docker compose run --rm builder
```

This mounts your source code into a `golang:1.23-alpine` container, compiles with `CGO_ENABLED=0`, and writes the binary back to your project directory. After this, you have a native static binary — Docker is no longer needed.

### Option B: Docker dev mode (run inside container)

```bash
# Run Idra in dev mode inside a container
docker compose up dev
```

This runs `go run ./cmd/idra run` inside the container with port 8080 forwarded. Useful for quick iteration without installing anything.

### Option C: Multi-stage Dockerfile (minimal production image)

```bash
# Build a minimal container image (just the binary, no OS, no shell)
docker build -t idra .
docker run -p 127.0.0.1:8080:8080 idra run
```

The final image uses `FROM scratch` — it contains nothing except the Idra binary. No shell, no package manager, no OS libraries. Attack surface is essentially zero.

### Option D: Install Go and build natively

```bash
# If you prefer to install Go: https://go.dev/dl/
CGO_ENABLED=0 go build -o idra ./cmd/idra
```

---

## 2. Runtime Isolation — How the Binary Stays Contained

Once built, the Idra binary enforces isolation through four mechanisms:

### 2a. Static binary — zero runtime dependencies

```
CGO_ENABLED=0 → no libc dependency
                 no .dll / .so / .dylib needed
                 no runtime (JVM, Node, Python) needed
                 works on any Linux regardless of glibc version
```

The binary is completely self-contained. Copy it to any machine and it runs. There is nothing to install, patch, or upgrade on the host.

### 2b. Localhost-only network binding

```go
// server.go — the server ONLY binds to 127.0.0.1, never 0.0.0.0
addr := fmt.Sprintf("127.0.0.1:%d", port)
```

- The HTTP server listens on `127.0.0.1` (loopback) exclusively.
- It is **unreachable from the network**. Other machines on the LAN, WAN, or internet cannot connect.
- Even on the same machine, only processes running under the same user context can reach it (plus the bearer token check below).

### 2c. Dedicated data directory with restricted permissions

```
~/.idra/                    ← Linux/macOS
%LOCALAPPDATA%\Idra\        ← Windows

Permissions:
  directory: 0700 (rwx------)  — only the owner can enter
  config:    0600 (rw-------)  — only the owner can read/write
```

- Idra never writes outside its own data directory.
- It does not modify system files, registry keys (beyond service registration), or other applications' data.
- Uninstalling is: stop the service, delete the binary, delete `~/.idra/`. Nothing else to clean up.

### 2d. Bearer token authentication

```
config.json:
  "bearer_token": "a3f8...64 hex chars...b2c1"
```

- A 256-bit random token is generated on first run using `crypto/rand`.
- Every API request (except `/api/v1/health`) must include `Authorization: Bearer <token>`.
- The token is stored in the config file, which has `0600` permissions — only the file owner can read it.
- This prevents other local processes from modifying Idra's configuration without explicit authorization.

### 2e. Unprivileged execution

- The Idra process runs as a regular user, not root/Administrator.
- Service installation may require elevation (to register with systemd/launchd/SCM), but the running service itself does not retain elevated privileges.
- The port (8080, or fallbacks 7601–7609) is above 1024, so no special privileges are needed to bind.

---

## 3. Isolation Boundaries — What CAN and CANNOT happen

| Action | Allowed? | Why |
|---|---|---|
| Remote machine connects to Idra | No | Bound to 127.0.0.1 only |
| Local process reads config without token | No | File is 0600, API requires bearer token |
| Idra writes outside ~/.idra/ | No | All paths are scoped to the data directory |
| Idra opens network connections | No | It only listens, never dials out |
| Idra modifies system files | No | Runs unprivileged, only touches its own dir |
| Idra survives binary deletion | No | Static binary, no installed runtimes |
| User reads/edits config by hand | Yes | It's a plain JSON file the user owns |
| User stops Idra completely | Yes | `Ctrl+C`, `idra service stop`, or kill the process |
| Full uninstall with no traces | Yes | Delete binary + ~/.idra/ directory |

---

## 4. Comparison with Other Isolation Approaches

| Approach | Idra's choice | Alternative | Why not the alternative |
|---|---|---|---|
| Packaging | Static binary | Docker container | Docker is a heavy dependency for a config tool; users may not have it |
| Network | 127.0.0.1 bind | Firewall rules | Firewalls are OS-specific and can be misconfigured; localhost bind is absolute |
| Auth | Bearer token | No auth / session cookies | Other local processes could access the API without auth |
| Data | Dedicated dir | System-wide paths | Harder to clean up, permission conflicts with other software |
| Privileges | Unprivileged | Root/admin | Principle of least privilege — no reason to run elevated |

---

## 5. Docker-Based Development Workflow (recommended if Go is not installed)

```bash
# First time — build the binary
docker compose run --rm builder

# Run locally (the binary is now on your machine)
./idra run

# Or run everything in Docker
docker compose up dev

# Cross-compile all platforms
docker compose run --rm builder sh -c "
  mkdir -p dist
  for os in linux darwin windows; do
    for arch in amd64 arm64; do
      ext=''; [ \$os = windows ] && ext='.exe'
      GOOS=\$os GOARCH=\$arch CGO_ENABLED=0 \
        go build -ldflags '-s -w' -o dist/idra-\$os-\$arch\$ext ./cmd/idra
    done
  done
  ls -lh dist/
"
```

This approach means your host machine needs **only Docker** — no Go, no Make, no build tools. The Go toolchain lives entirely inside the container and is discarded after the build.
