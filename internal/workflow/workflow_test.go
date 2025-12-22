package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

func getFixturePath(name string) string {
	// Find repo root
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		dir = filepath.Dir(dir)
	}
	return filepath.Join(dir, "testdata", "fixtures", name+".yaml")
}

func TestParseShellBasic(t *testing.T) {
	path := getFixturePath("shell_basic")
	wfs, err := LoadWorkflows(path)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if len(wfs) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(wfs))
	}

	wf := wfs[0]
	if wf.Name != "shell_basic" {
		t.Errorf("expected workflow name 'shell_basic', got %q", wf.Name)
	}

	if len(wf.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(wf.Steps))
	}

	// Check first step
	step1 := wf.Steps[0]
	if step1.Name != "echo_hello" {
		t.Errorf("expected step name 'echo_hello', got %q", step1.Name)
	}
	if step1.Type != StepShell {
		t.Errorf("expected step type 'shell', got %q", step1.Type)
	}
	if step1.Command != `echo "hello world"` {
		t.Errorf("unexpected command: %q", step1.Command)
	}
	if step1.Output != "hello_result" {
		t.Errorf("expected output 'hello_result', got %q", step1.Output)
	}
}

func TestParseCrossWorkflow(t *testing.T) {
	path := getFixturePath("cross_workflow")
	wfs, err := LoadWorkflows(path)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if len(wfs) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(wfs))
	}

	// Find consumer workflow
	var consumer *Workflow
	for i := range wfs {
		if wfs[i].Name == "consumer" {
			consumer = &wfs[i]
			break
		}
	}

	if consumer == nil {
		t.Fatal("consumer workflow not found")
	}

	// Check wait_for
	step := consumer.Steps[0]
	if step.WaitFor != "producer.produce" {
		t.Errorf("expected wait_for 'producer.produce', got %q", step.WaitFor)
	}
	if step.WaitTimeout != 10 {
		t.Errorf("expected wait_timeout 10, got %d", step.WaitTimeout)
	}
}

func TestParseConditional(t *testing.T) {
	path := getFixturePath("conditional")
	wfs, err := LoadWorkflows(path)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if len(wfs) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(wfs))
	}

	wf := wfs[0]
	if len(wf.Steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(wf.Steps))
	}

	// Check conditional step
	step2 := wf.Steps[1]
	if step2.If != "{{flag}} == 'yes'" {
		t.Errorf("unexpected if condition: %q", step2.If)
	}
}

func TestParseParallel(t *testing.T) {
	path := getFixturePath("parallel")
	wfs, err := LoadWorkflows(path)
	if err != nil {
		t.Fatalf("failed to parse workflow: %v", err)
	}

	if len(wfs) != 3 {
		t.Errorf("expected 3 workflows, got %d", len(wfs))
	}

	names := make(map[string]bool)
	for _, wf := range wfs {
		names[wf.Name] = true
	}

	expected := []string{"parallel_a", "parallel_b", "parallel_c"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected workflow %q not found", name)
		}
	}
}

func TestParseInvalidYAML(t *testing.T) {
	// Test with non-existent file
	_, err := LoadWorkflows("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestValidateWorkflow(t *testing.T) {
	tests := []struct {
		name    string
		wf      Workflow
		wantErr bool
	}{
		{
			name: "valid workflow",
			wf: Workflow{
				Name: "test",
				Steps: []WorkflowStep{
					{Name: "step1", Type: StepShell, Command: "echo hello"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			wf: Workflow{
				Steps: []WorkflowStep{
					{Name: "step1", Type: StepShell, Command: "echo hello"},
				},
			},
			wantErr: true,
		},
		{
			name: "empty steps",
			wf: Workflow{
				Name:  "test",
				Steps: []WorkflowStep{},
			},
			wantErr: true,
		},
		{
			name: "step without name",
			wf: Workflow{
				Name: "test",
				Steps: []WorkflowStep{
					{Type: StepShell, Command: "echo hello"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.wf.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
