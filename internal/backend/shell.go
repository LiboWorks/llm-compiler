package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ShellBackendImpl implements ShellBackend using os/exec.
type ShellBackendImpl struct {
	shell string // e.g., "sh", "bash", "zsh"
}

// ShellConfig holds configuration for the shell backend.
type ShellConfig struct {
	// Shell is the shell to use (default: "sh")
	Shell string
}

// NewShellBackend creates a new shell backend.
func NewShellBackend(cfg ShellConfig) *ShellBackendImpl {
	shell := cfg.Shell
	if shell == "" {
		shell = "sh"
	}
	return &ShellBackendImpl{shell: shell}
}

// Run implements ShellBackend.
func (s *ShellBackendImpl) Run(ctx context.Context, command string) (string, error) {
	return s.RunWithEnv(ctx, command, nil)
}

// RunWithEnv implements ShellBackend.
func (s *ShellBackendImpl) RunWithEnv(ctx context.Context, command string, env map[string]string) (string, error) {
	cmd := exec.CommandContext(ctx, s.shell, "-c", command)

	// Inherit parent environment and add extras
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include output in error for debugging
		return string(output), fmt.Errorf("shell command failed: %w\noutput: %s", err, strings.TrimSpace(string(output)))
	}

	return string(output), nil
}
