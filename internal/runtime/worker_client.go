package runtime

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

type workerClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	enc *json.Encoder

	pendingMu sync.Mutex
	pending   map[string]chan WorkerResponse

	idCounter uint64
}

func newWorkerClient() (*workerClient, error) {
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

	// redirect child's stderr to parent's stderr to surface errors
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	wc := &workerClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		enc:     json.NewEncoder(stdin),
		pending: make(map[string]chan WorkerResponse),
	}

	go wc.readLoop()

	return wc, nil
}

func (w *workerClient) readLoop() {
	dec := json.NewDecoder(bufio.NewReader(w.stdout))
	for {
		var resp WorkerResponse
		if err := dec.Decode(&resp); err != nil {
			if err == io.EOF {
				return
			}
			fmt.Fprintf(os.Stderr, "worker client decode error: %v\n", err)
			return
		}

		w.pendingMu.Lock()
		ch, ok := w.pending[resp.ID]
		if ok {
			delete(w.pending, resp.ID)
		}
		w.pendingMu.Unlock()

		if ok {
			ch <- resp
			close(ch)
		}
	}
}

func (w *workerClient) sendRequest(modelSpec, prompt string, maxTokens int) (string, error) {
	id := fmt.Sprintf("%d", atomic.AddUint64(&w.idCounter, 1))
	req := WorkerRequest{ID: id, ModelSpec: modelSpec, Prompt: prompt, MaxTokens: maxTokens}

	ch := make(chan WorkerResponse, 1)
	w.pendingMu.Lock()
	w.pending[id] = ch
	w.pendingMu.Unlock()

	if err := w.enc.Encode(req); err != nil {
		return "", err
	}

	resp := <-ch
	if resp.Err != "" {
		return resp.Val, fmt.Errorf(resp.Err)
	}
	return resp.Val, nil
}

func (w *workerClient) Close() error {
	// closing stdin will cause worker to exit its read loop
	w.stdin.Close()
	return w.cmd.Wait()
}
