package workflow

type Workflow struct {
	Name  string         `yaml:"name"`
	Steps []WorkflowStep `yaml:"steps"`
}

type StepType string

const (
	StepShell    StepType = "shell"
	StepLLM      StepType = "llm" // <-- add this
	StepLocalLLM StepType = "local_llm"
)

type WorkflowStep struct {
	Name      string   `yaml:"name"`
	Type      StepType `yaml:"type"`
	Command   string   `yaml:"command,omitempty"` // for shell
	Prompt    string   `yaml:"prompt,omitempty"`  // for LLM
	Model     string   `yaml:"model,omitempty"`   // for LLM
	MaxTokens int      `yaml:"max_tokens,omitempty"`
	Output    string   `yaml:"output,omitempty"`
	If        string
	// WaitFor optionally specifies another workflow step to wait on before
	// executing this step. Format: "workflowName.stepName". When the
	// producer step completes and has an `output`, its value will be sent on
	// a coordination channel under that key and the waiting step will receive
	// it and store it into its local context under the same key.
	WaitFor string `yaml:"wait_for,omitempty"`
	// Optional timeout in seconds to wait for the producer. 0 means block
	// indefinitely.
	WaitTimeout int `yaml:"wait_timeout,omitempty"`
}
