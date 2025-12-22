package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/LiboWorks/llm-compiler/internal/llama"
)

// LlamaBackend implements LLMBackend using local llama.cpp inference.
type LlamaBackend struct {
	mu     sync.Mutex
	models map[string]*llama.Model

	// Worker client for subprocess-based inference (optional).
	// When set, inference is delegated to a subprocess to avoid
	// ggml/llama C-level concurrency issues.
	worker WorkerClient

	// Default generation options
	defaultMaxTokens int
	defaultTopK      int
	defaultTopP      float64
	defaultTemp      float64
}

// WorkerClient is the interface for subprocess worker communication.
// This allows the backend to delegate inference to isolated processes.
type WorkerClient interface {
	SendRequest(modelSpec, prompt string, maxTokens int) (string, error)
	Close() error
}

// LlamaConfig holds configuration for the Llama backend.
type LlamaConfig struct {
	// UseSubprocess enables subprocess-based inference for concurrency safety.
	UseSubprocess bool

	// WorkerClient is an optional pre-configured worker client.
	// If nil and UseSubprocess is true, a new worker will be created.
	WorkerClient WorkerClient

	// Default generation parameters
	MaxTokens int
	TopK      int
	TopP      float64
	Temp      float64
}

// NewLlamaBackend creates a new local llama backend.
func NewLlamaBackend(cfg LlamaConfig) *LlamaBackend {
	b := &LlamaBackend{
		models:           make(map[string]*llama.Model),
		defaultMaxTokens: 256,
		defaultTopK:      40,
		defaultTopP:      0.9,
		defaultTemp:      0.8,
	}

	if cfg.MaxTokens > 0 {
		b.defaultMaxTokens = cfg.MaxTokens
	}
	if cfg.TopK > 0 {
		b.defaultTopK = cfg.TopK
	}
	if cfg.TopP > 0 {
		b.defaultTopP = cfg.TopP
	}
	if cfg.Temp > 0 {
		b.defaultTemp = cfg.Temp
	}

	if cfg.WorkerClient != nil {
		b.worker = cfg.WorkerClient
	}

	return b
}

// SetWorker sets the worker client for subprocess-based inference.
func (b *LlamaBackend) SetWorker(w WorkerClient) {
	b.worker = w
}

// LoadModel loads a GGUF model from the given path.
func (b *LlamaBackend) LoadModel(modelPath string) (*llama.Model, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	abs, err := filepath.Abs(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model path: %w", err)
	}

	if m, ok := b.models[abs]; ok {
		return m, nil
	}

	model, err := llama.LoadModel(abs, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %s: %w", abs, err)
	}

	b.models[abs] = model
	return model, nil
}

// predictMu serializes in-process Predict calls to avoid C-level concurrency issues.
var predictMu sync.Mutex

// Generate implements LLMBackend.
func (b *LlamaBackend) Generate(ctx context.Context, prompt string, model string, maxTokens int) (string, error) {
	// Validate model path
	if model == "" {
		return "", fmt.Errorf("model path is required for llama backend")
	}

	// Check if file exists
	if _, err := os.Stat(model); os.IsNotExist(err) {
		return "", fmt.Errorf("model file not found: %s", model)
	}

	// Use worker if available
	if b.worker != nil {
		return b.worker.SendRequest(model, prompt, maxTokens)
	}

	// In-process inference
	m, err := b.LoadModel(model)
	if err != nil {
		return "", err
	}

	mt := b.defaultMaxTokens
	if maxTokens > 0 {
		mt = maxTokens
	}

	// Serialize calls to avoid ggml concurrency issues
	predictMu.Lock()
	defer predictMu.Unlock()

	out, err := m.Predict(prompt, llama.PredictOptions{
		MaxTokens: mt,
		TopK:      b.defaultTopK,
		TopP:      float32(b.defaultTopP),
		Temp:      float32(b.defaultTemp),
	})
	if err != nil {
		return "", fmt.Errorf("prediction failed: %w", err)
	}

	return out, nil
}

// Name implements LLMBackend.
func (b *LlamaBackend) Name() string {
	return "llama"
}

// Close implements LLMBackend.
func (b *LlamaBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.worker != nil {
		if err := b.worker.Close(); err != nil {
			return err
		}
	}

	// Note: llama.Model cleanup would go here if the wrapper supports it
	b.models = make(map[string]*llama.Model)
	return nil
}
