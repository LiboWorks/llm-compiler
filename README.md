# llm-compiler

![Go Version](https://img.shields.io/badge/go-1.25+-blue)
![License](https://img.shields.io/badge/license-Apache--2.0-green)
![CI](https://github.com/LiboWorks/llm-compiler/actions/workflows/ci.yml/badge.svg)

A Go-based compiler and runtime for integrating small LLMs into systems like CLI tools, agents, and edge deployments.

> **Focus:** Local inference, modular backends, and production-oriented design.

`llm-compiler` compiles multi-document YAML workflow definitions into a native Go program that orchestrates shell commands and local LLM inference via `llama.cpp`.

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
- **CLI integration** – Built with Cobra for seamless command-line workflows
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

3. Build the project:

```bash
go build ./...
```

4. Compile your workflows (example):

```bash
go run main.go compile example.yaml -o ./build
```

5. Run the generated program:

```bash
# Run in-process (LLM calls are serialized):
./build/workflows
# Or run with subprocess workers for true concurrency:
LLMC_SUBPROCESS=1 ./build/workflows
```

Notes about concurrency
-----------------------
- By default the local LLM runtime serializes C-level `Predict` calls to avoid concurrency issues with the ggml/llama C binding. This means multiple `local_llm` steps will be queued when running in-process.
- Use `LLMC_SUBPROCESS=1` to enable subprocess workers; each worker is an isolated process that can load models independently and run in parallel.

Building with Pro features
--------------------------
This repo supports an optional private `pro` module. To build with Pro features locally use a `go.work` or `replace` to make the private module available and build with `-tags pro`.

CI and tests
------------
- Run unit and integration (for all fixtures) tests: `go test ./...`
- For CI, avoid enabling `LLMC_SUBPROCESS=1` unless the runner has sufficient memory for multiple model processes.

Cleanup before pushing
----------------------
Remove generated artifacts before committing. The repo's `.gitignore` excludes common generated folders (e.g., `build/`, `testdata/output/`). If you previously committed generated files, remove them from the index first:

```bash
git rm -r --cached build/ testdata/output/ || true
git commit -m "chore: remove generated artifacts from repo"
```

How to contribute
-----------------
See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on opening issues and submitting pull requests.

Third-Party Dependencies
------------------------
This project integrates the following open-source software:

- **llama.cpp**  
  https://github.com/ggml-org/llama.cpp  
  Licensed under the MIT License  
  Copyright (c) 2023–2024 The ggml authors

`llama.cpp` is included as a git submodule and remains under its original license.

Public API Stability
--------------------
Only packages under `pkg/` are considered public API.

- Packages under `internal/` are private implementation details
- CLI behavior may change between minor versions
- Public APIs may change during `v0.x`, but breaking changes will be documented

Do not depend on non-`pkg/` packages.

Roadmap
-------
- [ ] Stable public API
- [ ] Additional backend support
- [ ] Example projects

License
-------
This project is licensed under the Apache 2.0 License. See [LICENSE](LICENSE) for details.
