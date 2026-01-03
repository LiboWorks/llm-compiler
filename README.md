# llm-compiler

![Go Version](https://img.shields.io/badge/go-1.25+-blue)
![License](https://img.shields.io/badge/license-Apache--2.0-green)
![CI](https://github.com/LiboWorks/llm-compiler/actions/workflows/ci.yml/badge.svg)

`llm-compiler` is a Go library and CLI that compiles LLM workflow definitions into standalone binaries with embedded local inference. Use it as a **CLI tool** or as a **Go library** to build CLI tools, local agents, and offline edge deployments.

> **Focus:** Local inference, modular backends, and production-oriented design.

Who is this for?
----------------
- **Go developers** working with local or small LLMs
- **CLI/service builders** embedding LLMs into command-line tools or backend services
- **Performance-focused developers** who prefer native performance over Python stacks

Supported platforms
-------------------
This project is tested on **macOS**, **Linux (Ubuntu)**, and **Windows**. CI builds and tests run on all three platforms.

Key features
------------
- **Modular LLM backends** – llama.cpp included via submodule; designed for extensibility
- **Go-first architecture** – Native performance, single binary deployment
- **CLI and library API** – Use the `llmc` CLI or import `pkg/llmc` as a Go library
- **Workflow compilation** – Compile YAML workflows into standalone Go binaries
- Cross-workflow synchronization via `wait_for` with optional timeouts and fail-fast error propagation
- Shell steps with template substitution using workflow outputs
- Optional subprocess worker mode (`LLMC_SUBPROCESS=1`) for true concurrent model execution
- Integration test harness that compiles example workflows and persists outputs for debugging in CI

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

Options:
```bash
./scripts/build-llama.sh --clean    # Clean build
./scripts/build-llama.sh --cuda     # Enable CUDA (Linux/Windows)
./scripts/build-llama.sh --vulkan   # Enable Vulkan (Linux/Windows)
```

<details>
<summary>Manual build instructions (if script doesn't work)</summary>

**macOS (Metal backend):**
```bash
cd third_party/llama.cpp
mkdir -p build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_SHARED_LIBS=OFF \
  -DGGML_METAL=ON \
  -DGGML_BLAS=ON \
  -DGGML_BLAS_VENDOR=Apple
cmake --build . --config Release -j$(sysctl -n hw.ncpu)
cd ../../..
```

**Linux (Ubuntu, CPU backend):**
```bash
cd third_party/llama.cpp
mkdir -p build && cd build
cmake .. -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_SHARED_LIBS=OFF \
  -DGGML_METAL=OFF \
  -DGGML_BLAS=OFF \
  -DLLAMA_CURL=OFF
cmake --build . --config Release -j$(nproc)
cd ../../..
```

**Windows (CPU backend with MinGW):**
```powershell
cd third_party/llama.cpp
mkdir -p build; cd build
cmake .. -G "MinGW Makefiles" -DCMAKE_BUILD_TYPE=Release `
  -DBUILD_SHARED_LIBS=OFF `
  -DGGML_METAL=OFF `
  -DGGML_BLAS=OFF `
  -DGGML_OPENMP=OFF `
  -DLLAMA_CURL=OFF `
  -DLLAMA_BUILD_COMMON=OFF
cmake --build . --config Release -j $env:NUMBER_OF_PROCESSORS
cd ../../..
```
</details>

3. Build the CLI:

```bash
go build ./cmd/llmc
```

4. Compile your workflows (example):

```bash
./llmc compile -i example.yaml -o ./build
```

5. Run the generated program:

```bash
# Run in-process (LLM calls are serialized):
./build/workflows
# Or run with subprocess workers for true concurrency:
LLMC_SUBPROCESS=1 ./build/workflows
```

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

Notes about concurrency
-----------------------
- By default the local LLM runtime serializes C-level `Predict` calls to avoid concurrency issues with the ggml/llama C binding. This means multiple `local_llm` steps will be queued when running in-process.
- Use `LLMC_SUBPROCESS=1` to enable subprocess workers; each worker is an isolated process that can load models independently and run in parallel.

Building with Pro features
--------------------------
This repo supports an optional private `pro` module. To build with Pro features locally use a `go.work` or `replace` to make the private module available and build with `-tags pro`.

Third-Party Dependencies
------------------------
This project integrates the following open-source software:

- **llama.cpp**  
  https://github.com/ggml-org/llama.cpp  
  Licensed under the MIT License  
  Copyright (c) 2023–2024 The ggml authors

`llama.cpp` is included as a git submodule and remains under its original license.

License
-------
This project is licensed under the Apache 2.0 License. See [LICENSE](LICENSE) for details.

How to contribute
-----------------
See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on opening issues and submitting pull requests.

Roadmap
-------
- [ ] Stable public API
- [ ] Additional backend support
- [ ] Example projects
