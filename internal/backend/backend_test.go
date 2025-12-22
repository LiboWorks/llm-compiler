package backend

import (
	"context"
	"testing"
)

func TestBackendRegistry(t *testing.T) {
	registry := NewRegistry()

	// Register a mock backend
	mockBackend := &mockLLMBackend{}
	registry.RegisterLLM("mock", mockBackend)

	// Retrieve it
	retrieved, ok := registry.GetLLM("mock")
	if !ok || retrieved == nil {
		t.Error("failed to retrieve registered backend")
	}

	// Non-existent backend
	notFound, ok := registry.GetLLM("nonexistent")
	if ok || notFound != nil {
		t.Error("expected not found for non-existent backend")
	}
}

func TestShellBackendRegistry(t *testing.T) {
	registry := NewRegistry()

	// Register shell backend
	shellBackend := &mockShellBackend{}
	registry.RegisterShell(shellBackend)

	// Retrieve it
	retrieved := registry.GetShell()
	if retrieved == nil {
		t.Error("failed to retrieve registered shell backend")
	}
}

func TestDefaultLLM(t *testing.T) {
	registry := NewRegistry()

	// Register first backend - should become default
	mock1 := &mockLLMBackend{name: "mock1"}
	registry.RegisterLLM("mock1", mock1)

	// Get with empty name should return default
	retrieved, ok := registry.GetLLM("")
	if !ok || retrieved == nil {
		t.Error("failed to get default backend")
	}
	if retrieved.Name() != "mock1" {
		t.Errorf("expected mock1, got %s", retrieved.Name())
	}

	// Register second and set as default
	mock2 := &mockLLMBackend{name: "mock2"}
	registry.RegisterLLM("mock2", mock2)
	registry.SetDefaultLLM("mock2")

	retrieved, _ = registry.GetLLM("")
	if retrieved.Name() != "mock2" {
		t.Errorf("expected mock2 as default, got %s", retrieved.Name())
	}
}

// Mock implementations for testing
type mockLLMBackend struct {
	name string
}

func (m *mockLLMBackend) Generate(ctx context.Context, prompt, model string, maxTokens int) (string, error) {
	return "mock response", nil
}

func (m *mockLLMBackend) Name() string {
	if m.name != "" {
		return m.name
	}
	return "mock"
}

func (m *mockLLMBackend) Close() error {
	return nil
}

type mockShellBackend struct{}

func (m *mockShellBackend) Run(ctx context.Context, command string) (string, error) {
	return "mock output", nil
}

func (m *mockShellBackend) RunWithEnv(ctx context.Context, command string, env map[string]string) (string, error) {
	return "mock output", nil
}

func TestShellBackendImpl(t *testing.T) {
	shell := NewShellBackend(ShellConfig{})

	// Test simple echo command
	result, err := shell.Run(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result != "hello\n" {
		t.Errorf("Run() = %q, want %q", result, "hello\n")
	}
}

func TestShellBackendWithError(t *testing.T) {
	shell := NewShellBackend(ShellConfig{})

	// Test command that fails
	_, err := shell.Run(context.Background(), "exit 1")
	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestShellBackendWithPipe(t *testing.T) {
	shell := NewShellBackend(ShellConfig{})

	// Test command with pipe
	result, err := shell.Run(context.Background(), "echo hello | tr 'h' 'H'")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result != "Hello\n" {
		t.Errorf("Run() = %q, want %q", result, "Hello\n")
	}
}

func TestShellBackendWithEnv(t *testing.T) {
	shell := NewShellBackend(ShellConfig{})

	// Test command with environment variable
	result, err := shell.RunWithEnv(context.Background(), "echo $TEST_VAR", map[string]string{
		"TEST_VAR": "test_value",
	})
	if err != nil {
		t.Fatalf("RunWithEnv() error = %v", err)
	}

	if result != "test_value\n" {
		t.Errorf("RunWithEnv() = %q, want %q", result, "test_value\n")
	}
}
