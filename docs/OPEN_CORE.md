Open-core architecture for llm-compiler
=====================================

This document explains a recommended open-core layout and developer workflows
for `github.com/LiboWorks/llm-compiler` (OSS) and a private commercial module
`github.com/libochen/llm-compiler-pro` (Pro). It includes code patterns,
build instructions, CI hints, and examples to keep IP separated while making
the developer experience simple.

Goals
- Public core is fully OSS and 'go get'able.
- Private Pro module holds commercial logic and registers dynamic features
  at build/runtime without the public core importing private code.
- Developers can work locally using `go.work` or `replace`.
- Pro features auto-activate only when present and licensed.

1. High-level pattern
- Public repo defines extension interfaces and a registration helper in a
  small package: `internal/pluginapi` (keeps the public API minimal).
- Public core uses `pluginapi` via safe helper functions like
  `pluginapi.EnhanceTextIfAvailable(...)`.
- Private repo implements `pluginapi.ProFeatures` and calls
  `pluginapi.Register()` (typically in `init()` or explicit activation).

2. Why this pattern?
- The public repo never imports private code — preventing accidental
  leakage into public builds.
- The private repo imports the public internal API and calls `Register`.
- Final binaries that include Pro must explicitly add the private module
  (via `go.work`, `replace` or top-level import); OSS users won't have it.

3. Where we added support in this repo
- `internal/pluginapi/pluginapi.go` — minimal interface and registration
  helpers. This is the single hook for Pro code to register.
- `cmd/pro_register.go` (build-tagged `pro`) — optional file that blank
  imports the private register package (only included when building with
  `-tags pro`).

4. Developer workflows

Local development using `go.work` (recommended if you have both repos):

```bash
# workspace directory that contains both repos
cd $HOME/src
git clone git@github.com:libochen/llm-compiler.git
git clone git@github.com:libochen/llm-compiler-pro.git   # private
cd llm-compiler
go work init ../llm-compiler ../llm-compiler-pro
go work use ./llm-compiler ./llm-compiler-pro
cd llm-compiler
go build ./...
```

Using `replace` in `go.mod` for quick experiments:

```bash
# in llm-compiler/go.mod add a temporary replace
replace github.com/libochen/llm-compiler-pro => ../llm-compiler-pro
go build ./...
```

Building OSS-only (public users / CI):

```bash
go build ./...
```

Building a Pro-enabled binary (licensed customers):

Option A — use `go.work` or `replace` (recommended for local builds):

```bash
go work init ./llm-compiler ./llm-compiler-pro
cd llm-compiler
go build -tags pro ./...
```

Option B — top-level binary imports the private register package
and builds normally (CI or release infra must have access to private repo):

```go
// in cmd/llmc-pro/main.go
package main
import (
    _ "github.com/libochen/llm-compiler-pro/register"
    "github.com/LiboWorks/llm-compiler/cmd"
)
func main(){ cmd.Execute() }
```

5. Build tags approach
- We include `cmd/pro_register.go` guarded by `//go:build pro` which
  blank-imports the private register package. Default builds (without
  `-tags pro`) will not require the private repo.

6. Licensing / activation
- The Pro module should validate licenses in `init()` or expose `Activate(key string)`.
- Keep license checks in the Pro module — core remains license-free.

7. CI hints
- Public repo CI must not reference private modules. Use standard
  `go test ./...` in public CI.
- Private CI can run integration tests that import core.
- To build a Pro binary in CI, configure `GOPRIVATE` and credentials, or use
  a self-hosted runner with access to the private repo.

Example GitHub Actions snippet (private-runner or credentials required):

```yaml
name: Build Pro
on: [push]
jobs:
  build-pro:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: 1.21
      - name: Configure GOPRIVATE
        run: |
          git config --global url."git@github.com:".insteadOf "https://github.com/"
          go env -w GOPRIVATE=github.com/libochen/llm-compiler-pro
      - name: Build (pro)
        run: go build -tags pro ./...
```

8. Security considerations
- Do not add the private module to the public `go.mod`.
- Use code reviews and branch protections to avoid accidental adds.

9. Examples
- `internal/pluginapi` (already added) shows the registration pattern.
- `cmd/pro_register.go` (build-tagged) shows how to opt-into Pro at build time.

10. Next steps (optional)
- Add a small `cmd/llmc-pro` wrapper and CI job to the Pro repo.
- Add optional sanitizers and escaping when inserting LLM output into shell commands.
- Add unit tests and lints enforcing no private imports in public modules.

If you'd like, I can scaffold the private repo example locally (`../llm-compiler-pro`) and add a `README.md` and tiny `register` implementation to try the `go.work` flow — say the word and I'll create those files.

11. Cross-workflow coordination (example)

The generator supports running multiple workflows in parallel and synchronising
individual steps using a `wait_for` field on a step. This lets you run
independent producers/consumers concurrently and only block where data is
needed, dramatically reducing latency compared to serial workflows.

Key points:
- `wait_for` format: `"workflowName.stepName"` — the consumer step will wait
  for the producer step with that name and receive the producer's `output`.
- The received value is stored in the waiting step's `ctx.Vars` under the
  same key (`workflowName.stepName`). You can reference it in templates via
  `{{ index . "workflowName.stepName" }}` or shorthand `{{workflowName.stepName}}`.
- `wait_timeout` is an optional integer (seconds) to avoid blocking forever.

Example YAML (two workflows) — producer generates a short value; consumer
waits for it and echoes it. Save this as `example-wait.yaml` and run the
compiler to produce a runnable program.

```yaml
- name: producer
  steps:
    - name: generate
      type: local_llm
      model: meta-llama-3-8b-instruct.Q4_K_M.gguf
      prompt: "Provide a single-line identifier suitable for the consumer"
      max_tokens: 16
      output: produced

- name: consumer
  steps:
    - name: wait_and_echo
      type: shell
      # wait_for references the producer.step by name
      wait_for: "producer.generate"
      # optional timeout (seconds)
      wait_timeout: 10
      # use the received value in a template; index form is safe for dots
      command: 'echo "Received from producer: {{ index . "producer.generate" }}"'
```

Compile & run (example):

```bash
# generate a program into build/
llmc compile example-wait.yaml -o build/
# build will emit a Go program in build/; run it
go run build/producer_*.go
```

The generated program runs both workflows concurrently; the consumer waits
for the producer only where `wait_for` is set. If the producer never sends a
value, the consumer will block until `wait_timeout` elapses (if provided).
