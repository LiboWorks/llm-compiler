package llmc

import (
	"github.com/LiboWorks/llm-compiler/internal/workflow"
)

// StepType represents the type of a workflow step.
type StepType string

const (
	// StepTypeShell executes a shell command.
	StepTypeShell StepType = "shell"

	// StepTypeLLM calls a remote LLM API (e.g., OpenAI).
	StepTypeLLM StepType = "llm"

	// StepTypeLocalLLM runs inference locally via llama.cpp.
	StepTypeLocalLLM StepType = "local_llm"
)

// Workflow represents a compiled workflow with its steps.
type Workflow struct {
	// Name is the unique identifier for this workflow.
	Name string

	// Steps contains the ordered list of workflow steps.
	Steps []*Step
}

// Step represents a single step in a workflow.
type Step struct {
	// Name is the unique identifier for this step within the workflow.
	Name string

	// Type specifies how this step executes (shell, llm, local_llm).
	Type StepType

	// Command is the shell command to execute (for StepTypeShell).
	Command string

	// Prompt is the LLM prompt template (for StepTypeLLM, StepTypeLocalLLM).
	Prompt string

	// Model specifies which LLM model to use.
	Model string

	// MaxTokens limits the LLM response length.
	MaxTokens int

	// Output is the variable name to store this step's result.
	// Can be referenced in subsequent steps via {{output_name}}.
	Output string

	// If is a conditional expression. Step only runs if it evaluates to true.
	// Supports template substitution: "{{mode}} == 'production'"
	If string

	// WaitFor specifies another workflow step to wait for before executing.
	// Format: "workflowName.stepName"
	WaitFor string

	// WaitTimeout is the timeout in seconds when waiting for another step.
	// 0 means wait indefinitely.
	WaitTimeout int
}

// NewWorkflow creates a new workflow with the given name.
func NewWorkflow(name string) *Workflow {
	return &Workflow{
		Name:  name,
		Steps: make([]*Step, 0),
	}
}

// AddStep appends a step to the workflow.
func (w *Workflow) AddStep(step *Step) *Workflow {
	w.Steps = append(w.Steps, step)
	return w
}

// StepBuilder provides a fluent API for constructing steps.
type StepBuilder struct {
	step *Step
}

// ShellStep creates a new shell step.
func ShellStep(name, command string) *StepBuilder {
	return &StepBuilder{
		step: &Step{
			Name:    name,
			Type:    StepTypeShell,
			Command: command,
		},
	}
}

// LLMStep creates a new remote LLM step.
func LLMStep(name, prompt string) *StepBuilder {
	return &StepBuilder{
		step: &Step{
			Name:   name,
			Type:   StepTypeLLM,
			Prompt: prompt,
		},
	}
}

// LocalLLMStep creates a new local LLM step (llama.cpp).
func LocalLLMStep(name, prompt string) *StepBuilder {
	return &StepBuilder{
		step: &Step{
			Name:   name,
			Type:   StepTypeLocalLLM,
			Prompt: prompt,
		},
	}
}

// WithOutput sets the output variable name for the step.
func (b *StepBuilder) WithOutput(output string) *StepBuilder {
	b.step.Output = output
	return b
}

// WithModel sets the LLM model to use.
func (b *StepBuilder) WithModel(model string) *StepBuilder {
	b.step.Model = model
	return b
}

// WithMaxTokens sets the maximum tokens for LLM response.
func (b *StepBuilder) WithMaxTokens(tokens int) *StepBuilder {
	b.step.MaxTokens = tokens
	return b
}

// WithCondition sets a conditional expression for the step.
func (b *StepBuilder) WithCondition(condition string) *StepBuilder {
	b.step.If = condition
	return b
}

// WaitFor sets a step dependency to wait for.
func (b *StepBuilder) WaitFor(stepRef string) *StepBuilder {
	b.step.WaitFor = stepRef
	return b
}

// WithTimeout sets the wait timeout in seconds.
func (b *StepBuilder) WithTimeout(seconds int) *StepBuilder {
	b.step.WaitTimeout = seconds
	return b
}

// Build returns the constructed Step.
func (b *StepBuilder) Build() *Step {
	return b.step
}

// Conversion helpers

func (w *Workflow) toInternal() workflow.Workflow {
	steps := make([]workflow.WorkflowStep, len(w.Steps))
	for i, s := range w.Steps {
		steps[i] = workflow.WorkflowStep{
			Name:        s.Name,
			Type:        workflow.StepType(s.Type),
			Command:     s.Command,
			Prompt:      s.Prompt,
			Model:       s.Model,
			MaxTokens:   s.MaxTokens,
			Output:      s.Output,
			If:          s.If,
			WaitFor:     s.WaitFor,
			WaitTimeout: s.WaitTimeout,
		}
	}
	return workflow.Workflow{
		Name:  w.Name,
		Steps: steps,
	}
}

func fromInternalWorkflow(wf workflow.Workflow) *Workflow {
	steps := make([]*Step, len(wf.Steps))
	for i, s := range wf.Steps {
		steps[i] = &Step{
			Name:        s.Name,
			Type:        StepType(s.Type),
			Command:     s.Command,
			Prompt:      s.Prompt,
			Model:       s.Model,
			MaxTokens:   s.MaxTokens,
			Output:      s.Output,
			If:          s.If,
			WaitFor:     s.WaitFor,
			WaitTimeout: s.WaitTimeout,
		}
	}
	return &Workflow{
		Name:  wf.Name,
		Steps: steps,
	}
}
