package pb

import (
	"context"
	"io"

	"google.golang.org/grpc"
)

// AgentClient is a gRPC client for the AgentService.
type AgentClient struct {
	cc *grpc.ClientConn
}

// NewAgentClient creates a new AgentService client using the given connection.
func NewAgentClient(cc *grpc.ClientConn) *AgentClient {
	return &AgentClient{cc: cc}
}

var executeDesc = &grpc.StreamDesc{
	StreamName:    "Execute",
	ServerStreams:  true,
}

// Execute calls the Execute RPC and returns a stream of TaskEvents.
func (c *AgentClient) Execute(ctx context.Context, req *TaskRequest) (*ExecuteStream, error) {
	stream, err := c.cc.NewStream(ctx, executeDesc, "/agent.AgentService/Execute",
		grpc.ForceCodec(Codec{}))
	if err != nil {
		return nil, err
	}
	if err := stream.SendMsg(req); err != nil {
		return nil, err
	}
	if err := stream.CloseSend(); err != nil {
		return nil, err
	}
	return &ExecuteStream{stream: stream}, nil
}

// Health calls the Health RPC.
func (c *AgentClient) Health(ctx context.Context) (*HealthResponse, error) {
	resp := &HealthResponse{}
	err := c.cc.Invoke(ctx, "/agent.AgentService/Health", &Empty{}, resp,
		grpc.ForceCodec(Codec{}))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Close closes the underlying connection.
func (c *AgentClient) Close() error {
	return c.cc.Close()
}

// ExecuteStream wraps a gRPC client stream for receiving TaskEvents.
type ExecuteStream struct {
	stream grpc.ClientStream
}

// Recv reads the next TaskEvent from the stream.
// Returns io.EOF when the stream is complete.
func (s *ExecuteStream) Recv() (*TaskEvent, error) {
	ev := &TaskEvent{}
	if err := s.stream.RecvMsg(ev); err != nil {
		return nil, err
	}
	return ev, nil
}

// RecvAll reads all events from the stream until EOF.
func (s *ExecuteStream) RecvAll() ([]*TaskEvent, error) {
	var events []*TaskEvent
	for {
		ev, err := s.Recv()
		if err == io.EOF {
			return events, nil
		}
		if err != nil {
			return events, err
		}
		events = append(events, ev)
	}
}
