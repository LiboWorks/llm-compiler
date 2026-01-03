package llmc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LiboWorks/llm-compiler/pkg/llmc"
)

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
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

// ============================================================================
// LoadWorkflows tests - Tests YAML loading and type conversion to public types
// ============================================================================

func TestLoadWorkflows(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("could not find repo root: %v", err)
	}

	t.Run("converts internal types to public types", func(t *testing.T) {
		fixturePath := filepath.Join(repoRoot, "testdata", "fixtures", "shell_basic.yaml")
		workflows, err := llmc.LoadWorkflows(fixturePath)
		if err != nil {
			t.Fatalf("LoadWorkflows failed: %v", err)
		}

		if len(workflows) == 0 {
			t.Fatal("expected at least one workflow")
		}

		// Verify public Workflow type is populated correctly
		wf := workflows[0]
		if wf.Name == "" {
			t.Error("workflow name should not be empty")
		}
		if len(wf.Steps) == 0 {
			t.Error("expected at least one step")
		}

		// Verify Step fields are converted
		step := wf.Steps[0]
		if step.Name == "" {
			t.Error("step name should not be empty")
		}
		if step.Type != llmc.StepTypeShell {
			t.Errorf("expected StepTypeShell, got %s", step.Type)
		}
	})

	t.Run("multi workflow YAML", func(t *testing.T) {
		fixturePath := filepath.Join(repoRoot, "testdata", "fixtures", "cross_workflow.yaml")
		workflows, err := llmc.LoadWorkflows(fixturePath)
		if err != nil {
			t.Fatalf("LoadWorkflows failed: %v", err)
		}

		if len(workflows) < 2 {
			t.Errorf("expected at least 2 workflows, got %d", len(workflows))
		}
	})

	t.Run("error propagation", func(t *testing.T) {
		_, err := llmc.LoadWorkflows("/nonexistent/file.yaml")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

// ============================================================================
// Validate tests - Tests public Validate wraps internal validation
// ============================================================================

func TestValidate(t *testing.T) {
	t.Run("valid workflow", func(t *testing.T) {
		wf := llmc.NewWorkflow("valid")
		wf.AddStep(llmc.ShellStep("step1", "echo 'test'").Build())

		err := llmc.Validate(wf)
		if err != nil {
			t.Errorf("expected valid workflow, got error: %v", err)
		}
	})

	t.Run("empty workflow name", func(t *testing.T) {
		wf := llmc.NewWorkflow("")
		wf.AddStep(llmc.ShellStep("step1", "echo 'test'").Build())

		err := llmc.Validate(wf)
		if err == nil {
			t.Error("expected error for empty workflow name")
		}
	})

	t.Run("empty steps", func(t *testing.T) {
		wf := llmc.NewWorkflow("empty-steps")

		err := llmc.Validate(wf)
		if err == nil {
			t.Error("expected error for workflow with no steps")
		}
	})
}

// ============================================================================
// Type conversion tests - Ensure public types convert correctly to internal
// ============================================================================

func TestWorkflowToInternalConversion(t *testing.T) {
	// Build a workflow with all field types populated
	wf := llmc.NewWorkflow("conversion-test")
	wf.AddStep(llmc.ShellStep("shell-step", "echo 'hello'").
		WithOutput("result").
		WithCondition("{{mode}} == 'test'").
		Build())
	wf.AddStep(llmc.LLMStep("llm-step", "Summarize: {{result}}").
		WithModel("gpt-4").
		WithMaxTokens(1024).
		WithOutput("summary").
		Build())
	wf.AddStep(llmc.LocalLLMStep("local-step", "Generate: {{summary}}").
		WithModel("/path/to/model.gguf").
		WaitFor("other.step").
		WithTimeout(30).
		Build())

	// Validate exercises the conversion path (toInternal is called)
	err := llmc.Validate(wf)
	if err != nil {
		t.Errorf("workflow should be valid, got: %v", err)
	}

	// Verify all step types are set correctly
	if wf.Steps[0].Type != llmc.StepTypeShell {
		t.Errorf("expected shell type, got %s", wf.Steps[0].Type)
	}
	if wf.Steps[1].Type != llmc.StepTypeLLM {
		t.Errorf("expected llm type, got %s", wf.Steps[1].Type)
	}
	if wf.Steps[2].Type != llmc.StepTypeLocalLLM {
		t.Errorf("expected local_llm type, got %s", wf.Steps[2].Type)
	}
}

// ============================================================================
// API error handling tests
// ============================================================================

func TestCompileFileErrorHandling(t *testing.T) {
	t.Run("file not found", func(t *testing.T) {
		_, err := llmc.CompileFile("/nonexistent/file.yaml", nil)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("nil options uses defaults", func(t *testing.T) {
		// This tests that nil options don't panic
		_, err := llmc.CompileFile("/nonexistent/file.yaml", nil)
		if err == nil {
			t.Error("expected error (file not found, not nil pointer)")
		}
	})
}
