package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/kardianos/service"

	"idra/internal/agent"
	"idra/internal/config"
	"idra/internal/platform"
	"idra/internal/server"
)

type program struct {
	srv *server.Server
	mgr *agent.Manager
	ctx context.Context
	cancel context.CancelFunc
}

func (p *program) Start(s service.Service) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())

	// Discover and start agents
	agentsDir := resolveAgentsDir()
	if agentsDir != "" {
		reg, err := agent.NewRegistry(agentsDir)
		if err != nil {
			slog.Warn("agent registry error", "error", err)
		} else {
			p.mgr = agent.NewManager(reg)
			p.mgr.StartAll(p.ctx)
			agent.StartHealthLoop(p.ctx, p.mgr, 30*time.Second)
		}
	}

	p.srv, err = server.New(cfg, p.mgr)
	if err != nil {
		return err
	}

	if cfg.AutoOpen {
		go func() {
			// Small delay to let the server start
			time.Sleep(500 * time.Millisecond)
			url := "http://" + p.srv.Addr()
			if err := platform.OpenBrowser(url); err != nil {
				slog.Warn("could not open browser", "error", err)
			}
		}()
	}

	go func() {
		if err := p.srv.ListenAndServe(); err != nil {
			slog.Error("http server error", "error", err)
		}
	}()

	return nil
}

func (p *program) Stop(s service.Service) error {
	// Stop agents first
	if p.mgr != nil {
		p.mgr.StopAll()
	}
	if p.cancel != nil {
		p.cancel()
	}
	if p.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return p.srv.Shutdown(ctx)
	}
	return nil
}

// resolveAgentsDir finds agents/ next to executable, then CWD.
func resolveAgentsDir() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "agents")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	cwd, err := os.Getwd()
	if err == nil {
		dir := filepath.Join(cwd, "agents")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}

func NewService() (service.Service, error) {
	svcConfig := &service.Config{
		Name:        "idra",
		DisplayName: "Idra",
		Description: "Idra background service with web UI",
	}

	prg := &program{}
	return service.New(prg, svcConfig)
}

func Install() error {
	s, err := NewService()
	if err != nil {
		return err
	}
	return s.Install()
}

func Uninstall() error {
	s, err := NewService()
	if err != nil {
		return err
	}
	return s.Uninstall()
}

func Start() error {
	s, err := NewService()
	if err != nil {
		return err
	}
	return s.Start()
}

func Stop() error {
	s, err := NewService()
	if err != nil {
		return err
	}
	return s.Stop()
}

func RunInteractive() error {
	s, err := NewService()
	if err != nil {
		return err
	}
	return s.Run()
}
