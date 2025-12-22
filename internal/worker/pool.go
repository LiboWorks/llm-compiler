package worker

import (
	"os"

	"github.com/LiboWorks/llm-compiler/internal/config"
)

// IsWorkerProcess returns true if the current process is running as a worker
func IsWorkerProcess() bool {
	return os.Getenv("LLMC_WORKER") == "1"
}

// ShouldUseSubprocess returns true if subprocess mode is enabled
func ShouldUseSubprocess() bool {
	cfg := config.Get()
	return cfg.UseSubprocess
}

// Pool manages a pool of worker clients for parallel processing
type Pool struct {
	clients []*Client
	current int
}

// NewPool creates a new worker pool with the specified number of workers
func NewPool(size int) (*Pool, error) {
	if size <= 0 {
		size = 1
	}

	clients := make([]*Client, size)
	for i := 0; i < size; i++ {
		client, err := NewClient()
		if err != nil {
			// Clean up any clients we've already created
			for j := 0; j < i; j++ {
				clients[j].Close()
			}
			return nil, err
		}
		clients[i] = client
	}

	return &Pool{clients: clients}, nil
}

// Get returns the next client in the pool (round-robin)
func (p *Pool) Get() *Client {
	client := p.clients[p.current]
	p.current = (p.current + 1) % len(p.clients)
	return client
}

// Close shuts down all workers in the pool
func (p *Pool) Close() error {
	var lastErr error
	for _, client := range p.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Size returns the number of workers in the pool
func (p *Pool) Size() int {
	return len(p.clients)
}
