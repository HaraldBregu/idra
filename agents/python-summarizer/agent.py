"""Idra Python agent — extractive text summarizer.

Implements the AgentService gRPC contract defined in proto/agent.proto.
On startup, binds to a random port and prints AGENT_PORT=XXXXX to stdout
so the orchestrator can discover it.

Uses manual protobuf wire format encoding so the only dependency is grpcio.
"""

import re
import sys
from concurrent import futures

import grpc


# ---------------------------------------------------------------------------
# Minimal protobuf wire format codec
# ---------------------------------------------------------------------------

def _encode_varint(value):
    buf = bytearray()
    while value > 0x7F:
        buf.append((value & 0x7F) | 0x80)
        value >>= 7
    buf.append(value & 0x7F)
    return bytes(buf)


def _decode_varint(data, pos):
    result, shift = 0, 0
    while pos < len(data):
        b = data[pos]
        result |= (b & 0x7F) << shift
        pos += 1
        if not (b & 0x80):
            return result, pos
        shift += 7
    raise ValueError("truncated varint")


def _encode_string_field(field_num, value):
    if not value:
        return b""
    tag = _encode_varint((field_num << 3) | 2)
    raw = value.encode("utf-8")
    return tag + _encode_varint(len(raw)) + raw


def _decode_fields(data):
    """Decode protobuf bytes into {field_num: [values]}."""
    fields = {}
    pos = 0
    while pos < len(data):
        tag, pos = _decode_varint(data, pos)
        field_num = tag >> 3
        wire_type = tag & 0x07
        if wire_type == 2:  # length-delimited
            length, pos = _decode_varint(data, pos)
            value = data[pos : pos + length]
            pos += length
        elif wire_type == 0:  # varint
            value, pos = _decode_varint(data, pos)
        else:
            raise ValueError(f"unsupported wire type {wire_type}")
        fields.setdefault(field_num, []).append(value)
    return fields


# ---------------------------------------------------------------------------
# Message serialization / deserialization
# ---------------------------------------------------------------------------

class TaskRequest:
    __slots__ = ("task_id", "skill", "input", "metadata")

    def __init__(self, task_id="", skill="", input_="", metadata=None):
        self.task_id = task_id
        self.skill = skill
        self.input = input_
        self.metadata = metadata or {}

    @classmethod
    def from_bytes(cls, data):
        fields = _decode_fields(data)
        req = cls()
        for v in fields.get(1, []):
            req.task_id = v.decode("utf-8") if isinstance(v, bytes) else str(v)
        for v in fields.get(2, []):
            req.skill = v.decode("utf-8") if isinstance(v, bytes) else str(v)
        for v in fields.get(3, []):
            req.input = v.decode("utf-8") if isinstance(v, bytes) else str(v)
        # field 4 = map entries (each is a sub-message with key=1, value=2)
        for entry_bytes in fields.get(4, []):
            if isinstance(entry_bytes, bytes):
                entry_fields = _decode_fields(entry_bytes)
                k = entry_fields.get(1, [b""])[0]
                v = entry_fields.get(2, [b""])[0]
                req.metadata[k.decode("utf-8")] = v.decode("utf-8")
        return req


class TaskEvent:
    __slots__ = ("task_id", "type", "payload")

    def __init__(self, task_id="", type_="", payload=""):
        self.task_id = task_id
        self.type = type_
        self.payload = payload

    def to_bytes(self):
        return (
            _encode_string_field(1, self.task_id)
            + _encode_string_field(2, self.type)
            + _encode_string_field(3, self.payload)
        )


class HealthResponse:
    __slots__ = ("status", "agent_name")

    def __init__(self, status="", agent_name=""):
        self.status = status
        self.agent_name = agent_name

    def to_bytes(self):
        return (
            _encode_string_field(1, self.status)
            + _encode_string_field(2, self.agent_name)
        )


# ---------------------------------------------------------------------------
# Service implementation
# ---------------------------------------------------------------------------

def _split_sentences(text):
    parts = re.split(r"(?<=[.!?])\s+", text.strip())
    return [s for s in parts if s]


class AgentServicer:
    def Execute(self, request_bytes, context):
        req = TaskRequest.from_bytes(request_bytes)

        if req.skill != "summarize":
            yield TaskEvent(
                task_id=req.task_id, type_="error", payload=f"unknown skill: {req.skill}"
            ).to_bytes()
            return

        sentences = _split_sentences(req.input)
        n = min(3, len(sentences))
        summary = " ".join(sentences[:n])
        if not summary:
            summary = req.input[:200] if len(req.input) > 200 else req.input

        yield TaskEvent(
            task_id=req.task_id,
            type_="progress",
            payload=f"Extracted {n} sentences from {len(sentences)} total",
        ).to_bytes()

        yield TaskEvent(
            task_id=req.task_id, type_="result", payload=summary
        ).to_bytes()

    def Health(self, request_bytes, context):
        return HealthResponse(status="ok", agent_name="python-summarizer").to_bytes()


# ---------------------------------------------------------------------------
# gRPC server
# ---------------------------------------------------------------------------

def _identity(x):
    """Pass-through serializer — we handle encoding ourselves."""
    return x


class _Handler(grpc.GenericRpcHandler):
    def __init__(self, servicer):
        self._servicer = servicer
        self._methods = {
            "/agent.AgentService/Execute": grpc.unary_stream_rpc_method_handler(
                servicer.Execute,
                request_deserializer=_identity,
                response_serializer=_identity,
            ),
            "/agent.AgentService/Health": grpc.unary_unary_rpc_method_handler(
                servicer.Health,
                request_deserializer=_identity,
                response_serializer=_identity,
            ),
        }

    def service(self, handler_call_details):
        return self._methods.get(handler_call_details.method)


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    server.add_generic_rpc_handlers([_Handler(AgentServicer())])

    port = server.add_insecure_port("127.0.0.1:0")
    server.start()

    # Handshake: tell the orchestrator which port we're on
    print(f"AGENT_PORT={port}", flush=True)

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        server.stop(grace=5)


if __name__ == "__main__":
    serve()
