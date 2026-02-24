package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"idra/internal/agent"
	"idra/internal/agent/pb"
)

func handleAgents(mgr *agent.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, mgr.AllStatuses())
	}
}

func handleAgent(mgr *agent.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
		// Strip trailing path segments (e.g. /tasks)
		if idx := strings.Index(name, "/"); idx >= 0 {
			name = name[:idx]
		}

		switch r.Method {
		case http.MethodGet:
			status, ok := mgr.AgentStatus(name)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
				return
			}
			writeJSON(w, http.StatusOK, status)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	}
}

func handleAgentTasks(mgr *agent.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		// Extract agent name from /api/v1/agents/{name}/tasks
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent name"})
			return
		}
		agentName := parts[0]

		var body struct {
			Skill    string            `json:"skill"`
			Input    string            `json:"input"`
			Metadata map[string]string `json:"metadata,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		if body.Skill == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "skill is required"})
			return
		}

		req := &pb.TaskRequest{
			TaskId:   generateTaskID(),
			Skill:    body.Skill,
			Input:    body.Input,
			Metadata: body.Metadata,
		}

		events, err := mgr.RouteTask(r.Context(), agentName, req)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"task_id": req.TaskId,
			"events":  events,
		})
	}
}

func generateTaskID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("task-%x", b)
}
