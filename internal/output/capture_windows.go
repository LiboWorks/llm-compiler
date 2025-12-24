//go:build windows

package output

import (
	"fmt"
)

// RedirectStdoutToFile is not supported on Windows.
// Returns an error indicating the limitation.
func (c *Capturer) RedirectStdoutToFile(filePath string) error {
	return fmt.Errorf("stdout redirection is not supported on Windows")
}

// RestoreStdout is a no-op on Windows since redirect is not supported.
func (c *Capturer) RestoreStdout() error {
	return nil
}
