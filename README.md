# llm-compiler

![Go Version](https://img.shields.io/badge/go-1.25+-blue)
![License](https://img.shields.io/badge/license-Apache--2.0-green)
![CI](https://github.com/LiboWorks/llm-compiler/actions/workflows/ci.yml/badge.svg)

**Compile LLM workflows into explicit, deterministic execution graphs.**

`llm-compiler` turns LLM-driven workflows into inspectable, testable, and versionable artifacts — so LLM behavior can be treated like real code, not hidden magic.

Why this exists
---------------
Most LLM applications today suffer from the same problems:
- Execution order is implicit
- Control flow is buried in prompts and glue code
- Failures are hard to reproduce
- Behavior drifts silently over time
- **Your data leaves your machine** — cloud APIs mean your prompts and outputs travel through third-party servers

When something breaks, you guess. When data leaks, you don't even know.

**llm-compiler makes LLM workflows explicit — and keeps them local.**

Instead of opaque prompt chains hitting remote APIs, you get a compiled execution plan that runs entirely on your hardware:
- **Inspect** — see exactly what runs and in what order
- **Diff** — understand what changed between versions
- **Test** — write assertions against deterministic behavior
- **Version** — track workflow changes like code
- **Reason about** — debug failures with clear execution traces
- **Keep data private** — no API calls, no telemetry, no data leaves your machine

This shifts LLM workflows from runtime improvisation to compile-time reasoning — with full local execution and zero data leakage.

Status
------
This project is early-stage and evolving. The core ideas are stable:
- Explicit workflows
- Deterministic compilation
- Inspectable artifacts

Expect APIs to change before v1.0.

Who is this for?
----------------
- **Go developers** building LLM-powered systems
- **Engineers** who care about determinism and debuggability
- **Teams** tired of prompt spaghetti and invisible logic
- **Anyone** who wants LLM workflows to behave like software

It is **not** a no-code tool or a prompt playground.

Supported platforms
-------------------
This project is tested on **macOS**, **Linux (Ubuntu)**, and **Windows**. CI builds and tests run on all three platforms.

Key features
------------
- **100% local execution** – Your data never leaves your machine. No API calls, no cloud dependencies
- **Explicit execution graphs** – See exactly what runs and in what order
- **Deterministic compilation** – Same input → same output, every time
- **Inspectable artifacts** – Debug with clear execution traces and JSON output
- **Modular LLM backends** – llama.cpp included; designed for extensibility
- **Go-first architecture** – Native performance, single binary deployment
- **CLI and library API** – Use `llmc` CLI or import `pkg/llmc` as a Go library
- Cross-workflow synchronization via `wait_for` with optional timeouts
- Shell steps with template substitution using workflow outputs
- Optional subprocess worker mode for concurrent model execution

Quickstart
----------
1. Clone the repo:

```bash
git clone --recurse-submodules https://github.com/LiboWorks/llm-compiler.git
cd llm-compiler
```

2. Build llama.cpp (required for `local_llm` steps):

```bash
./scripts/build-llama.sh
```

The script auto-detects your OS and configures the appropriate backend:
- **macOS**: Metal + Apple BLAS (GPU acceleration)
- **Linux**: CPU backend (use `--cuda` or `--vulkan` for GPU)
- **Windows**: CPU backend via MinGW (use `--cuda` or `--vulkan` for GPU)

3. Build the CLI and run the demo:

```bash
go build ./cmd/llmc
cd demo && ./run-demo.sh
```

📖 **See [demo/README.md](demo/README.md) for a complete walkthrough** with detailed explanations of the output JSON, workflow features, and how channels/contexts work together.

Go Library API
--------------
Use `llm-compiler` programmatically by importing the public API:

```go
import "github.com/LiboWorks/llm-compiler/pkg/llmc"

// Compile a workflow file to a binary
result, err := llmc.CompileFile("workflow.yaml", &llmc.CompileOptions{
    OutputDir: "./build",
})

// Or load and inspect workflows first
workflows, err := llmc.LoadWorkflows("workflow.yaml")

// Build workflows programmatically
wf := llmc.NewWorkflow("my-workflow").
    AddStep(llmc.LLMStep("analyze", "Analyze these items and summarize: {{items}}").
		WithModel("gpt-4").
		WithMaxTokens(1024).
		WithOutput("analysis").
		Build())
```

See `pkg/llmc` for the full API surface.

Public API Stability
--------------------
Only packages under `pkg/` are considered public API.

- Packages under `internal/` are private implementation details
- CLI behavior may change between minor versions
- Public APIs may change during `v0.x`, but breaking changes will be documented

Do not depend on non-`pkg/` packages.

Building with Pro features
--------------------------
This repo supports an optional private `pro` module. To build with Pro features locally use a `go.work` or `replace` to make the private module available and build with `-tags pro`.

Third-Party Dependencies
------------------------
This project integrates the following open-source software:

- **llama.cpp**  
  https://github.com/ggml-org/llama.cpp  
  Licensed under the MIT License  

`llama.cpp` is included as a git submodule and remains under its original license.

License
-------
This project is licensed under the Apache 2.0 License. See [LICENSE](LICENSE) for details.

How to contribute
-----------------
Contributions are welcome, especially around:
- Workflow semantics
- Execution graph formats
- Testing strategies for LLM workflows

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on opening issues and submitting pull requests.

Roadmap
-------
- [ ] Stable public API
- [ ] Additional backend support
- [ ] Example projects
