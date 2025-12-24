//go:build !windows

package output

import (
	"fmt"
	"os"
	"syscall"
)

// RedirectStdoutToFile redirects stdout to a file. Call RestoreStdout() to restore.
func (c *Capturer) RedirectStdoutToFile(filePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Save original stdout
	c.originalFd, _ = syscall.Dup(int(os.Stdout.Fd()))

	// Open the file
	var flag int
	if c.truncated[filePath] {
		flag = os.O_APPEND | os.O_WRONLY | os.O_CREATE
	} else {
		flag = os.O_TRUNC | os.O_WRONLY | os.O_CREATE
		c.truncated[filePath] = true
	}

	var err error
	c.llamaFile, err = os.OpenFile(filePath, flag, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for redirect: %w", err)
	}

	// Redirect stdout
	err = syscall.Dup2(int(c.llamaFile.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		c.llamaFile.Close()
		return fmt.Errorf("failed to redirect stdout: %w", err)
	}

	return nil
}

// RestoreStdout restores stdout to its original state
func (c *Capturer) RestoreStdout() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.originalFd == 0 {
		return nil // Nothing to restore
	}

	// Restore original stdout
	err := syscall.Dup2(c.originalFd, int(os.Stdout.Fd()))
	if err != nil {
		return fmt.Errorf("failed to restore stdout: %w", err)
	}

	syscall.Close(c.originalFd)
	c.originalFd = 0

	if c.llamaFile != nil {
		c.llamaFile.Close()
		c.llamaFile = nil
	}

	return nil
}
