// Package worker provides subprocess worker management for llm-compiler.
// It handles spawning worker processes, communication protocol, and lifecycle management.
package worker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// Request is sent from client to worker over stdin as JSON newline.
type Request struct {
	ID        string `json:"id"`
	ModelSpec string `json:"model_spec"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
}

// Response is sent from worker to client over stdout as JSON newline.
type Response struct {
	ID  string `json:"id"`
	Val string `json:"val"`
	Err string `json:"err"`
}

// Handler is the interface that must be implemented to handle worker requests
type Handler interface {
	// Generate processes a prompt and returns the generated text
	Generate(prompt, modelSpec string, maxTokens int) (string, error)
}

// Client manages communication with a worker subprocess
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	enc *json.Encoder

	pendingMu sync.Mutex
	pending   map[string]chan Response

	idCounter uint64
}

// NewClient creates and starts a new worker subprocess
func NewClient() (*Client, error) {
	return NewClientWithFd(nil)
}

// NewClientWithFd creates a worker subprocess and passes the given file as fd3
func NewClientWithFd(fd3File *os.File) (*Client, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}

	// Use absolute path to executable
	exe, _ = filepath.Abs(exe)

	cmd := exec.Command(exe)
	// Build child environment: inherit parent's env but remove LLMC_SUBPROCESS
	// to avoid recursive worker spawning. Then set LLMC_WORKER=1 for the child.
	parentEnv := os.Environ()
	childEnv := make([]string, 0, len(parentEnv)+1)
	for _, e := range parentEnv {
		// filter out LLMC_SUBPROCESS to avoid child thinking it's the parent
		if strings.HasPrefix(e, "LLMC_SUBPROCESS=") {
			continue
		}
		childEnv = append(childEnv, e)
	}
	childEnv = append(childEnv, "LLMC_WORKER=1")
	cmd.Env = childEnv

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Pass fd3 to child for fmt output capture if provided
	// If not provided, try to use the inherited fd3
	if fd3File != nil {
		cmd.ExtraFiles = []*os.File{fd3File}
	} else {
		// Try to inherit fd3 from parent
		inheritedFd3 := os.NewFile(uintptr(3), "fmt_output_fd")
		if inheritedFd3 != nil {
			cmd.ExtraFiles = []*os.File{inheritedFd3}
		}
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	wc := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		enc:     json.NewEncoder(stdin),
		pending: make(map[string]chan Response),
	}

	go wc.readLoop()

	return wc, nil
}

// readLoop reads responses from the worker subprocess
func (c *Client) readLoop() {
	dec := json.NewDecoder(bufio.NewReader(c.stdout))
	for {
		var resp Response
		if err := dec.Decode(&resp); err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "worker client decode error: %v\n", err)
			return
		}

		c.pendingMu.Lock()
		ch, ok := c.pending[resp.ID]
		if ok {
			delete(c.pending, resp.ID)
		}
		c.pendingMu.Unlock()

		if ok {
			ch <- resp
			close(ch)
		}
	}
}

// SendRequest sends a request to the worker and waits for the response
func (c *Client) SendRequest(modelSpec, prompt string, maxTokens int) (string, error) {
	id := fmt.Sprintf("%d", atomic.AddUint64(&c.idCounter, 1))
	req := Request{ID: id, ModelSpec: modelSpec, Prompt: prompt, MaxTokens: maxTokens}

	ch := make(chan Response, 1)
	c.pendingMu.Lock()
	c.pending[id] = ch
	c.pendingMu.Unlock()

	if err := c.enc.Encode(req); err != nil {
		return "", err
	}

	resp := <-ch
	if resp.Err != "" {
		return resp.Val, fmt.Errorf("%s", resp.Err)
	}
	return resp.Val, nil
}

// Close shuts down the worker subprocess
func (c *Client) Close() error {
	// closing stdin will cause worker to exit its read loop
	c.stdin.Close()
	return c.cmd.Wait()
}

// Pid returns the process ID of the worker subprocess
func (c *Client) Pid() int {
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Pid
	}
	return 0
}
