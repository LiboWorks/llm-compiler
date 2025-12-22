# llm-compiler

`llm-compiler` is a Go-based workflow compiler and runtime for running LLM-powered workflows locally. It compiles multi-document YAML workflow definitions into a native Go program that orchestrates shell commands and (optionally) local LLM inference via `llama.cpp`.

Important platform note
-----------------------
This project is currently supported on macOS. Windows is not supported at this time (Linux support is partial and may require manual backend choices for `llama.cpp`).

Key features
------------
- Compile one or more YAML workflows into a single Go binary that runs workflows concurrently.
- Cross-workflow synchronization via `wait_for` with optional timeouts and fail-fast error propagation.
- Shell steps with template substitution using workflow outputs.
- Local LLM inference via an internal `llama.cpp` binding (`local_llm` step type).
- Optional subprocess worker mode (`LLMC_SUBPROCESS=1`) for true concurrent model execution.
- Integration test harness that compiles example workflows and persists outputs for debugging in CI.

Quickstart
----------
1. Clone the repo and build (macOS recommended):

```bash
git clone https://github.com/LiboWorks/llm-compiler.git
cd llm-compiler
go build ./...
```

2. Compile your workflows (example):

```bash
go run main.go compile example.yaml -o ./build
```

3. Run the generated program:

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

Prerequisites: `llama.cpp`
---------------------------------
To use `local_llm` you must initialize the `llama.cpp` submodule and build it locally.

```bash
# Initialize the submodule
git submodule update --init --recursive

# Build llama.cpp (macOS with Metal backend)
cd third_party/llama.cpp
mkdir -p build && cd build
cmake .. -DLLAMA_BACKEND=metal -DCMAKE_BUILD_TYPE=Release
cmake --build . --config Release -j$(sysctl -n hw.ncpu)
cd ../../..
```

On Linux choose an appropriate backend (e.g., `-DLLAMA_BACKEND=cpu`).

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
- Open issues for bugs and feature requests.
- For larger changes, open a draft PR and discuss design before implementing.

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
See the `LICENSE` file at the repository root.
