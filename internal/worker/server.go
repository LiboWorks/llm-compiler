package worker

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Server runs inside a spawned worker process and handles incoming requests
type Server struct {
	handler   Handler
	statusOut io.Writer
	mu        sync.Mutex
}

// NewServer creates a new worker server with the given handler
func NewServer(handler Handler) *Server {
	return &Server{handler: handler}
}

// Run starts the server loop, reading requests from stdin and writing responses to stdout.
// This method blocks until stdin is closed or an error occurs.
func (s *Server) Run() {
	// Announce worker start so callers can observe init ran.
	// Write status/log messages to fd 3 (if provided) so the parent can
	// capture worker fmt logs separately from native stderr. Fallback to
	// os.Stderr if fd 3 is not available.
	statusOut := os.NewFile(uintptr(3), "worker-fmt")
	var statusWriter *bufio.Writer
	if statusOut != nil {
		statusWriter = bufio.NewWriter(statusOut)
		statusWriter.WriteString(fmt.Sprintf("LLMC worker: starting (pid=%d)\n", os.Getpid()))
		statusWriter.Flush()
		s.statusOut = statusWriter
	} else {
		fmt.Fprintf(os.Stderr, "LLMC worker: starting (pid=%d)\n", os.Getpid())
		s.statusOut = os.Stderr
	}

	w := bufio.NewWriter(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Bytes()
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := Response{ID: req.ID, Val: "", Err: fmt.Sprintf("invalid request: %v", err)}
			enc.Encode(resp)
			w.Flush()
			continue
		}

		s.mu.Lock()
		val, err := s.handler.Generate(req.Prompt, req.ModelSpec, req.MaxTokens)
		s.mu.Unlock()

		resp := Response{ID: req.ID, Val: val}
		if err != nil {
			resp.Err = err.Error()
		}
		enc.Encode(resp)
		w.Flush()
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		if statusWriter != nil {
			statusWriter.WriteString(fmt.Sprintf("worker input error: %v\n", err))
			statusWriter.Flush()
		} else {
			fmt.Fprintf(os.Stderr, "worker input error: %v\n", err)
		}
	}
	if statusWriter != nil {
		statusWriter.WriteString(fmt.Sprintf("LLMC worker: exiting (pid=%d)\n", os.Getpid()))
		statusWriter.Flush()
	} else {
		fmt.Fprintf(os.Stderr, "LLMC worker: exiting (pid=%d)\n", os.Getpid())
	}
	if statusOut != nil {
		statusOut.Close()
	}
}

// WriteStatus writes a status message to the status output (fd3 or stderr)
func (s *Server) WriteStatus(format string, args ...interface{}) {
	if s.statusOut != nil {
		fmt.Fprintf(s.statusOut, format, args...)
		if bw, ok := s.statusOut.(*bufio.Writer); ok {
			bw.Flush()
		}
	}
}
