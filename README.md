# llm-compiler

llm-compiler is a Go-based workflow compiler and runtime for running LLM-powered workflows locally. It supports:

- Multi-workflow compilation from YAML (multi-document) into a single Go program that runs workflows concurrently.
- Cross-workflow synchronization via `wait_for` on a step level with optional `wait_timeout`.
- Error propagation from producers to waiting steps (fail-fast semantics).
- Support for local LLM inference via an internal `llama.cpp` binding.
- Optional subprocess worker mode for true concurrent model execution (one worker process per runtime instance).
- Integration test scaffolding that compiles example workflows and persists outputs for CI debugging.

Quick features
- Compile workflow YAML into a Go binary: `go run ./cmd/compile.go -i example.yaml -o ./build/workflows.go`
- Run generated binary: `./build/workflows` (or run with subprocess workers: `LLMC_SUBPROCESS=1 ./build/workflows`)
- Enable subprocess workers: set environment variable `LLMC_SUBPROCESS=1` before running the generated program. Each runtime will spawn a worker subprocess (the same executable) with `LLMC_WORKER=1` to isolate model contexts and allow concurrent predictions.

Prerequisites: system tools and `llama.cpp`
--------------------------------------------------
This project uses the `llama.cpp` codebase for local model inference. Because `third_party/` is ignored, you'll need to fetch and build `llama.cpp` locally before running the runtime.

1) Clone a specific tested `llama.cpp` tag (example uses `v0.1.0` — replace with the version you want):

```bash
# clone into third_party/llama.cpp
git clone --depth 1 --branch v0.1.0 https://github.com/ggerganov/llama.cpp.git third_party/llama.cpp
```

2) Build the C/C++ libraries (macOS example using CMake):

```bash
cd third_party/llama.cpp
mkdir -p build && cd build
cmake .. -DLLAMA_BACKEND=metal -DCMAKE_BUILD_TYPE=Release
cmake --build . --config Release -j$(sysctl -n hw.ncpu)
cd ../../
```

On Linux you might use `-DLLAMA_BACKEND=cpu` or appropriate backend flags.

3) After building, you can compile and run the project as normal. Example:

```bash
# build the compiler or the generated program
go build ./...

# run the compile command to generate a binary for your workflows
go run ./cmd/compile.go -i example.yaml -o ./build/workflows.go

# build and run (use LLMC_SUBPROCESS=1 to enable subprocess workers)
go build -o build/workflows ./build/workflows.go
LLMC_SUBPROCESS=1 ./build/workflows
```

Cloning the repository with submodules
------------------------------------
If you (or CI) prefer the repository to bring in the `llama.cpp` submodule automatically, clone with submodules:

```bash
git clone --recurse-submodules https://github.com/your-org/llm-compiler.git
# or, if you already cloned without submodules:
git submodule update --init --recursive
```

This ensures `third_party/llama.cpp` is populated and ready for the build steps described above.

macOS (Metal) notes
--------------------
If you plan to run with Metal (Apple GPUs), build `llama.cpp` on macOS with the Metal backend. The GitHub macOS runner does not include Metal libraries for production inference, but the build step below demonstrates how to compile for Metal on macOS:

```bash
cd third_party/llama.cpp
mkdir -p build && cd build
cmake .. -DLLAMA_BACKEND=metal -DCMAKE_BUILD_TYPE=Release
cmake --build . --config Release -j$(sysctl -n hw.ncpu)
```

Be cautious: Metal-backed model execution may require device-specific drivers and enough GPU memory; it is resource-heavy.

Design notes and important behavior
- By default, in-process model calls are serialized at a low level to avoid ggml/llama C-level concurrency issues (this is safe but serializes generation). Use `LLMC_SUBPROCESS=1` to enable true concurrency via subprocess isolation.
- The runtime supports both stateless and stateful generation. Currently the built-in `Predict` resets the internal context before each call to avoid sequence-position mismatches when callers issue independent predictions. If you require multi-turn sessions, consider using the lower-level runtime APIs or request an enhancement to expose per-session contexts.

Project layout (key paths)
- `cmd/` — CLI entry points (compiler, pro_register, root).
- `internal/` — core packages:
  - `internal/generator` — codegen for producing runnable Go programs from workflows.
  - `internal/runtime` — runtime adapters, including `LocalLlamaRuntime` and subprocess worker client.
  - `internal/llama` — cgo wrapper for `llama.cpp`/ggml.
  - `internal/workflow` — parser and model for workflow YAML.
- `build/` — generated files and test runs (can be removed; see cleanup section).
  - This folder contains generated Go sources and binaries produced by the compiler and test harnesses.

Cleanup before pushing
To keep the repo tidy, remove generated artifacts locally before pushing. The project ignores `build/`, `generated/`, and `utils/` (see `.gitignore`), but to remove them locally run:

```bash
# remove generated build artifacts and binaries
rm -rf build/ generated/ utils/

# ensure no large model files are accidentally committed
git status --porcelain
git add -A
git commit -m "chore: tidy generated artifacts"
```

CI and integration tests
- The repository includes an integration test that compiles `example.yaml` and runs the generated program. For CI, prefer enabling `LLMC_SUBPROCESS=1` only if the CI runners have enough memory to host multiple model processes; otherwise, run with the default in-process mode or use a smaller test model.

How to contribute
- Open issues for bugs and feature requests.
- For large changes (e.g., introducing a worker pool or session APIs), open a draft PR and discuss design before implementation.

If you want, I can (A) remove generated files in the repo tree now (I can only modify files I can safely delete), or (B) run the integration test with subprocess workers and capture logs. Tell me which and I'll proceed.
