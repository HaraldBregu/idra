package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"idra/internal/agent/pb"
)

// Manager orchestrates all agent runners.
type Manager struct {
	registry *Registry
	runners  map[string]*Runner // agent name → runner
	mu       sync.RWMutex
}

// NewManager creates a manager from a registry.
func NewManager(reg *Registry) *Manager {
	runners := make(map[string]*Runner, len(reg.Agents()))
	for _, m := range reg.Agents() {
		runners[m.Name] = NewRunner(m, reg.BaseDir())
	}
	return &Manager{
		registry: reg,
		runners:  runners,
	}
}

// StartAll starts all registered agents. Errors are logged but don't stop other agents.
func (m *Manager) StartAll(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var wg sync.WaitGroup
	for name, runner := range m.runners {
		wg.Add(1)
		go func(name string, r *Runner) {
			defer wg.Done()
			if err := r.Start(ctx); err != nil {
				slog.Error("failed to start agent", "agent", name, "error", err)
			}
		}(name, runner)
	}
	wg.Wait()
	slog.Info("all agents started", "count", len(m.runners))
}

// StopAll stops all running agents.
func (m *Manager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, runner := range m.runners {
		runner.Stop()
	}
	slog.Info("all agents stopped")
}

// RouteTask finds the agent that handles the given skill and executes the task.
func (m *Manager) RouteTask(ctx context.Context, agentName string, req *pb.TaskRequest) ([]*pb.TaskEvent, error) {
	m.mu.RLock()
	runner, ok := m.runners[agentName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", agentName)
	}

	return runner.Execute(ctx, req)
}

// AllStatuses returns the status of every registered agent.
func (m *Manager) AllStatuses() []AgentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]AgentStatus, 0, len(m.runners))
	for _, runner := range m.runners {
		statuses = append(statuses, runner.Status())
	}
	return statuses
}

// AgentStatus returns the status of a single agent by name.
func (m *Manager) AgentStatus(name string) (AgentStatus, bool) {
	m.mu.RLock()
	runner, ok := m.runners[name]
	m.mu.RUnlock()

	if !ok {
		return AgentStatus{}, false
	}
	return runner.Status(), true
}

// Runner returns the runner for a named agent.
func (m *Manager) Runner(name string) (*Runner, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runners[name]
	return r, ok
}

// Registry returns the underlying registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}
