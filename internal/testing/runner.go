// Package testing provides test utilities and helpers for llm-compiler tests.
package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFixture represents a test workflow fixture
type TestFixture struct {
	Name        string
	YAMLPath    string
	Description string
}

// TestResult holds the results of running a compiled workflow
type TestResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Duration   time.Duration
	Contexts   map[string]map[string]string
	Signals    map[string]map[string]string
	FmtOutput  string
	LlamaOutput string
}

// TestRunner provides utilities for running workflow tests
type TestRunner struct {
	RepoRoot    string
	FixturesDir string
	OutputDir   string
	t           *testing.T
}

// NewTestRunner creates a new test runner with isolated output directory
func NewTestRunner(t *testing.T) (*TestRunner, error) {
	t.Helper()

	// Find repository root
	repoRoot, err := findRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find repo root: %w", err)
	}

	// Create a unique output directory for this test to avoid parallel conflicts
	baseOutputDir := filepath.Join(repoRoot, "testdata", "output")
	uniqueOutputDir := filepath.Join(baseOutputDir, fmt.Sprintf("run_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(uniqueOutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	// Clean up the unique directory when test completes
	t.Cleanup(func() {
		os.RemoveAll(uniqueOutputDir)
	})

	return &TestRunner{
		RepoRoot:    repoRoot,
		FixturesDir: filepath.Join(repoRoot, "testdata", "fixtures"),
		OutputDir:   uniqueOutputDir,
		t:           t,
	}, nil
}

// findRepoRoot finds the repository root by looking for go.mod
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// GetFixture returns a test fixture by name
func (r *TestRunner) GetFixture(name string) TestFixture {
	return TestFixture{
		Name:     name,
		YAMLPath: filepath.Join(r.FixturesDir, name+".yaml"),
	}
}

// ListFixtures returns all available fixtures
func (r *TestRunner) ListFixtures() ([]TestFixture, error) {
	files, err := filepath.Glob(filepath.Join(r.FixturesDir, "*.yaml"))
	if err != nil {
		return nil, err
	}

	fixtures := make([]TestFixture, len(files))
	for i, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".yaml")
		fixtures[i] = TestFixture{
			Name:     name,
			YAMLPath: f,
		}
	}
	return fixtures, nil
}

// CompileWorkflow compiles a workflow YAML to a Go program
func (r *TestRunner) CompileWorkflow(fixture TestFixture, outputName string) (string, error) {
	r.t.Helper()

	outDir := filepath.Join(r.OutputDir, "generated")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	// Record existing files before compilation
	existingFiles := make(map[string]bool)
	if files, err := filepath.Glob(filepath.Join(outDir, "*.go")); err == nil {
		for _, f := range files {
			existingFiles[f] = true
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "run", "main.go", "compile", fixture.YAMLPath, "-o", outDir)
	cmd.Dir = r.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compile failed: %v\noutput: %s", err, string(out))
	}

	// Find newly generated file (the one that wasn't there before)
	files, err := filepath.Glob(filepath.Join(outDir, "*.go"))
	if err != nil {
		return "", fmt.Errorf("failed to list generated files: %w", err)
	}

	var newFile string
	for _, f := range files {
		if !existingFiles[f] {
			newFile = f
			break
		}
	}

	if newFile == "" {
		// Fall back to finding the most recently modified file
		var latestFile string
		var latestTime time.Time
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = f
			}
		}
		if latestFile == "" {
			return "", fmt.Errorf("no generated files found")
		}
		newFile = latestFile
	}

	return newFile, nil
}

// BuildWorkflow builds a compiled Go program
func (r *TestRunner) BuildWorkflow(sourcePath string, binaryName string) (string, error) {
	r.t.Helper()

	binDir := filepath.Join(r.OutputDir, "binaries")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create binary dir: %w", err)
	}

	binPath := filepath.Join(binDir, binaryName)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, sourcePath)
	cmd.Dir = r.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build failed: %v\noutput: %s", err, string(out))
	}

	return binPath, nil
}

// RunWorkflow runs a compiled workflow binary and returns results
func (r *TestRunner) RunWorkflow(binaryPath string, timeout time.Duration, env ...string) (*TestResult, error) {
	r.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()

	// Run binary from its own directory so output files are written there
	binDir := filepath.Dir(binaryPath)
	cmd := exec.CommandContext(ctx, binaryPath)
	cmd.Dir = binDir
	cmd.Env = append(os.Environ(), env...)

	// Capture both stdout and stderr
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	duration := time.Since(start)

	result := &TestResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	// Load contexts and signals
	jsonPath := filepath.Join(binDir, "contexts_and_signals.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
		var dump map[string]interface{}
		if json.Unmarshal(data, &dump) == nil {
			if contexts, ok := dump["contexts"].(map[string]interface{}); ok {
				result.Contexts = make(map[string]map[string]string)
				for k, v := range contexts {
					if vm, ok := v.(map[string]interface{}); ok {
						result.Contexts[k] = make(map[string]string)
						for kk, vv := range vm {
							if s, ok := vv.(string); ok {
								result.Contexts[k][kk] = s
							}
						}
					}
				}
			}
			if channels, ok := dump["channels"].(map[string]interface{}); ok {
				result.Signals = make(map[string]map[string]string)
				for k, v := range channels {
					if vm, ok := v.(map[string]interface{}); ok {
						result.Signals[k] = make(map[string]string)
						for kk, vv := range vm {
							if s, ok := vv.(string); ok {
								result.Signals[k][kk] = s
							}
						}
					}
				}
			}
		}
	}

	// Load fmt output
	fmtPath := filepath.Join(binDir, "fmt_output.txt")
	if data, err := os.ReadFile(fmtPath); err == nil {
		result.FmtOutput = string(data)
	}

	// Load llama output
	llamaPath := filepath.Join(binDir, "llama_output.txt")
	if data, err := os.ReadFile(llamaPath); err == nil {
		result.LlamaOutput = string(data)
	}

	return result, nil
}

// CompileAndRun is a convenience method that compiles, builds, and runs a fixture
func (r *TestRunner) CompileAndRun(fixture TestFixture, timeout time.Duration, env ...string) (*TestResult, error) {
	r.t.Helper()

	sourcePath, err := r.CompileWorkflow(fixture, fixture.Name)
	if err != nil {
		return nil, fmt.Errorf("compile failed: %w", err)
	}

	binaryPath, err := r.BuildWorkflow(sourcePath, fixture.Name)
	if err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	return r.RunWorkflow(binaryPath, timeout, env...)
}

// Assertions provides test assertion helpers
type Assertions struct {
	t      *testing.T
	result *TestResult
}

// NewAssertions creates a new assertions helper
func NewAssertions(t *testing.T, result *TestResult) *Assertions {
	return &Assertions{t: t, result: result}
}

// Completed asserts the workflow completed successfully
func (a *Assertions) Completed() *Assertions {
	a.t.Helper()
	if !strings.Contains(a.result.Stdout, "Workflows completed") &&
		!strings.Contains(a.result.Stdout, "✅ Workflows completed") {
		a.t.Errorf("workflow did not complete successfully, stdout:\n%s\nstderr:\n%s", a.result.Stdout, a.result.Stderr)
	}
	return a
}

// ExitCode asserts the exit code
func (a *Assertions) ExitCode(expected int) *Assertions {
	a.t.Helper()
	if a.result.ExitCode != expected {
		a.t.Errorf("expected exit code %d, got %d", expected, a.result.ExitCode)
	}
	return a
}

// StdoutContains asserts stdout contains a string
func (a *Assertions) StdoutContains(expected string) *Assertions {
	a.t.Helper()
	if !strings.Contains(a.result.Stdout, expected) {
		a.t.Errorf("stdout does not contain %q, got:\n%s", expected, a.result.Stdout)
	}
	return a
}

// StdoutNotContains asserts stdout does not contain a string
func (a *Assertions) StdoutNotContains(unexpected string) *Assertions {
	a.t.Helper()
	if strings.Contains(a.result.Stdout, unexpected) {
		a.t.Errorf("stdout should not contain %q, got:\n%s", unexpected, a.result.Stdout)
	}
	return a
}

// ContextHasValue asserts a context variable has an expected value
func (a *Assertions) ContextHasValue(workflow, key, expected string) *Assertions {
	a.t.Helper()
	if a.result.Contexts == nil {
		a.t.Errorf("no contexts available")
		return a
	}
	wfCtx, ok := a.result.Contexts[workflow]
	if !ok {
		a.t.Errorf("workflow %q not found in contexts", workflow)
		return a
	}
	actual, ok := wfCtx[key]
	if !ok {
		a.t.Errorf("key %q not found in workflow %q context", key, workflow)
		return a
	}
	if !strings.Contains(strings.TrimSpace(actual), strings.TrimSpace(expected)) {
		a.t.Errorf("context[%s][%s] = %q, expected to contain %q", workflow, key, actual, expected)
	}
	return a
}

// SignalHasValue asserts a signal has an expected value
func (a *Assertions) SignalHasValue(key, expected string) *Assertions {
	a.t.Helper()
	if a.result.Signals == nil {
		a.t.Errorf("no signals available")
		return a
	}
	signal, ok := a.result.Signals[key]
	if !ok {
		a.t.Errorf("signal %q not found", key)
		return a
	}
	actual := signal["val"]
	if !strings.Contains(strings.TrimSpace(actual), strings.TrimSpace(expected)) {
		a.t.Errorf("signal[%s].val = %q, expected to contain %q", key, actual, expected)
	}
	return a
}

// NoRuntimeErrors asserts no llama/ggml runtime errors occurred
func (a *Assertions) NoRuntimeErrors() *Assertions {
	a.t.Helper()
	badPatterns := []string{
		"init: the tokens of sequence 0",
		"decode: failed to initialize batch",
		"llama_decode: failed to decode",
		"GGML_ASSERT",
		"KV cache",
	}
	for _, p := range badPatterns {
		if strings.Contains(a.result.Stdout, p) || strings.Contains(a.result.LlamaOutput, p) {
			a.t.Errorf("detected runtime error: %q", p)
		}
	}
	return a
}

// DurationLessThan asserts the execution took less than the specified duration
func (a *Assertions) DurationLessThan(d time.Duration) *Assertions {
	a.t.Helper()
	if a.result.Duration >= d {
		a.t.Errorf("execution took %v, expected less than %v", a.result.Duration, d)
	}
	return a
}
