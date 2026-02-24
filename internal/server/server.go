package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"idra/internal/agent"
	"idra/internal/config"
	"idra/web"
)

var (
	Version   = "dev"
	startTime = time.Now()
)

type Server struct {
	httpServer *http.Server
	addr       string
}

func New(cfg config.Config, mgr *agent.Manager) (*Server, error) {
	mux := http.NewServeMux()

	// Static files (embedded)
	staticFS, err := fs.Sub(web.StaticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("embed sub: %w", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Web UI root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := web.StaticFiles.ReadFile("static/index.html")
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// API routes
	mux.HandleFunc("/api/v1/health", handleHealth)
	mux.HandleFunc("/api/v1/config", authMiddleware(handleConfig))
	mux.HandleFunc("/api/v1/status", authMiddleware(handleStatus))

	// Agent API routes
	if mgr != nil {
		mux.HandleFunc("/api/v1/agents", authMiddleware(handleAgents(mgr)))
		// Use a path-based router: /api/v1/agents/{name} and /api/v1/agents/{name}/tasks
		mux.HandleFunc("/api/v1/agents/", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/tasks") {
				handleAgentTasks(mgr)(w, r)
			} else {
				handleAgent(mgr)(w, r)
			}
		}))
	}

	addr, err := resolveAddr(cfg.Port)
	if err != nil {
		return nil, err
	}

	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
		addr: addr,
	}, nil
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) ListenAndServe() error {
	slog.Info("starting HTTP server", "addr", s.addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// resolveAddr tries the configured port, then falls back to 7601-7609.
func resolveAddr(port int) (string, error) {
	candidates := []int{port}
	for p := 7601; p <= 7609; p++ {
		if p != port {
			candidates = append(candidates, p)
		}
	}

	for _, p := range candidates {
		addr := fmt.Sprintf("127.0.0.1:%d", p)
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			ln.Close()
			if p != port {
				// Persist the fallback port
				config.Update(func(c *config.Config) { c.Port = p })
				slog.Warn("port conflict, using fallback", "requested", port, "actual", p)
			}
			return addr, nil
		}
	}
	return "", fmt.Errorf("no available port (tried %d and 7601-7609)", port)
}

// authMiddleware checks the Bearer token for API endpoints.
// Health endpoint is excluded so monitoring tools can probe without auth.
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := config.Get()
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")

		// Allow requests from the web UI (same-origin, no auth header) via
		// Referer check — the UI is served from localhost only.
		referer := r.Header.Get("Referer")
		isSameOrigin := strings.HasPrefix(referer, "http://127.0.0.1:") ||
			strings.HasPrefix(referer, "http://localhost:")

		if token != cfg.BearerToken && !isSameOrigin {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, config.Get())

	case http.MethodPut:
		var c config.Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		updated, err := config.Replace(c)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, updated)

	case http.MethodPatch:
		var partial map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&partial); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		updated, err := config.Update(func(c *config.Config) {
			if v, ok := partial["port"]; ok {
				json.Unmarshal(v, &c.Port)
			}
			if v, ok := partial["auto_open_browser"]; ok {
				json.Unmarshal(v, &c.AutoOpen)
			}
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, updated)

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	cfg := config.Get()
	uptime := time.Since(startTime).Truncate(time.Second)
	writeJSON(w, http.StatusOK, map[string]any{
		"version":  Version,
		"uptime":   uptime.String(),
		"port":     cfg.Port,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
