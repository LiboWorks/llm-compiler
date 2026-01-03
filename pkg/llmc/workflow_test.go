package llmc_test

import (
	"testing"

	"github.com/LiboWorks/llm-compiler/pkg/llmc"
)

func TestNewWorkflow(t *testing.T) {
	wf := llmc.NewWorkflow("my-workflow")

	if wf.Name != "my-workflow" {
		t.Errorf("expected name 'my-workflow', got %s", wf.Name)
	}
	if wf.Steps == nil {
		t.Error("Steps should not be nil")
	}
	if len(wf.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(wf.Steps))
	}
}

func TestWorkflowAddStep(t *testing.T) {
	wf := llmc.NewWorkflow("test")

	step1 := llmc.ShellStep("step1", "echo 'hello'").Build()
	step2 := llmc.ShellStep("step2", "echo 'world'").Build()

	// Test chaining
	result := wf.AddStep(step1).AddStep(step2)

	if result != wf {
		t.Error("AddStep should return the workflow for chaining")
	}
	if len(wf.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(wf.Steps))
	}
}

func TestShellStep(t *testing.T) {
	step := llmc.ShellStep("my-step", "echo 'test'").Build()

	if step.Name != "my-step" {
		t.Errorf("expected name 'my-step', got %s", step.Name)
	}
	if step.Type != llmc.StepTypeShell {
		t.Errorf("expected type Shell, got %s", step.Type)
	}
	if step.Command != "echo 'test'" {
		t.Errorf("expected command \"echo 'test'\", got %s", step.Command)
	}
}

func TestLLMStep(t *testing.T) {
	step := llmc.LLMStep("llm-step", "Summarize: {{input}}").Build()

	if step.Name != "llm-step" {
		t.Errorf("expected name 'llm-step', got %s", step.Name)
	}
	if step.Type != llmc.StepTypeLLM {
		t.Errorf("expected type LLM, got %s", step.Type)
	}
	if step.Prompt != "Summarize: {{input}}" {
		t.Errorf("expected prompt 'Summarize: {{input}}', got %s", step.Prompt)
	}
}

func TestLocalLLMStep(t *testing.T) {
	step := llmc.LocalLLMStep("local-step", "Generate: {{data}}").Build()

	if step.Name != "local-step" {
		t.Errorf("expected name 'local-step', got %s", step.Name)
	}
	if step.Type != llmc.StepTypeLocalLLM {
		t.Errorf("expected type LocalLLM, got %s", step.Type)
	}
	if step.Prompt != "Generate: {{data}}" {
		t.Errorf("expected prompt 'Generate: {{data}}', got %s", step.Prompt)
	}
}

func TestStepBuilderWithOutput(t *testing.T) {
	step := llmc.ShellStep("step", "echo 'test'").
		WithOutput("result").
		Build()

	if step.Output != "result" {
		t.Errorf("expected output 'result', got %s", step.Output)
	}
}

func TestStepBuilderWithModel(t *testing.T) {
	step := llmc.LLMStep("step", "prompt").
		WithModel("gpt-4").
		Build()

	if step.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %s", step.Model)
	}
}

func TestStepBuilderWithMaxTokens(t *testing.T) {
	step := llmc.LLMStep("step", "prompt").
		WithMaxTokens(1024).
		Build()

	if step.MaxTokens != 1024 {
		t.Errorf("expected max_tokens 1024, got %d", step.MaxTokens)
	}
}

func TestStepBuilderWithCondition(t *testing.T) {
	step := llmc.ShellStep("step", "echo 'test'").
		WithCondition("{{mode}} == 'prod'").
		Build()

	if step.If != "{{mode}} == 'prod'" {
		t.Errorf("expected condition \"{{mode}} == 'prod'\", got %s", step.If)
	}
}

func TestStepBuilderWaitFor(t *testing.T) {
	step := llmc.ShellStep("step", "echo 'test'").
		WaitFor("other.step").
		Build()

	if step.WaitFor != "other.step" {
		t.Errorf("expected wait_for 'other.step', got %s", step.WaitFor)
	}
}

func TestStepBuilderWithTimeout(t *testing.T) {
	step := llmc.ShellStep("step", "echo 'test'").
		WaitFor("other.step").
		WithTimeout(30).
		Build()

	if step.WaitTimeout != 30 {
		t.Errorf("expected timeout 30, got %d", step.WaitTimeout)
	}
}

func TestStepBuilderChaining(t *testing.T) {
	step := llmc.LLMStep("analyze", "Analyze: {{data}}").
		WithModel("gpt-4").
		WithMaxTokens(2048).
		WithOutput("analysis").
		WithCondition("{{enabled}} == 'true'").
		Build()

	if step.Name != "analyze" {
		t.Errorf("expected name 'analyze', got %s", step.Name)
	}
	if step.Type != llmc.StepTypeLLM {
		t.Errorf("expected type LLM, got %s", step.Type)
	}
	if step.Prompt != "Analyze: {{data}}" {
		t.Errorf("expected prompt 'Analyze: {{data}}', got %s", step.Prompt)
	}
	if step.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %s", step.Model)
	}
	if step.MaxTokens != 2048 {
		t.Errorf("expected max_tokens 2048, got %d", step.MaxTokens)
	}
	if step.Output != "analysis" {
		t.Errorf("expected output 'analysis', got %s", step.Output)
	}
	if step.If != "{{enabled}} == 'true'" {
		t.Errorf("expected condition \"{{enabled}} == 'true'\", got %s", step.If)
	}
}

func TestStepTypes(t *testing.T) {
	if llmc.StepTypeShell != "shell" {
		t.Errorf("StepTypeShell should be 'shell', got %s", llmc.StepTypeShell)
	}
	if llmc.StepTypeLLM != "llm" {
		t.Errorf("StepTypeLLM should be 'llm', got %s", llmc.StepTypeLLM)
	}
	if llmc.StepTypeLocalLLM != "local_llm" {
		t.Errorf("StepTypeLocalLLM should be 'local_llm', got %s", llmc.StepTypeLocalLLM)
	}
}

func TestComplexWorkflowConstruction(t *testing.T) {
	// Build a realistic multi-step workflow
	wf := llmc.NewWorkflow("data-pipeline")

	wf.AddStep(llmc.ShellStep("fetch", "curl -s https://api.example.com/data").
		WithOutput("raw_data").
		Build())

	wf.AddStep(llmc.ShellStep("preprocess", "echo '{{raw_data}}' | jq '.items'").
		WithOutput("items").
		Build())

	wf.AddStep(llmc.LLMStep("analyze", "Analyze these items and summarize: {{items}}").
		WithModel("gpt-4").
		WithMaxTokens(1024).
		WithOutput("analysis").
		Build())

	wf.AddStep(llmc.ShellStep("save", "echo '{{analysis}}' > /tmp/report.txt").
		WithCondition("{{analysis}} != ''").
		Build())

	// Verify structure
	if wf.Name != "data-pipeline" {
		t.Errorf("expected workflow name 'data-pipeline', got %s", wf.Name)
	}
	if len(wf.Steps) != 4 {
		t.Errorf("expected 4 steps, got %d", len(wf.Steps))
	}

	// Verify step order and properties
	steps := wf.Steps
	if steps[0].Name != "fetch" || steps[0].Type != llmc.StepTypeShell {
		t.Error("first step should be 'fetch' shell step")
	}
	if steps[2].Name != "analyze" || steps[2].Type != llmc.StepTypeLLM {
		t.Error("third step should be 'analyze' LLM step")
	}
	if steps[3].If == "" {
		t.Error("fourth step should have a condition")
	}
}

func TestCrossWorkflowConstruction(t *testing.T) {
	// Producer workflow
	producer := llmc.NewWorkflow("producer")
	producer.AddStep(llmc.ShellStep("generate", "echo 'data-123'").
		WithOutput("data").
		Build())

	// Consumer workflow with wait_for
	consumer := llmc.NewWorkflow("consumer")
	consumer.AddStep(llmc.ShellStep("receive", "echo 'received: {{producer.generate}}'").
		WaitFor("producer.generate").
		WithTimeout(10).
		WithOutput("received").
		Build())

	// Verify wait_for configuration
	consumerStep := consumer.Steps[0]
	if consumerStep.WaitFor != "producer.generate" {
		t.Errorf("expected wait_for 'producer.generate', got %s", consumerStep.WaitFor)
	}
	if consumerStep.WaitTimeout != 10 {
		t.Errorf("expected timeout 10, got %d", consumerStep.WaitTimeout)
	}
}
