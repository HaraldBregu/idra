package agent

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"idra/internal/agent/pb"
)

// State represents the lifecycle state of an agent subprocess.
type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateFailed   State = "failed"
)

// Runner manages the lifecycle of a single agent subprocess.
type Runner struct {
	manifest Manifest
	baseDir  string

	mu     sync.RWMutex
	state  State
	port   int
	client *pb.AgentClient
	cancel context.CancelFunc
	done   chan struct{} // closed when process exits
	err    error
}

// NewRunner creates a runner for the given agent manifest.
func NewRunner(m Manifest, baseDir string) *Runner {
	return &Runner{
		manifest: m,
		baseDir:  baseDir,
		state:    StateStopped,
	}
}

// Start spawns the agent subprocess, reads the port handshake, and connects gRPC.
func (r *Runner) Start(parentCtx context.Context) error {
	r.mu.Lock()
	if r.state == StateRunning || r.state == StateStarting {
		r.mu.Unlock()
		return nil
	}
	r.state = StateStarting
	r.err = nil
	r.done = make(chan struct{})
	r.mu.Unlock()

	ctx, cancel := context.WithCancel(parentCtx)

	workDir := r.manifest.AbsDir(r.baseDir)
	cmd := exec.CommandContext(ctx, r.manifest.Command, r.manifest.Args...)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		r.setFailed(fmt.Errorf("stdout pipe: %w", err))
		return r.err
	}
	cmd.Stderr = &logWriter{name: r.manifest.Name}

	if err := cmd.Start(); err != nil {
		cancel()
		r.setFailed(fmt.Errorf("start command: %w", err))
		return r.err
	}

	r.mu.Lock()
	r.cancel = cancel
	r.mu.Unlock()

	slog.Info("agent process started", "agent", r.manifest.Name, "pid", cmd.Process.Pid)

	// Start monitor goroutine (the only place that calls cmd.Wait)
	go r.monitor(cmd)

	// Read AGENT_PORT=XXXXX from stdout (15s timeout)
	portCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			slog.Debug("agent stdout", "agent", r.manifest.Name, "line", line)
			if strings.HasPrefix(line, "AGENT_PORT=") {
				var port int
				if _, err := fmt.Sscanf(line, "AGENT_PORT=%d", &port); err == nil {
					portCh <- port
					return
				}
			}
		}
		errCh <- fmt.Errorf("agent exited without printing AGENT_PORT")
	}()

	select {
	case port := <-portCh:
		r.mu.Lock()
		r.port = port
		r.mu.Unlock()
		slog.Info("agent port received", "agent", r.manifest.Name, "port", port)
	case err := <-errCh:
		r.setFailed(err)
		return r.err
	case <-time.After(15 * time.Second):
		r.setFailed(fmt.Errorf("timeout waiting for AGENT_PORT"))
		return r.err
	}

	// Connect gRPC client
	addr := fmt.Sprintf("127.0.0.1:%d", r.port)
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		r.setFailed(fmt.Errorf("grpc connect: %w", err))
		return r.err
	}

	r.mu.Lock()
	r.client = pb.NewAgentClient(conn)
	r.state = StateRunning
	r.mu.Unlock()

	slog.Info("agent connected", "agent", r.manifest.Name, "addr", addr)
	return nil
}

// Stop gracefully shuts down the agent subprocess.
func (r *Runner) Stop() {
	r.mu.Lock()
	state := r.state
	r.state = StateStopped // set before cancel so monitor doesn't mark as Failed
	client := r.client
	cancel := r.cancel
	done := r.done
	r.client = nil
	r.cancel = nil
	r.mu.Unlock()

	if client != nil {
		client.Close()
	}
	if cancel != nil {
		cancel() // kills the process via CommandContext
	}

	// Wait for process to finish (monitor goroutine closes done)
	if done != nil && (state == StateRunning || state == StateStarting) {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			slog.Warn("agent stop timed out", "agent", r.manifest.Name)
		}
	}

	slog.Info("agent stopped", "agent", r.manifest.Name)
}

// Execute sends a task to the agent and collects all streamed events.
func (r *Runner) Execute(ctx context.Context, req *pb.TaskRequest) ([]*pb.TaskEvent, error) {
	r.mu.RLock()
	client := r.client
	state := r.state
	r.mu.RUnlock()

	if state != StateRunning || client == nil {
		return nil, fmt.Errorf("agent %s is not running (state: %s)", r.manifest.Name, state)
	}

	stream, err := client.Execute(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("execute on %s: %w", r.manifest.Name, err)
	}

	return stream.RecvAll()
}

// Health pings the agent's Health RPC.
func (r *Runner) Health(ctx context.Context) (*pb.HealthResponse, error) {
	r.mu.RLock()
	client := r.client
	state := r.state
	r.mu.RUnlock()

	if state != StateRunning || client == nil {
		return nil, fmt.Errorf("agent %s is not running", r.manifest.Name)
	}

	return client.Health(ctx)
}

// Status returns the current agent status as a JSON-friendly struct.
func (r *Runner) Status() AgentStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := AgentStatus{
		Name:   r.manifest.Name,
		State:  string(r.state),
		Skills: r.manifest.Skills,
		Port:   r.port,
	}
	if r.err != nil {
		s.Error = r.err.Error()
	}
	return s
}

// Name returns the agent manifest name.
func (r *Runner) Name() string {
	return r.manifest.Name
}

// AgentStatus is the JSON representation of an agent's state.
type AgentStatus struct {
	Name   string   `json:"name"`
	State  string   `json:"state"`
	Skills []string `json:"skills"`
	Port   int      `json:"port,omitempty"`
	Error  string   `json:"error,omitempty"`
}

func (r *Runner) setFailed(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = StateFailed
	r.err = err
	slog.Error("agent failed", "agent", r.manifest.Name, "error", err)
}

// monitor waits for the process to exit. It is the only goroutine that calls cmd.Wait().
func (r *Runner) monitor(cmd *exec.Cmd) {
	err := cmd.Wait()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Signal that the process has exited
	if r.done != nil {
		close(r.done)
	}

	// Only mark as failed if we're still in Running state
	// (Stop() sets state to Stopped before cancelling)
	if r.state == StateRunning {
		r.state = StateFailed
		if err != nil {
			r.err = fmt.Errorf("process exited unexpectedly: %w", err)
		} else {
			r.err = fmt.Errorf("process exited unexpectedly with code 0")
		}
		slog.Warn("agent process exited", "agent", r.manifest.Name, "error", r.err)
	}
}

// logWriter sends agent stderr output to slog.
type logWriter struct {
	name string
}

func (w *logWriter) Write(p []byte) (int, error) {
	lines := strings.Split(strings.TrimRight(string(p), "\n"), "\n")
	for _, line := range lines {
		if line != "" {
			slog.Debug("agent stderr", "agent", w.name, "line", line)
		}
	}
	return len(p), nil
}
