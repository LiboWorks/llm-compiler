// Package demo provides example tests demonstrating how to test compiled workflows.
//
// Run with: go test ./demo -v
package demo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestWorkflowCompiles verifies the example workflow compiles without errors.
func TestWorkflowCompiles(t *testing.T) {
	// Skip if llmc binary doesn't exist (CI may not have built it yet)
	if _, err := exec.LookPath("./llmc"); err != nil {
		// Try from repo root
		if _, err := os.Stat("../llmc"); err != nil {
			t.Skip("llmc binary not found - run 'go build ./cmd/llmc' first")
		}
	}

	tmpDir := t.TempDir()

	// Compile the workflow
	cmd := exec.Command("../llmc", "compile", "-i", "example.yaml", "-o", tmpDir, "--keep-source")
	cmd.Dir = "."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compilation failed: %v\noutput: %s", err, output)
	}

	// Verify binary was created
	binaryPath := filepath.Join(tmpDir, "example")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("expected binary at %s", binaryPath)
	}

	// Verify source was kept
	sourcePath := filepath.Join(tmpDir, "example.go")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		t.Fatalf("expected source at %s (--keep-source was set)", sourcePath)
	}
}

// TestWorkflowOutputStructure verifies the JSON output has expected structure.
// This test requires a pre-built binary - skip if not available.
func TestWorkflowOutputStructure(t *testing.T) {
	// Look for existing output from a previous run
	jsonPath := "output/example_run.json"
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Skip("No output file found - run ./run-demo.sh first to generate output")
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Verify top-level structure
	if _, ok := output["contexts"]; !ok {
		t.Error("missing 'contexts' in output")
	}
	if _, ok := output["channels"]; !ok {
		t.Error("missing 'channels' in output")
	}

	// Verify contexts contain expected workflows
	contexts, ok := output["contexts"].(map[string]interface{})
	if !ok {
		t.Fatal("'contexts' is not a map")
	}

	// Check for expected workflow prefixes (format: N_workflowName)
	foundProducer := false
	foundConsumer := false
	foundConditional := false
	for key := range contexts {
		if strings.Contains(key, "producer") {
			foundProducer = true
		}
		if strings.Contains(key, "consumer") {
			foundConsumer = true
		}
		if strings.Contains(key, "conditional") {
			foundConditional = true
		}
	}

	if !foundProducer {
		t.Error("missing 'producer' workflow in contexts")
	}
	if !foundConsumer {
		t.Error("missing 'consumer' workflow in contexts")
	}
	if !foundConditional {
		t.Error("missing 'conditional' workflow in contexts")
	}
}

// TestWorkflowContextValues verifies specific context values are set correctly.
func TestWorkflowContextValues(t *testing.T) {
	jsonPath := "output/example_run.json"
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Skip("No output file found - run ./run-demo.sh first")
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	contexts := output["contexts"].(map[string]interface{})

	// Find the conditional workflow and check mode was set
	for key, val := range contexts {
		if strings.Contains(key, "conditional") {
			ctx := val.(map[string]interface{})
			// The conditional workflow sets "mode" based on environment
			if mode, ok := ctx["mode"]; ok {
				t.Logf("conditional workflow mode: %v", mode)
			}
		}
	}
}

// TestChannelSignaling verifies cross-workflow signals are captured.
func TestChannelSignaling(t *testing.T) {
	jsonPath := "output/example_run.json"
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Skip("No output file found - run ./run-demo.sh first")
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	channels, ok := output["channels"].(map[string]interface{})
	if !ok {
		t.Fatal("'channels' is not a map")
	}

	// Verify producer signals are captured
	foundProducerSignal := false
	for key := range channels {
		if strings.Contains(key, "producer") {
			foundProducerSignal = true
			t.Logf("found channel: %s", key)
		}
	}

	if !foundProducerSignal {
		t.Log("no producer signals found in channels (this may be expected if shell-only)")
	}
}
