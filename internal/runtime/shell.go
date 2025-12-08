package runtime

import (
	"os/exec"
)

// ShellRuntime runs shell commands from workflow steps
type ShellRuntime struct{}

func NewShellRuntime() *ShellRuntime {
	return &ShellRuntime{}
}

func (s *ShellRuntime) Run(command string) (string, error) {
	// Use `sh -c` so full shell syntax works
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err

}
