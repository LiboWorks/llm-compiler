package workflow

import "fmt"

// Allowed step types
var validStepTypes = map[string]bool{
	"llm": true,
}

func (wf *Workflow) Validate() error {
	if wf.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(wf.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	for i, step := range wf.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d is missing a name", i+1)
		}

		switch step.Type {
		case StepShell:
			if step.Command == "" {
				return fmt.Errorf("shell step %s missing command", step.Name)
			}
		case StepLLM:
		case StepLocalLLM:
			if step.Prompt == "" {
				return fmt.Errorf("llm step %s missing prompt", step.Name)
			}
			if step.Model == "" {
				return fmt.Errorf("llm step %s missing model", step.Name)
			}
		default:
			return fmt.Errorf("unknown step type: %s", step.Type)
		}

	}
	return nil
}
