package runtime

import (
	"os"

	"github.com/LiboWorks/llm-compiler/internal/worker"
)

// localLlamaHandler adapts LocalLlamaRuntime to the worker.Handler interface
type localLlamaHandler struct {
	llama *LocalLlamaRuntime
}

func (h *localLlamaHandler) Generate(prompt, modelSpec string, maxTokens int) (string, error) {
	return h.llama.Generate(prompt, modelSpec, maxTokens)
}

func init() {
	if worker.IsWorkerProcess() {
		// Create handler with local llama runtime
		handler := &localLlamaHandler{llama: NewLocalLlamaRuntime()}
		// Run the worker server
		server := worker.NewServer(handler)
		server.Run()
		os.Exit(0)
	}
}
