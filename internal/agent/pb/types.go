// Package pb defines message types for the Idra agent gRPC protocol.
// These are plain Go structs encoded/decoded via a custom protobuf wire-format codec,
// avoiding the need for protoc code generation while remaining wire-compatible
// with agents that use standard protobuf stubs.
package pb

// TaskRequest is sent by the orchestrator to an agent.
type TaskRequest struct {
	TaskId   string            `json:"task_id"`
	Skill    string            `json:"skill"`
	Input    string            `json:"input"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TaskEvent is streamed back from the agent during execution.
type TaskEvent struct {
	TaskId  string `json:"task_id"`
	Type    string `json:"type"`    // "progress", "result", "error"
	Payload string `json:"payload"`
}

// HealthResponse is returned by the agent health check.
type HealthResponse struct {
	Status    string `json:"status"`
	AgentName string `json:"agent_name"`
}

// Empty mirrors google.protobuf.Empty.
type Empty struct{}
