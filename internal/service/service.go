package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/kardianos/service"

	"idra/internal/config"
	"idra/internal/platform"
	"idra/internal/server"
)

type program struct {
	srv *server.Server
}

func (p *program) Start(s service.Service) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	p.srv, err = server.New(cfg)
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
	if p.srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return p.srv.Shutdown(ctx)
	}
	return nil
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
