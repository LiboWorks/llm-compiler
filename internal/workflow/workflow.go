package workflow

type Workflow struct {
    Name  string            `yaml:"name"`
    Steps []WorkflowStep    `yaml:"steps"`
}

type StepType string

const (
    StepShell StepType = "shell"
    StepLLM   StepType = "llm" // <-- add this
)

type WorkflowStep struct {
    Name    string   `yaml:"name"`
    Type    StepType `yaml:"type"`
    Command string   `yaml:"command,omitempty"` // for shell
    Prompt  string   `yaml:"prompt,omitempty"`  // for LLM
    Model   string   `yaml:"model,omitempty"`   // for LLM
    Output  string   `yaml:"output,omitempty"`
    If      string
}
