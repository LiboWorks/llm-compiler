// Package output provides output capture utilities for llm-compiler.
// It handles redirecting stdout/stderr to files and capturing formatted output.
package output

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Capturer manages output capture to files with proper synchronization
type Capturer struct {
	fmtFile    *os.File
	llamaFile  *os.File
	originalFd int
	mu         sync.Mutex
	truncated  map[string]bool
}

// NewCapturer creates a new output capturer
func NewCapturer() *Capturer {
	return &Capturer{
		truncated: make(map[string]bool),
	}
}

// Global capturer instance for convenience
var globalCapturer = NewCapturer()

// GetCapturer returns the global capturer instance
func GetCapturer() *Capturer {
	return globalCapturer
}

// WriteToFile writes content to a file, truncating on first write, appending thereafter.
// This is the primary method for capturing fmt.Print output.
func (c *Capturer) WriteToFile(filePath, content string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var flag int
	if c.truncated[filePath] {
		flag = os.O_APPEND | os.O_WRONLY | os.O_CREATE
	} else {
		flag = os.O_TRUNC | os.O_WRONLY | os.O_CREATE
		c.truncated[filePath] = true
	}

	f, err := os.OpenFile(filePath, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filePath, err)
	}

	return nil
}

// AppendToFile appends content to a file (always appends, never truncates)
func (c *Capturer) AppendToFile(filePath, content string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filePath, err)
	}

	return nil
}

// TruncateFile truncates a file to zero length or creates it if it doesn't exist
func (c *Capturer) TruncateFile(filePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.OpenFile(filePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to truncate file %s: %w", filePath, err)
	}
	defer f.Close()

	c.truncated[filePath] = true
	return nil
}

// ResetTruncateState resets the truncate state for a file
func (c *Capturer) ResetTruncateState(filePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.truncated, filePath)
}

// Reset resets all state
func (c *Capturer) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.truncated = make(map[string]bool)
}

// Fd3Writer provides a writer that writes to file descriptor 3 (for subprocess communication)
type Fd3Writer struct {
	file *os.File
}

// NewFd3Writer creates a writer for fd3
func NewFd3Writer() (*Fd3Writer, error) {
	// Try to open fd3
	f := os.NewFile(3, "fd3")
	if f == nil {
		return nil, fmt.Errorf("fd3 not available")
	}
	return &Fd3Writer{file: f}, nil
}

// Write implements io.Writer
func (w *Fd3Writer) Write(p []byte) (n int, err error) {
	if w.file == nil {
		return 0, fmt.Errorf("fd3 not available")
	}
	return w.file.Write(p)
}

// Close closes the fd3 writer
func (w *Fd3Writer) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// MultiWriter creates an io.Writer that writes to multiple destinations
func MultiWriter(writers ...io.Writer) io.Writer {
	return io.MultiWriter(writers...)
}

// TeeCapture captures output while also writing to the original destination
type TeeCapture struct {
	original io.Writer
	capture  io.Writer
}

// NewTeeCapture creates a tee capture
func NewTeeCapture(original, capture io.Writer) *TeeCapture {
	return &TeeCapture{
		original: original,
		capture:  capture,
	}
}

// Write implements io.Writer
func (t *TeeCapture) Write(p []byte) (n int, err error) {
	// Write to capture first
	if t.capture != nil {
		t.capture.Write(p)
	}
	// Then write to original
	if t.original != nil {
		return t.original.Write(p)
	}
	return len(p), nil
}
