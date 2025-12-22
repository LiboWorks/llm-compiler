// Package backend defines interfaces for workflow step execution backends.
// This allows easy extension with new backend types (e.g., Anthropic, Ollama)
// and facilitates testing through mock implementations.
package backend

import "context"

// LLMBackend is the interface for language model backends.
// Implementations include OpenAI API, local llama.cpp, and potentially
// other providers like Anthropic, Ollama, etc.
type LLMBackend interface {
	// Generate produces a completion for the given prompt.
	// model specifies which model to use (interpretation is backend-specific).
	// maxTokens limits the response length (0 means use backend default).
	Generate(ctx context.Context, prompt string, model string, maxTokens int) (string, error)

	// Name returns a human-readable name for the backend.
	Name() string

	// Close releases any resources held by the backend.
	Close() error
}

// ShellBackend executes shell commands.
type ShellBackend interface {
	// Run executes a shell command and returns combined stdout/stderr.
	Run(ctx context.Context, command string) (string, error)

	// RunWithEnv executes with additional environment variables.
	RunWithEnv(ctx context.Context, command string, env map[string]string) (string, error)
}

// Registry manages available backends and allows lookup by name.
type Registry struct {
	llmBackends   map[string]LLMBackend
	shellBackend  ShellBackend
	defaultLLM    string
}

// NewRegistry creates an empty backend registry.
func NewRegistry() *Registry {
	return &Registry{
		llmBackends: make(map[string]LLMBackend),
	}
}

// RegisterLLM adds an LLM backend to the registry.
func (r *Registry) RegisterLLM(name string, backend LLMBackend) {
	r.llmBackends[name] = backend
	if r.defaultLLM == "" {
		r.defaultLLM = name
	}
}

// RegisterShell sets the shell backend.
func (r *Registry) RegisterShell(backend ShellBackend) {
	r.shellBackend = backend
}

// SetDefaultLLM sets which LLM backend to use when none is specified.
func (r *Registry) SetDefaultLLM(name string) {
	r.defaultLLM = name
}

// GetLLM returns an LLM backend by name, or the default if name is empty.
func (r *Registry) GetLLM(name string) (LLMBackend, bool) {
	if name == "" {
		name = r.defaultLLM
	}
	b, ok := r.llmBackends[name]
	return b, ok
}

// GetShell returns the shell backend.
func (r *Registry) GetShell() ShellBackend {
	return r.shellBackend
}

// Close releases all backend resources.
func (r *Registry) Close() error {
	for _, b := range r.llmBackends {
		if err := b.Close(); err != nil {
			return err
		}
	}
	return nil
}

// ListLLMBackends returns names of all registered LLM backends.
func (r *Registry) ListLLMBackends() []string {
	names := make([]string, 0, len(r.llmBackends))
	for name := range r.llmBackends {
		names = append(names, name)
	}
	return names
}
