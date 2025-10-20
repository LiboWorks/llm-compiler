package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	llama "github.com/go-skynet/go-llama.cpp" // binding
)

// LocalLlamaRuntime manages loaded models (cached) and generation.
type LocalLlamaRuntime struct {
	mu     sync.Mutex
	models map[string]*llama.Model // model handle from binding
	opts   llama.ModelOptions      // default options if binding uses this
}

func NewLocalLlamaRuntime() *LocalLlamaRuntime {
	return &LocalLlamaRuntime{
		models: make(map[string]*llama.Model),
	}
}

// LoadModel loads a gguf model from filePath (caches handle).
func (r *LocalLlamaRuntime) LoadModel(filePath string) (*llama.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	abs, _ := filepath.Abs(filePath)
	if m, ok := r.models[abs]; ok {
		return m, nil
	}

	// Example: the binding may expose a NewModel or LoadModel function:
	model, err := llama.NewModelFromFile(abs, llama.Config{
		// set defaults: threads, n_ctx etc.
		// Thread count: use runtime.NumCPU() or env var
		Threads: 4,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load model %s: %w", abs, err)
	}

	r.models[abs] = model
	return model, nil
}

// Generate runs the model with prompt and returns the completion text.
// modelSpec expected to be "file:/absolute/or/relative/path.gguf" OR logical name mapped to path.
func (r *LocalLlamaRuntime) Generate(prompt string, modelSpec string) (string, error) {
	// Determine path
	var modelPath string
	if strings.HasPrefix(modelSpec, "file:") {
		modelPath = strings.TrimPrefix(modelSpec, "file:")
	} else {
		// If not file: treat as logical name, map to ./models/<name>.gguf
		modelPath = "./models/" + modelSpec + ".gguf"
	}

	model, err := r.LoadModel(modelPath)
	if err != nil {
		return "", err
	}

	// Call the binding's generate/predict API; exact API may differ.
	resp, err := model.Predict(prompt, llama.PredictOptions{
		TopK: 40,
		TopP: 0.9,
		Temp: 0.8,
		// MaxTokens, Stop, etc.
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}
