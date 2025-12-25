// Package runtime provides output capture utilities for generated workflows.
package runtime

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LiboWorks/llm-compiler/internal/config"
)

// OutputCapture manages capturing stdout/stderr to files.
// It handles both Go-level and native (cgo) output redirection.
type OutputCapture struct {
	fmtFile    *os.File
	llamaFile  *os.File
	savedStdout *os.File
	savedStderr *os.File
	wGoOut     *os.File
	wGoErr     *os.File
	wCOut      *os.File
	wCErr      *os.File
	wg         sync.WaitGroup
	initialized bool
}

// NewOutputCapture creates a new output capture instance.
func NewOutputCapture() *OutputCapture {
	return &OutputCapture{}
}

// Start begins capturing output to files next to the executable.
// Returns the saved stdout for printing messages to terminal.
// If LLMC_NO_CAPTURE=1 is set, output capture is skipped.
func (oc *OutputCapture) Start() (*os.File, *os.File, error) {
	// Skip capture if disabled via environment variable
	if os.Getenv("LLMC_NO_CAPTURE") == "1" {
		return os.Stdout, os.Stderr, nil
	}

	cfg := config.Get()
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	fmtOutPath := filepath.Join(exeDir, cfg.FmtOutputFile)
	llamaOutPath := filepath.Join(exeDir, cfg.LlamaOutputFile)

	var err error
	oc.fmtFile, err = os.OpenFile(fmtOutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open fmt output file: %w", err)
	}
	oc.llamaFile, err = os.OpenFile(llamaOutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		oc.fmtFile.Close()
		return nil, nil, fmt.Errorf("failed to open llama output file: %w", err)
	}

	// Write run header
	header := fmt.Sprintf("=== RUN START: %s pid=%d ===\n", time.Now().Format(time.RFC3339), os.Getpid())
	oc.fmtFile.WriteString(header)
	oc.fmtFile.Sync()
	oc.llamaFile.WriteString(header)
	oc.llamaFile.Sync()

	// Platform-specific setup (implemented in capture_*.go files)
	savedStdout, savedStderr, err := oc.setupPlatformCapture()
	if err != nil {
		oc.fmtFile.Close()
		oc.llamaFile.Close()
		return nil, nil, err
	}
	oc.savedStdout = savedStdout
	oc.savedStderr = savedStderr

	// Go-level output capture: replace os.Stdout/os.Stderr with pipe writers
	rGoOut, wGoOut, _ := os.Pipe()
	rGoErr, wGoErr, _ := os.Pipe()
	oc.wGoOut = wGoOut
	oc.wGoErr = wGoErr
	os.Stdout = wGoOut
	os.Stderr = wGoErr

	// Copy Go-level output to both terminal and fmtFile
	oc.wg.Add(2)
	go func() {
		defer oc.wg.Done()
		defer rGoOut.Close()
		io.Copy(io.MultiWriter(savedStdout, oc.fmtFile), rGoOut)
	}()
	go func() {
		defer oc.wg.Done()
		defer rGoErr.Close()
		io.Copy(io.MultiWriter(savedStderr, oc.fmtFile), rGoErr)
	}()

	oc.initialized = true
	return savedStdout, savedStderr, nil
}

// Stop ends output capture and restores original stdout/stderr.
func (oc *OutputCapture) Stop() {
	if !oc.initialized {
		return
	}

	// Close pipe writers FIRST (this will cause the io.Copy goroutines to finish)
	if oc.wGoOut != nil {
		oc.wGoOut.Close()
	}
	if oc.wGoErr != nil {
		oc.wGoErr.Close()
	}

	// Platform-specific cleanup (closes wCOut, wCErr)
	oc.cleanupPlatformCapture()

	// Wait for copy goroutines to finish
	oc.wg.Wait()

	// Flush and close files
	if oc.fmtFile != nil {
		oc.fmtFile.Sync()
		oc.fmtFile.Close()
	}
	if oc.llamaFile != nil {
		oc.llamaFile.Sync()
		oc.llamaFile.Close()
	}
	if oc.savedStdout != nil {
		oc.savedStdout.Close()
	}
	if oc.savedStderr != nil {
		oc.savedStderr.Close()
	}

	oc.initialized = false
}
