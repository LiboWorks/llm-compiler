package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LiboWorks/llm-compiler/internal/workflow"
)

func TestGenerateShellWorkflow(t *testing.T) {
	wfs := []workflow.Workflow{
		{
			Name: "test",
			Steps: []workflow.WorkflowStep{
				{
					Name:    "step1",
					Type:    workflow.StepShell,
					Command: "echo hello",
					Output:  "result",
				},
			},
		},
	}

	code, err := Generate(wfs)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check that generated code contains expected elements
	checks := []string{
		"package main",
		"import (",
		"Workflows completed",
	}

	for _, check := range checks {
		if !strings.Contains(code, check) {
			t.Errorf("generated code missing: %q", check)
		}
	}
}

func TestGenerateMultipleWorkflows(t *testing.T) {
	wfs := []workflow.Workflow{
		{
			Name: "workflow_a",
			Steps: []workflow.WorkflowStep{
				{Name: "a1", Type: workflow.StepShell, Command: "echo A"},
			},
		},
		{
			Name: "workflow_b",
			Steps: []workflow.WorkflowStep{
				{Name: "b1", Type: workflow.StepShell, Command: "echo B"},
			},
		},
	}

	code, err := Generate(wfs)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check for workflow comments
	if !strings.Contains(code, "Workflow: workflow_a") {
		t.Error("missing workflow_a comment")
	}
	if !strings.Contains(code, "Workflow: workflow_b") {
		t.Error("missing workflow_b comment")
	}

	// Check for goroutines
	if strings.Count(code, "wg.Add(1)") != 2 {
		t.Error("expected 2 wg.Add(1) calls for 2 workflows")
	}
}

func TestGenerateWithWaitFor(t *testing.T) {
	wfs := []workflow.Workflow{
		{
			Name: "producer",
			Steps: []workflow.WorkflowStep{
				{Name: "produce", Type: workflow.StepShell, Command: "echo data", Output: "data"},
			},
		},
		{
			Name: "consumer",
			Steps: []workflow.WorkflowStep{
				{
					Name:        "consume",
					Type:        workflow.StepShell,
					WaitFor:     "producer.produce",
					WaitTimeout: 10,
					Command:     "echo received",
				},
			},
		},
	}

	code, err := Generate(wfs)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check for wait_for handling
	if !strings.Contains(code, `mk("producer.produce")`) {
		t.Error("missing signal key for producer.produce")
	}

	// Check for timeout handling
	if !strings.Contains(code, "time.After") {
		t.Error("missing timeout handling")
	}
}

func TestGenerateWithConditional(t *testing.T) {
	wfs := []workflow.Workflow{
		{
			Name: "conditional",
			Steps: []workflow.WorkflowStep{
				{
					Name:    "step1",
					Type:    workflow.StepShell,
					If:      "{{flag}} == 'yes'",
					Command: "echo conditional",
				},
			},
		},
	}

	code, err := Generate(wfs)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check for conditional
	if !strings.Contains(code, "runtime.EvalCondition") {
		t.Error("missing EvalCondition call")
	}
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "test_output.go")

	content := "package main\n\nfunc main() {}\n"
	err := SaveToFile(outPath, content)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content mismatch: got %q, want %q", string(data), content)
	}
}

func TestGenerateLLMWorkflow(t *testing.T) {
	wfs := []workflow.Workflow{
		{
			Name: "llm_test",
			Steps: []workflow.WorkflowStep{
				{
					Name:      "generate",
					Type:      workflow.StepLLM,
					Prompt:    "Say hello",
					Model:     "gpt-4",
					MaxTokens: 100,
					Output:    "response",
				},
			},
		},
	}

	code, err := Generate(wfs)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check for prompt handling
	if !strings.Contains(code, "Say hello") {
		t.Error("missing prompt in generated code")
	}
}

func TestGenerateLocalLLMWorkflow(t *testing.T) {
	wfs := []workflow.Workflow{
		{
			Name: "local_llm_test",
			Steps: []workflow.WorkflowStep{
				{
					Name:      "generate",
					Type:      workflow.StepLocalLLM,
					Prompt:    "Say hello",
					Model:     "/path/to/model.gguf",
					MaxTokens: 100,
					Output:    "response",
				},
			},
		},
	}

	code, err := Generate(wfs)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Check for local llama runtime
	if !strings.Contains(code, "localLlama") {
		t.Error("missing local llama handling in generated code")
	}
}
