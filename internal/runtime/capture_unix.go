//go:build unix

package runtime

import (
	"io"
	"os"
	"syscall"
)

// setupPlatformCapture sets up platform-specific output capture.
// On Unix, this uses syscall.Dup/Dup2 to redirect native file descriptors.
func (oc *OutputCapture) setupPlatformCapture() (*os.File, *os.File, error) {
	// Save original terminal fds
	savedStdoutFd, err := syscall.Dup(int(os.Stdout.Fd()))
	if err != nil {
		return nil, nil, err
	}
	savedStderrFd, err := syscall.Dup(int(os.Stderr.Fd()))
	if err != nil {
		syscall.Close(savedStdoutFd)
		return nil, nil, err
	}
	savedStdout := os.NewFile(uintptr(savedStdoutFd), "saved_stdout")
	savedStderr := os.NewFile(uintptr(savedStderrFd), "saved_stderr")

	// Duplicate the fmt file descriptor to fd 3 for subprocess workers
	syscall.Dup2(int(oc.fmtFile.Fd()), 3)

	// Native-level output capture: redirect fd 1/2 to separate pipes
	rCOut, wCOut, _ := os.Pipe()
	rCErr, wCErr, _ := os.Pipe()
	oc.wCOut = wCOut
	oc.wCErr = wCErr
	syscall.Dup2(int(wCOut.Fd()), 1)
	syscall.Dup2(int(wCErr.Fd()), 2)

	// Copy native-level output to both terminal and llamaFile
	oc.wg.Add(2)
	go func() {
		defer oc.wg.Done()
		defer rCOut.Close()
		io.Copy(io.MultiWriter(savedStdout, oc.llamaFile), rCOut)
	}()
	go func() {
		defer oc.wg.Done()
		defer rCErr.Close()
		io.Copy(io.MultiWriter(savedStderr, oc.llamaFile), rCErr)
	}()

	return savedStdout, savedStderr, nil
}

// cleanupPlatformCapture restores original file descriptors on Unix.
func (oc *OutputCapture) cleanupPlatformCapture() {
	// Close the pipe write ends FIRST - this signals EOF to the io.Copy goroutines
	if oc.wCOut != nil {
		oc.wCOut.Close()
	}
	if oc.wCErr != nil {
		oc.wCErr.Close()
	}
	
	// Restore native fd 1 and 2 to terminal (not os.Stdout.Fd() which is the Go pipe)
	// This also closes the dup'd references to the pipes at fd 1/2
	if oc.savedStdout != nil {
		syscall.Dup2(int(oc.savedStdout.Fd()), 1)
	}
	if oc.savedStderr != nil {
		syscall.Dup2(int(oc.savedStderr.Fd()), 2)
	}
}
