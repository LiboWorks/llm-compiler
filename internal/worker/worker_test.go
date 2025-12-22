package worker_test

import (
	"testing"

	"github.com/LiboWorks/llm-compiler/internal/worker"
)

func TestIsWorkerProcess(t *testing.T) {
	// In normal test context, should be false
	if worker.IsWorkerProcess() {
		t.Error("IsWorkerProcess() should be false in test context")
	}
}

func TestRequestResponse(t *testing.T) {
	// Test Request struct
	req := worker.Request{
		ID:        "1",
		ModelSpec: "/path/to/model.gguf",
		Prompt:    "Hello",
		MaxTokens: 100,
	}

	if req.ID != "1" {
		t.Errorf("Request.ID = %q, want %q", req.ID, "1")
	}
	if req.ModelSpec != "/path/to/model.gguf" {
		t.Errorf("Request.ModelSpec = %q, want %q", req.ModelSpec, "/path/to/model.gguf")
	}
	if req.MaxTokens != 100 {
		t.Errorf("Request.MaxTokens = %d, want %d", req.MaxTokens, 100)
	}

	// Test Response struct
	resp := worker.Response{
		ID:  "1",
		Val: "Generated text",
		Err: "",
	}

	if resp.ID != "1" {
		t.Errorf("Response.ID = %q, want %q", resp.ID, "1")
	}
	if resp.Val != "Generated text" {
		t.Errorf("Response.Val = %q, want %q", resp.Val, "Generated text")
	}
}

func TestPoolCreation(t *testing.T) {
	// Skip if we can't create workers (no compiled binary)
	t.Skip("Skipping pool test - requires compiled binary")

	pool, err := worker.NewPool(2)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()

	if pool.Size() != 2 {
		t.Errorf("Pool.Size() = %d, want 2", pool.Size())
	}
}
