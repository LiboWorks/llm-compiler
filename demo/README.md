# llm-compiler Demo

This folder contains a complete example workflow and its output, demonstrating all key features of `llm-compiler`.

## Quick Start

### Prerequisites

1. **Build llama.cpp** (required for `local_llm` steps):
   ```bash
   # From repo root
   ./scripts/build-llama.sh
   ```

2. **Build the CLI**:
   ```bash
   go build ./cmd/llmc
   ```

3. **Download a GGUF model** (for local LLM inference):
   - Get a quantized model like meta-llama-3-8b-instruct.Q4_K_M.gguf or many different LLM models (each model for each step)
   - Update the `model:` paths in `example.yaml` to point to your downloaded `.gguf` file

### Run the Demo

```bash
cd demo
./run-demo.sh
```

---

## Files in This Demo

| File | Description |
|------|-------------|
| [example.yaml](example.yaml) | Source workflow definition with 3 parallel workflows |
| [output/example.go](output/example.go) | Generated Go source code (with `--keep-source`) |
| [output/example_run.json](output/example_run.json) | Runtime output with contexts, channels, and step results |
| [run-demo.sh](run-demo.sh) | Helper script to compile and run the demo |

---

## Understanding the Output JSON

After running the compiled workflow, `example_run.json` is generated with two main sections:

### 1. `channels` — Step Signals

Each step sends a signal when it completes. The channel key format is:
```
{workflow_index}_{workflow_name}.{workflow_index}_{step_index}/{total_steps}_{step_name}
```

Example:
```json
"channels": {
  "1_producer.1_1/4_generate_data": {
    "err": null,
    "val": "Hello from Producer\n"
  },
  "1_producer.1_3/4_final_output": {
    "err": null,
    "val": "Producer says: Hello from Producer\n | Status: ready\n\n"
  },
  "2_consumer.2_1/3_wait_for_producer": {
    "err": null,
    "val": "Consumer received: Producer says: Hello from Producer\n..."
  }
}
```

- **`val`**: The step's output value (stdout for shell, response for LLM)
- **`err`**: Error message if the step failed, otherwise `null`

Channels enable **cross-workflow synchronization** via `wait_for`. When a consumer workflow uses `wait_for: producer.final_output`, it blocks until that channel receives a value.

### 2. `contexts` — Workflow Variables

Each workflow maintains its own context (key-value store). Variables are set via the `output:` field in steps.

Example:
```json
"contexts": {
  "1_producer": {
    "message": "Hello from Producer\n",
    "status": "ready\n",
    "producer_result": "Producer says: Hello from Producer\n | Status: ready\n\n"
  },
  "2_consumer": {
    "producer.final_output": "Producer says: Hello from Producer\n...",
    "received_data": "Consumer received: ...",
    "consumer_result": "Processed: ..."
  }
}
```

- Variables are scoped to their workflow
- Cross-workflow data is accessed via `wait_for` and stored with the original key (e.g., `producer.final_output`)
- Template substitution `{{variable}}` reads from the workflow's context

---

## How It All Works Together

```
┌─────────────────────────────────────────────────────────────────┐
│                        YAML Workflow                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  producer    │  │  consumer    │  │  conditional │           │
│  │  (4 steps)   │  │  (3 steps)   │  │  (5 steps)   │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ llmc compile
┌─────────────────────────────────────────────────────────────────┐
│                     Generated Go Code                           │
│  - Per-workflow goroutines with sync.WaitGroup                  │
│  - Signal channels for wait_for synchronization                 │
│  - Template rendering for {{variable}} substitution             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ go build
┌─────────────────────────────────────────────────────────────────┐
│                    Standalone Binary                            │
│  - Embedded llama.cpp for local_llm steps                       │
│  - No external dependencies at runtime                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ execute
┌─────────────────────────────────────────────────────────────────┐
│                    Runtime Execution                            │
│                                                                 │
│  producer ──────────────────────────────────────────────►       │
│     │ send(final_output)                                        │
│     ▼                                                           │
│  consumer ◄─── wait_for: producer.final_output                  │
│     │                                                           │
│     ▼                                                           │
│  conditional (runs in parallel)                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    example_run.json                             │
│  {                                                              │
│    "channels": { ... step signals ... },                        │
│    "contexts": { ... workflow variables ... }                   │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Workflow Features Demonstrated

### 1. Shell Commands with Template Substitution
```yaml
- name: final_output
  type: shell
  command: 'echo "Producer says: {{message}} | Status: {{status}}"'
  output: producer_result
```

### 2. Cross-Workflow Synchronization
```yaml
- name: wait_for_producer
  wait_for: producer.final_output    # Blocks until producer completes
  wait_timeout: 10                   # Timeout in seconds
  command: 'echo "Received: {{producer.final_output}}"'
```

### 3. Conditional Execution
```yaml
- name: production_action
  if: "{{mode}} == 'production'"     # Only runs if condition is true
  command: 'echo "Running in PRODUCTION mode"'
```

### 4. Local LLM Inference
```yaml
- name: summarize
  type: local_llm
  model: /path/to/model.gguf
  prompt: 'Summarize: {{producer_result}}'
  max_tokens: 32
  output: summary
```

---

## Notes about Concurrency

- By default the local LLM runtime serializes C-level `Predict` calls to avoid concurrency issues with the ggml/llama C binding. This means multiple `local_llm` steps will be queued when running in-process.
- Use `LLMC_SUBPROCESS=1` to enable subprocess workers; each worker is an isolated process that can load models independently and run in parallel.

---

## Testing Workflows

llm-compiler workflows are testable like any other code. This demo includes example tests in `example_test.go`.

**Run the tests:**
```bash
# From repo root
go test ./demo -v
```

**What the tests verify:**
- `TestWorkflowCompiles` — The workflow YAML compiles without errors
- `TestWorkflowOutputStructure` — JSON output has expected top-level structure
- `TestWorkflowContextValues` — Specific context values are set correctly
- `TestChannelSignaling` — Cross-workflow signals are captured

**Write your own assertions:**
```go
// Load the output JSON
data, _ := os.ReadFile("output/example_run.json")
var output map[string]interface{}
json.Unmarshal(data, &output)

// Assert on contexts
contexts := output["contexts"].(map[string]interface{})
assert.Contains(t, contexts, "1_producer")

// Assert on specific values
producerCtx := contexts["1_producer"].(map[string]interface{})
assert.NotEmpty(t, producerCtx["fetch_data"])
```

---

## Versioning & Diffing Workflows

Because workflows compile to deterministic artifacts, you can track changes over time.

**Example: Comparing workflow versions**

Suppose you have two versions of a workflow:

```yaml
# v1: Simple sequential steps
steps:
  - name: fetch
    type: shell
    command: curl -s https://api.example.com/data
    output: raw_data
  - name: process
    type: shell
    command: echo "{{raw_data}}" | jq '.items'
    output: result
```

```yaml
# v2: Added validation step
steps:
  - name: fetch
    type: shell
    command: curl -s https://api.example.com/data
    output: raw_data
  - name: validate        # NEW
    type: shell
    command: echo "{{raw_data}}" | jq -e '.items | length > 0'
    output: is_valid
  - name: process
    type: shell
    command: echo "{{raw_data}}" | jq '.items'
    output: result
    if: '{{is_valid}} == "true"'  # NEW: conditional
```

**What you can diff:**

1. **YAML source** — `diff workflow_v1.yaml workflow_v2.yaml`
2. **Generated Go code** — `diff v1/example.go v2/example.go` (use `--keep-source`)
3. **Execution output** — `diff v1/example_run.json v2/example_run.json`

This makes questions like "what changed?" and "why did behavior differ?" answerable from artifacts, not guesswork.

---

## Tips

- **Model paths**: Update `model:` in `example.yaml` to your local GGUF file path
- **Parallel LLM**: Use `LLMC_SUBPROCESS=1` for true parallel local_llm execution
- **Debug output**: Use `--keep-source` to inspect generated Go code
- **Skip LLM steps**: Comment out `local_llm` steps to test shell-only workflows quickly
- **Run tests**: Use `go test ./demo -v` to verify workflow behavior
