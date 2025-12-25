//go:build windows

package runtime

import (
	"os"
)

// setupPlatformCapture sets up platform-specific output capture.
// On Windows, syscall.Dup/Dup2 are not available, so we use a simplified
// approach that only captures Go-level output (not native C output).
func (oc *OutputCapture) setupPlatformCapture() (*os.File, *os.File, error) {
	// On Windows, we can't easily redirect native file descriptors.
	// We'll just save references to the original stdout/stderr.
	// The Go-level capture in Start() will still work.
	
	// Create copies of stdout/stderr for terminal output
	// Note: On Windows, we can't dup the actual FDs, so we just use the originals
	savedStdout := os.Stdout
	savedStderr := os.Stderr
	
	return savedStdout, savedStderr, nil
}

// cleanupPlatformCapture is a no-op on Windows since we don't redirect native FDs.
func (oc *OutputCapture) cleanupPlatformCapture() {
	// Nothing to restore on Windows
}
