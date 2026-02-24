/**
 * Idra TypeScript/Node.js agent — rule-based sentiment analyzer.
 *
 * Implements the AgentService gRPC contract defined in proto/agent.proto.
 * Uses dynamic proto loading via @grpc/proto-loader (no codegen needed).
 * On startup, binds to a random port and prints AGENT_PORT=XXXXX to stdout.
 */

const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const path = require("path");

// Load proto definition dynamically
const PROTO_PATH = path.resolve(__dirname, "..", "..", "proto", "agent.proto");

const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
  includeDirs: [path.resolve(__dirname, "..", "..", "proto")],
});

const agentProto = grpc.loadPackageDefinition(packageDefinition).agent;

// ---------------------------------------------------------------------------
// Sentiment analysis (rule-based)
// ---------------------------------------------------------------------------

const POSITIVE_WORDS = new Set([
  "good", "great", "excellent", "amazing", "wonderful", "fantastic",
  "awesome", "brilliant", "love", "happy", "joy", "best", "perfect",
  "beautiful", "outstanding", "superb", "pleasant", "delightful",
  "terrific", "marvelous", "glad", "pleased", "thankful", "grateful",
]);

const NEGATIVE_WORDS = new Set([
  "bad", "terrible", "awful", "horrible", "worst", "hate", "angry",
  "sad", "poor", "ugly", "disgusting", "dreadful", "miserable",
  "annoying", "disappointing", "frustrating", "pathetic", "painful",
  "useless", "boring", "failure", "broken", "wrong", "worse",
]);

function analyzeSentiment(text) {
  const words = text.toLowerCase().replace(/[^a-z\s]/g, "").split(/\s+/);
  let positive = 0;
  let negative = 0;

  for (const word of words) {
    if (POSITIVE_WORDS.has(word)) positive++;
    if (NEGATIVE_WORDS.has(word)) negative++;
  }

  const total = positive + negative;
  if (total === 0) {
    return { label: "neutral", score: 0, positive, negative };
  }

  const score = (positive - negative) / total;
  let label;
  if (score > 0.2) label = "positive";
  else if (score < -0.2) label = "negative";
  else label = "neutral";

  return { label, score: Math.round(score * 100) / 100, positive, negative };
}

// ---------------------------------------------------------------------------
// gRPC service implementation
// ---------------------------------------------------------------------------

function execute(call) {
  const req = call.request;
  const taskId = req.taskId || req.task_id || "";
  const skill = req.skill || "";
  const input = req.input || "";

  if (skill !== "sentiment") {
    call.write({
      taskId: taskId,
      type: "error",
      payload: `unknown skill: ${skill}`,
    });
    call.end();
    return;
  }

  const result = analyzeSentiment(input);

  // Progress event
  call.write({
    taskId: taskId,
    type: "progress",
    payload: `Analyzed ${input.split(/\s+/).length} words: ${result.positive} positive, ${result.negative} negative`,
  });

  // Result event
  call.write({
    taskId: taskId,
    type: "result",
    payload: JSON.stringify(result),
  });

  call.end();
}

function health(call, callback) {
  callback(null, { status: "ok", agentName: "ts-sentiment" });
}

// ---------------------------------------------------------------------------
// Server startup
// ---------------------------------------------------------------------------

function main() {
  const server = new grpc.Server();

  server.addService(agentProto.AgentService.service, {
    Execute: execute,
    Health: health,
  });

  server.bindAsync(
    "127.0.0.1:0",
    grpc.ServerCredentials.createInsecure(),
    (err, port) => {
      if (err) {
        console.error("Failed to bind:", err);
        process.exit(1);
      }

      // Handshake: tell the orchestrator which port we're on
      console.log(`AGENT_PORT=${port}`);
    }
  );
}

main();
