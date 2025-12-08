package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// WorkerRequest is sent from client to worker over stdin as JSON newline.
type WorkerRequest struct {
	ID        string `json:"id"`
	ModelSpec string `json:"model_spec"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
}

// WorkerResponse is sent from worker to client over stdout as JSON newline.
type WorkerResponse struct {
	ID  string `json:"id"`
	Val string `json:"val"`
	Err string `json:"err"`
}

// runWorkerLoop runs inside a spawned worker process when env LLMC_WORKER=1
// It reads newline-delimited JSON WorkerRequest from stdin and writes
// newline-delimited JSON WorkerResponse to stdout.
func runWorkerLoop() {
	// Announce worker start so callers can observe init ran.
	fmt.Fprintf(os.Stderr, "LLMC worker: starting (pid=%d)\n", os.Getpid())

	w := bufio.NewWriter(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)

	// Create an in-process LocalLlamaRuntime to serve requests.
	ll := NewLocalLlamaRuntime()
	var mu sync.Mutex // protect ll calls

	enc := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Bytes()
		var req WorkerRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := WorkerResponse{ID: req.ID, Val: "", Err: fmt.Sprintf("invalid request: %v", err)}
			enc.Encode(resp)
			w.Flush()
			continue
		}

		mu.Lock()
		val, err := ll.Generate(req.Prompt, req.ModelSpec, req.MaxTokens)
		mu.Unlock()

		resp := WorkerResponse{ID: req.ID, Val: val}
		if err != nil {
			resp.Err = err.Error()
		}
		enc.Encode(resp)
		w.Flush()
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "worker input error: %v\n", err)
	}
	fmt.Fprintf(os.Stderr, "LLMC worker: exiting (pid=%d)\n", os.Getpid())
}

func init() {
	if os.Getenv("LLMC_WORKER") == "1" {
		runWorkerLoop()
		os.Exit(0)
	}
}
