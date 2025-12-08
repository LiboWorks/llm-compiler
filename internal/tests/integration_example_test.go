package tests

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Integration test: compile a small two-workflow YAML (using shell steps), build
// the generated program and run it to verify the compile->run path and
// cross-workflow wait behavior.
func TestCompileAndRunExample(t *testing.T) {
	t.Helper()

	// Determine repository root (look for go.mod) and run compile from there.
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("pwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(repoRoot, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Fatalf("could not find repository root (go.mod)")
		}
		repoRoot = parent
	}

	// Use the repository's `example.yaml` as the workflow input for the
	// integration test so the test compiles a real example supplied with
	// the repo.
	wfPath := filepath.Join(repoRoot, "example.yaml")
	if _, statErr := os.Stat(wfPath); os.IsNotExist(statErr) {
		t.Fatalf("example.yaml not found at %s", wfPath)
	}

	// Create a persistent output folder under `build/` so generated files and
	// program output persist after the test for inspection.
	outDir := filepath.Join(repoRoot, "build", "integration_test_outputs")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("mkdir outDir: %v", err)
	}

	// Run the compile command from the repo root
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "main.go", "compile", wfPath, "-o", outDir)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compile failed: %v\noutput:\n%s", err, string(out))
	}

	// Find generated .go file
	files, err := filepath.Glob(filepath.Join(outDir, "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no generated files: %v", err)
	}
	gen := files[0]

	// Build the generated program
	binPath := filepath.Join(outDir, "prog")
	build := exec.CommandContext(ctx, "go", "build", "-o", binPath, gen)
	if bout, berr := build.CombinedOutput(); berr != nil {
		t.Fatalf("build generated failed: %v\n%s", berr, string(bout))
	}

	// Run the binary
	run := exec.CommandContext(ctx, binPath)
	// Run the generated binary from the repository root so model paths and
	// other relative resources resolve relative to the project, not the
	// test package directory.
	run.Dir = repoRoot
	bout, berr := run.CombinedOutput()
	if berr != nil {
		t.Fatalf("run generated failed: %v\n%s", berr, string(bout))
	}

	got := string(bout)
	// Write the program output to a file in the output directory so developers
	// can inspect it after a failing test. Overwrite on each run.
	outFile := filepath.Join(outDir, "prog_output.txt")
	if werr := ioutil.WriteFile(outFile, []byte(got), 0644); werr != nil {
		t.Fatalf("failed to write program output to %s: %v", outFile, werr)
	}
	// Fail the test if the llama/ggml runtime emitted decode or KV-cache
	// corruption errors. These indicate unsafe concurrent access to the
	// local LLM runtime and should make the integration test fail even if
	// the program printed a completion marker.
	badPatterns := []string{
		"init: the tokens of sequence 0",
		"decode: failed to initialize batch",
		"llama_decode: failed to decode",
		"GGML_ASSERT",
		"KV cache",
	}
	for _, p := range badPatterns {
		if strings.Contains(got, p) {
			t.Fatalf("detected llama/ggml runtime error in program output: %q\nfull output:\n%s", p, got)
		}
	}
	if !strings.Contains(got, "Workflows completed") && !strings.Contains(got, "✅ Workflows completed") {
		t.Fatalf("unexpected output, missing completion marker:\n%s", got)
	}
	if !strings.Contains(got, "consumer got: hello-from-p") {
		t.Fatalf("consumer didn't receive producer output:\n%s", got)
	}
}
