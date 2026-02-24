package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"idra/internal/agent"
	"idra/internal/config"
	"idra/internal/platform"
	"idra/internal/server"
	svc "idra/internal/service"
)

var version = "dev"

func main() {
	server.Version = version

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "run":
		runForeground()
	case "service":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: idra service <install|uninstall|start|stop>")
			os.Exit(1)
		}
		serviceCmd(os.Args[2])
	case "version", "--version", "-v":
		fmt.Printf("idra %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func runForeground() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("config loaded", "path", config.FilePath(), "port", cfg.Port)

	// Discover agents
	agentsDir := resolveAgentsDir()
	var mgr *agent.Manager
	if agentsDir != "" {
		reg, err := agent.NewRegistry(agentsDir)
		if err != nil {
			slog.Error("failed to create agent registry", "error", err)
			os.Exit(1)
		}
		mgr = agent.NewManager(reg)
	}

	srv, err := server.New(cfg, mgr)
	if err != nil {
		slog.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start agents
	if mgr != nil {
		mgr.StartAll(ctx)
		agent.StartHealthLoop(ctx, mgr, 30*time.Second)
	}

	// Open browser
	if cfg.AutoOpen {
		go func() {
			time.Sleep(300 * time.Millisecond)
			url := "http://" + srv.Addr()
			slog.Info("opening browser", "url", url)
			if err := platform.OpenBrowser(url); err != nil {
				slog.Warn("could not open browser", "error", err)
			}
		}()
	}

	// Start server in background
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	slog.Info("idra is running", "addr", srv.Addr(), "version", version)
	fmt.Printf("\n  Idra is running at http://%s\n  Press Ctrl+C to stop.\n\n", srv.Addr())

	// Wait for shutdown signal or server error
	select {
	case <-ctx.Done():
		slog.Info("shutting down...")

		// Stop agents first
		if mgr != nil {
			mgr.StopAll()
		}

		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	case err := <-errCh:
		if err != nil {
			slog.Error("server error", "error", err)
			if mgr != nil {
				mgr.StopAll()
			}
			os.Exit(1)
		}
	}

	slog.Info("stopped")
}

// resolveAgentsDir finds the agents/ directory. Checks next to the executable
// first (production), then the current working directory (development).
func resolveAgentsDir() string {
	// Next to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "agents")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			slog.Info("agents directory found (next to binary)", "dir", dir)
			return dir
		}
	}

	// Current working directory
	cwd, err := os.Getwd()
	if err == nil {
		dir := filepath.Join(cwd, "agents")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			slog.Info("agents directory found (cwd)", "dir", dir)
			return dir
		}
	}

	slog.Info("no agents directory found")
	return ""
}

func serviceCmd(action string) {
	var err error
	switch action {
	case "install":
		err = svc.Install()
		if err == nil {
			fmt.Println("Service installed successfully.")
		}
	case "uninstall":
		err = svc.Uninstall()
		if err == nil {
			fmt.Println("Service uninstalled successfully.")
		}
	case "start":
		err = svc.Start()
		if err == nil {
			fmt.Println("Service started.")
		}
	case "stop":
		err = svc.Stop()
		if err == nil {
			fmt.Println("Service stopped.")
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown service action: %s\n", action)
		fmt.Fprintln(os.Stderr, "Usage: idra service <install|uninstall|start|stop>")
		os.Exit(1)
	}
	if err != nil {
		slog.Error("service command failed", "action", action, "error", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Idra — AI Agent Fleet Orchestrator

Usage:
  idra run                    Run in foreground (dev mode)
  idra service install        Install as OS service
  idra service uninstall      Uninstall the OS service
  idra service start          Start the OS service
  idra service stop           Stop the OS service
  idra version                Print version
  idra help                   Print this help`)
}
