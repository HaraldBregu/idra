package agent

import (
	"context"
	"log/slog"
	"time"
)

// StartHealthLoop runs a background goroutine that pings every running agent
// every interval (typically 30s). Agents that fail the health check are
// marked as Failed.
func StartHealthLoop(ctx context.Context, mgr *Manager, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkAll(ctx, mgr)
			}
		}
	}()
}

func checkAll(ctx context.Context, mgr *Manager) {
	mgr.mu.RLock()
	runners := make([]*Runner, 0, len(mgr.runners))
	for _, r := range mgr.runners {
		runners = append(runners, r)
	}
	mgr.mu.RUnlock()

	for _, r := range runners {
		r.mu.RLock()
		state := r.state
		r.mu.RUnlock()

		if state != StateRunning {
			continue
		}

		hctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := r.Health(hctx)
		cancel()

		if err != nil {
			slog.Warn("health check failed", "agent", r.Name(), "error", err)
			r.setFailed(err)
		}
	}
}
