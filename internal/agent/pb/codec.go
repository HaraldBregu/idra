package pb

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// Codec is a gRPC codec that marshals/unmarshals the Idra agent message types
// using standard protobuf wire format. This makes Go clients wire-compatible
// with agents that use protoc-generated stubs (Python, TypeScript, etc.).
type Codec struct{}

func (Codec) Name() string { return "proto" }

func (Codec) Marshal(v any) ([]byte, error) {
	switch m := v.(type) {
	case *TaskRequest:
		return marshalTaskRequest(m), nil
	case *TaskEvent:
		return marshalTaskEvent(m), nil
	case *HealthResponse:
		return marshalHealthResponse(m), nil
	case *Empty:
		return nil, nil
	default:
		return nil, fmt.Errorf("pb.Codec: unsupported type %T", v)
	}
}

func (Codec) Unmarshal(data []byte, v any) error {
	switch m := v.(type) {
	case *TaskRequest:
		return unmarshalTaskRequest(data, m)
	case *TaskEvent:
		return unmarshalTaskEvent(data, m)
	case *HealthResponse:
		return unmarshalHealthResponse(data, m)
	case *Empty:
		return nil
	default:
		return fmt.Errorf("pb.Codec: unsupported type %T", v)
	}
}

// --- TaskRequest: task_id=1, skill=2, input=3, metadata=4 ---

func marshalTaskRequest(m *TaskRequest) []byte {
	var b []byte
	b = appendString(b, 1, m.TaskId)
	b = appendString(b, 2, m.Skill)
	b = appendString(b, 3, m.Input)
	for k, v := range m.Metadata {
		entry := appendString(nil, 1, k)
		entry = appendString(entry, 2, v)
		b = appendBytes(b, 4, entry)
	}
	return b
}

func unmarshalTaskRequest(data []byte, m *TaskRequest) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("invalid tag")
		}
		data = data[n:]
		switch typ {
		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("invalid bytes for field %d", num)
			}
			data = data[n:]
			switch num {
			case 1:
				m.TaskId = string(val)
			case 2:
				m.Skill = string(val)
			case 3:
				m.Input = string(val)
			case 4:
				if m.Metadata == nil {
					m.Metadata = make(map[string]string)
				}
				k, v, err := unmarshalMapEntry(val)
				if err != nil {
					return err
				}
				m.Metadata[k] = v
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("invalid field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}

// --- TaskEvent: task_id=1, type=2, payload=3 ---

func marshalTaskEvent(m *TaskEvent) []byte {
	var b []byte
	b = appendString(b, 1, m.TaskId)
	b = appendString(b, 2, m.Type)
	b = appendString(b, 3, m.Payload)
	return b
}

func unmarshalTaskEvent(data []byte, m *TaskEvent) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("invalid tag")
		}
		data = data[n:]
		switch typ {
		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("invalid bytes for field %d", num)
			}
			data = data[n:]
			switch num {
			case 1:
				m.TaskId = string(val)
			case 2:
				m.Type = string(val)
			case 3:
				m.Payload = string(val)
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("invalid field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}

// --- HealthResponse: status=1, agent_name=2 ---

func marshalHealthResponse(m *HealthResponse) []byte {
	var b []byte
	b = appendString(b, 1, m.Status)
	b = appendString(b, 2, m.AgentName)
	return b
}

func unmarshalHealthResponse(data []byte, m *HealthResponse) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("invalid tag")
		}
		data = data[n:]
		switch typ {
		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("invalid bytes for field %d", num)
			}
			data = data[n:]
			switch num {
			case 1:
				m.Status = string(val)
			case 2:
				m.AgentName = string(val)
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return fmt.Errorf("invalid field %d", num)
			}
			data = data[n:]
		}
	}
	return nil
}

// --- helpers ---

func appendString(b []byte, fieldNum protowire.Number, s string) []byte {
	if s == "" {
		return b
	}
	b = protowire.AppendTag(b, fieldNum, protowire.BytesType)
	b = protowire.AppendBytes(b, []byte(s))
	return b
}

func appendBytes(b []byte, fieldNum protowire.Number, data []byte) []byte {
	b = protowire.AppendTag(b, fieldNum, protowire.BytesType)
	b = protowire.AppendBytes(b, data)
	return b
}

func unmarshalMapEntry(data []byte) (string, string, error) {
	var k, v string
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return "", "", fmt.Errorf("invalid map entry tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return "", "", fmt.Errorf("invalid map entry field")
			}
			data = data[n:]
			continue
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return "", "", fmt.Errorf("invalid map entry bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			k = string(val)
		case 2:
			v = string(val)
		}
	}
	return k, v, nil
}
