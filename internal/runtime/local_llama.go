package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/LiboWorks/llm-compiler/internal/llama"
)

// LocalLlamaRuntime manages loaded models (cached) and generation.
type LocalLlamaRuntime struct {
	mu     sync.Mutex
	models map[string]*llama.Model // model handle from binding
	// Optional worker client for subprocess-backed generation
	worker *workerClient
	// default options; kept simple for the internal wrapper
	// (previously used external binding's ModelOptions)
	// opts field removed because the internal wrapper uses PredictOptions per-call
}

// predictMu serializes Predict calls into the llama binding to avoid
// concurrent access to underlying ggml/llama C contexts which can cause
// KV-cache / sequence-position corruption on some backends (Metal).
var predictMu sync.Mutex

func NewLocalLlamaRuntime() *LocalLlamaRuntime {
	r := &LocalLlamaRuntime{
		models: make(map[string]*llama.Model),
	}

	// If environment opts into subprocess mode, start a worker client.
	if os.Getenv("LLMC_SUBPROCESS") == "1" {
		wc, err := newWorkerClient()
		if err == nil {
			r.worker = wc
		} else {
			// if worker fails, fall back to in-process and surface debug to stderr
			fmt.Fprintf(os.Stderr, "failed to start worker client: %v\n", err)
		}
	}

	return r
}

// LoadModel loads a gguf model from filePath (caches handle).
func (r *LocalLlamaRuntime) LoadModel(filePath string) (*llama.Model, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	abs, _ := filepath.Abs(filePath)
	if m, ok := r.models[abs]; ok {
		return m, nil
	}

	// Use the internal wrapper's LoadModel API
	model, err := llama.LoadModel(abs, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %s: %w", abs, err)
	}

	r.models[abs] = model
	return model, nil
}

// Generate runs the model with prompt and returns the completion text.
// modelSpec expected to be "file:/absolute/or/relative/path.gguf" OR logical name mapped to path.
// maxTokens controls the number of tokens to generate (0 = use default inside runtime).
func (r *LocalLlamaRuntime) Generate(prompt string, modelSpec string, maxTokens int) (string, error) {
	// Determine path
	var modelPath string
	if strings.HasPrefix(modelSpec, "file:") {
		modelPath = strings.TrimPrefix(modelSpec, "file:")
	} else {
		// If modelSpec already looks like a path or already contains the .gguf
		// extension, avoid appending another .gguf.
		if strings.HasPrefix(modelSpec, "/") || strings.HasPrefix(modelSpec, "./") || strings.HasPrefix(modelSpec, "../") {
			modelPath = modelSpec
			if !strings.HasSuffix(modelPath, ".gguf") {
				modelPath = modelPath + ".gguf"
			}
		} else if strings.HasSuffix(modelSpec, ".gguf") {
			// logical name that already includes extension
			modelPath = "./models/" + modelSpec
		} else {
			// fallback: logical name without extension
			modelPath = "./models/" + modelSpec + ".gguf"
		}
	}

	model, err := r.LoadModel(modelPath)
	if err != nil {
		return "", err
	}
	// If worker client is configured, use it for true concurrency.
	if r.worker != nil {
		return r.worker.sendRequest(modelPath, prompt, maxTokens)
	}

	// Call the wrapper's Predict API (in-process). Use provided maxTokens if non-zero, otherwise fall back to 256
	mt := 256
	if maxTokens > 0 {
		mt = maxTokens
	}
	predictMu.Lock()
	out, err := model.Predict(prompt, llama.PredictOptions{
		MaxTokens: mt,
		TopK:      40,
		TopP:      0.9,
		Temp:      0.8,
	})
	predictMu.Unlock()
	if err != nil {
		return "", err
	}
	return out, nil
}
