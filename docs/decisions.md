# Idra — Architecture & Design Decisions

This document explains every significant decision made during the implementation of Idra, the reasoning behind each choice, and what alternatives were considered.

---

## 1. Language: Go

**Decision:** Use Go as the sole implementation language.

**Why:**
- **Trivial cross-compilation.** Setting `GOOS=linux GOARCH=arm64` is all it takes to produce a binary for a different platform. No cross-toolchains, no Docker, no VM needed.
- **Static binaries.** With `CGO_ENABLED=0`, Go produces a single static binary with zero runtime dependencies. No JVM, no .NET runtime, no Python interpreter on the target machine.
- **`go:embed`.** Native support for embedding files into the binary at compile time. The entire web UI ships inside the executable — no separate asset folder to distribute.
- **Excellent standard library.** `net/http` is production-grade, `encoding/json` handles config serialization, `log/slog` provides structured logging, `crypto/rand` generates tokens. Most of the core app uses zero external dependencies.
- **Fast compile times.** A clean build takes seconds, not minutes. This matters for developer iteration speed.

**Alternatives considered:**
- **Rust** — Superior performance and safety guarantees, but slower compile times, steeper learning curve, and more complex cross-compilation (especially for Windows).
- **Node.js / TypeScript** — Would require bundling the Node runtime or using `pkg`/`nexe`, resulting in large binaries (~40-80MB vs Go's ~10MB). Also, no native service management.
- **Python** — Would need PyInstaller or similar for distribution, making isolation much harder. No native embedding of static files.

---

## 2. Service Management: kardianos/service

**Decision:** Use the `github.com/kardianos/service` library for OS service integration.

**Why:**
- **Single API, three platforms.** One `service.Config` struct and one `service.Interface` implementation covers systemd (Linux), launchd (macOS), and Windows SCM (Windows Service Control Manager).
- **Self-registering binary.** The binary itself handles `install`, `uninstall`, `start`, `stop` commands. No separate installer or service definition files needed.
- **Battle-tested.** The library has been stable for years with 4k+ GitHub stars and is used in production by many projects.
- **Minimal surface area.** The library requires implementing just two methods: `Start()` and `Stop()`.

**Alternatives considered:**
- **Manual systemd/launchd/SCM integration** — Would require maintaining three separate code paths with OS-specific APIs. Significantly more code and testing burden.
- **Docker** — Overkill for a single binary. Adds Docker as a dependency and makes localhost access more complex (port mapping, networking).
- **Supervisor/PM2** — External dependency that the user would need to install separately, violating the "single command install" goal.

---

## 3. Web UI: Plain HTML/CSS/JS

**Decision:** Ship a minimal web UI using vanilla HTML, CSS, and JavaScript with no build toolchain.

**Why:**
- **Zero build dependencies for MVP.** No node_modules, no webpack, no vite, no npm. The HTML/CSS/JS files are authored directly and embedded into the binary.
- **Instant startup.** No hydration, no framework initialization. The page loads and is interactive immediately.
- **Replaceable.** The `web/static/` directory is a clean boundary. A future React/Svelte/Vue app can replace it by simply swapping the files — the Go server doesn't care what's in that directory.
- **Small payload.** The entire UI is ~5KB uncompressed. A React app with dependencies would be 100KB+ minimum.

**Alternatives considered:**
- **React/Svelte/Vue** — Would require a build step, increasing project complexity and making contribution harder for backend developers. Not justified for an MVP with one config form.
- **Go templates** — Would couple the UI rendering to the Go code. Harder to iterate on UI independently and impossible to later hand off to a frontend developer.

---

## 4. Config Format: JSON

**Decision:** Store configuration in JSON format at `~/.idra/config.json`.

**Why:**
- **Native Go support.** `encoding/json` is in the standard library. No external dependency needed for serialization.
- **Struct tags.** Go's `json:"field_name"` struct tags map directly between Go types and JSON, with compile-time type safety.
- **API consistency.** The REST API speaks JSON. Using JSON for on-disk config means the same format flows through the entire system — no translation layer between YAML config and JSON API responses.
- **Human-readable.** `json.MarshalIndent` produces well-formatted output that users can hand-edit if needed.

**Alternatives considered:**
- **YAML** — More human-friendly for complex nested config, but requires an external dependency (`gopkg.in/yaml.v3`) and adds potential for subtle indentation bugs.
- **TOML** — Popular in Go ecosystem (used by Hugo, Cargo), but again requires an external dependency and is less common for API-centric apps.
- **Environment variables** — Not suitable for a service that needs to persist config across restarts and expose it via a web UI.

---

## 5. Default Port: 8080

**Decision:** Default to port 8080, with automatic fallback to ports 7601-7609.

**Why:**
- **Convention.** 8080 is the most widely recognized alternative HTTP port. Users intuitively know what it means.
- **Unprivileged.** Ports above 1024 don't require root/admin privileges on any OS.
- **Fallback range.** If 8080 is occupied (common for developers running other services), Idra silently tries 7601-7609. The chosen port is persisted to config so subsequent starts use the same port.
- **7601-7609 range.** Chosen because it's rarely used by well-known services, reducing collision probability.

**Alternatives considered:**
- **Random port** — Would work but creates a discovery problem: how does the user know where the UI is? A fixed default with known fallbacks is predictable.
- **Port 3000** — Common for Node.js apps but would collide frequently for web developers.
- **Port 0 (OS-assigned)** — Same discovery problem as random port.

---

## 6. Module Path: `idra` (local)

**Decision:** Use `idra` as the Go module path (not a full URL like `github.com/org/idra`).

**Why:**
- **Greenfield project.** The repository URL hasn't been established yet. Using a local module path avoids committing to a remote URL prematurely.
- **Easy to update.** When the repo is published, a single find-and-replace changes `idra/internal/...` to `github.com/org/idra/internal/...` across all import statements.
- **Valid for local development.** Go modules work perfectly with non-URL paths for local-only projects.

---

## 7. Isolation Strategy: Self-Contained Static Binary

**Decision:** The binary is fully self-contained with `CGO_ENABLED=0`, binds only to localhost, uses a dedicated data directory, and runs unprivileged.

**Why each aspect matters:**

- **`CGO_ENABLED=0`** — Eliminates dependency on libc. The binary runs on any Linux distribution regardless of glibc version. Also enables true cross-compilation (cgo requires cross-compilers for each target).
- **Localhost-only (`127.0.0.1:8080`)** — The service is a local configuration tool, never a network server. Binding to `0.0.0.0` would expose the config API to the network, which is a security risk.
- **Dedicated data directory (`~/.idra/`)** — Keeps all Idra state in one place. Easy to backup, easy to uninstall (delete one folder). Follows XDG conventions on Linux.
- **Unprivileged execution** — No root/admin required for normal operation. Service installation may need elevation (depending on OS), but the running service drops to user privileges.

---

## 8. Installer Scripts: `install.sh` + `install.ps1`

**Decision:** Two thin shell scripts that handle download, verification, and service setup.

**Why two scripts (not one cross-platform tool):**
- **Minimal dependency.** `curl | bash` works on every Unix system out of the box. `irm | iex` (Invoke-RestMethod piped to Invoke-Expression) works on every modern Windows with PowerShell.
- **Scripts are thin.** All complex logic (service install/start, port detection, config generation) lives in the compiled Go binary. The scripts just download, place, and invoke it.
- **Transparent.** Users can read a 60-line script and understand exactly what it does before running it. A compiled installer is a black box.

**Why not a single cross-platform script:**
- Bash doesn't run natively on Windows (unless WSL is installed, which we can't assume).
- PowerShell doesn't run on most Linux/macOS systems by default.
- Attempting to support both in one script adds complexity for no real benefit.

---

## 9. Bearer Token Authentication

**Decision:** Generate a random bearer token on first run, store it in the config file, require it for API access.

**Why:**
- **Prevents local attacks.** Even though the server binds to localhost, other processes on the same machine could access the API. A bearer token ensures only authorized clients can modify config.
- **Auto-generated.** 32 bytes of `crypto/rand` encoded as hex (64 characters). No user action required — the token is ready at first boot.
- **Stored in config.** The token lives in `~/.idra/config.json` with `0600` permissions. Only the owning user can read it.
- **Web UI exemption.** The web UI is served from the same origin, so it uses a Referer-based check to avoid requiring the user to manually enter a token in the browser.

**Alternatives considered:**
- **No auth** — Risky even on localhost. Any local process or browser tab could hit the API.
- **Session cookies** — More complex to implement and wouldn't help for `curl`/API access.
- **mTLS** — Massive overkill for a localhost-only service.

---

## 10. Project Structure: `cmd/` + `internal/`

**Decision:** Follow the standard Go project layout with `cmd/idra/` for the entry point and `internal/` for private packages.

**Why:**
- **`cmd/idra/main.go`** — Standard Go convention for projects with a single binary. If we later add more binaries (e.g., `cmd/idra-ctl/`), the structure supports it.
- **`internal/`** — Go's built-in access control. Packages under `internal/` cannot be imported by external modules. This is a compile-time guarantee, not just a convention.
- **Package separation** — `config`, `server`, `service`, `platform` are independent concerns. Each package has a clear responsibility and minimal coupling to others.
- **`web/` at root level** — The web assets are a cross-cutting concern, not internal Go logic. Placing `embed.go` in `web/` makes the embed directive discoverable and keeps `internal/` focused on Go logic.

---

## 11. Build Tags for Platform Code

**Decision:** Use Go build tags (`//go:build linux`, `//go:build darwin`, `//go:build windows`) for OS-specific code rather than runtime detection.

**Why:**
- **Compile-time selection.** Only the relevant platform code is compiled into each binary. No dead code from other OSes ships in the final artifact.
- **Type safety.** Each platform file exports the same function signatures (`DataDir()`, `ConfigDir()`, `OpenBrowser()`). If a platform file is missing a function, compilation fails — not a runtime panic.
- **Standard Go practice.** This is how the Go standard library itself handles platform differences.

**Alternatives considered:**
- **`runtime.GOOS` switch** — Would work but includes code for all platforms in every binary and requires a default/fallback case.

---

## 12. Atomic Config Writes

**Decision:** Config is written to a `.tmp` file first, then atomically renamed to the target path.

**Why:**
- **Crash safety.** If the process crashes mid-write, the original config file is still intact. A direct overwrite could leave a corrupted half-written file.
- **Concurrent safety.** Combined with `sync.RWMutex`, this ensures that readers always see a complete, valid config file.
- **Standard pattern.** This write-tmp-then-rename approach is used by virtually every production system that writes config files (systemd, Docker, etcd, etc.).

---

## 13. Graceful Shutdown

**Decision:** The `run` command handles SIGINT/SIGTERM with a 10-second shutdown timeout.

**Why:**
- **Clean resource release.** Active HTTP connections are drained rather than severed. In-progress config saves complete.
- **Service manager integration.** When running as a service, the OS sends SIGTERM (Linux/macOS) or a stop signal (Windows SCM). Handling it gracefully means the service manager reports a clean stop rather than a crash.
- **10-second timeout.** Long enough to finish any reasonable request, short enough to not block system shutdown.

---

## 14. Structured Logging: `log/slog`

**Decision:** Use Go 1.21+'s standard library `log/slog` for all logging.

**Why:**
- **Standard library.** No external dependency. `slog` was specifically designed to be the standard structured logging package for Go.
- **Structured by default.** Key-value pairs (`slog.Info("starting", "port", 8080)`) produce parseable output, useful for debugging and monitoring.
- **Level control.** Built-in support for Debug/Info/Warn/Error levels.
- **Replaceable handler.** The `slog.Handler` interface means we can switch to JSON output for production without changing any call sites.

**Alternatives considered:**
- **`zerolog`/`zap`** — Excellent libraries but add an external dependency for no significant benefit over `slog` in this use case.
