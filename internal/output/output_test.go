package output_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LiboWorks/llm-compiler/internal/output"
)

func TestCapturerWriteToFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_output.txt")

	capturer := output.NewCapturer()

	// First write should truncate
	err := capturer.WriteToFile(filePath, "first write\n")
	if err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	// Second write should append
	err = capturer.WriteToFile(filePath, "second write\n")
	if err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := "first write\nsecond write\n"
	if string(data) != expected {
		t.Errorf("file content = %q, want %q", string(data), expected)
	}
}

func TestCapturerAppendToFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_append.txt")

	capturer := output.NewCapturer()

	// Write initial content
	err := os.WriteFile(filePath, []byte("initial\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Append should not truncate
	err = capturer.AppendToFile(filePath, "appended\n")
	if err != nil {
		t.Fatalf("AppendToFile() error = %v", err)
	}

	// Read and verify
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := "initial\nappended\n"
	if string(data) != expected {
		t.Errorf("file content = %q, want %q", string(data), expected)
	}
}

func TestCapturerTruncateFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_truncate.txt")

	capturer := output.NewCapturer()

	// Write some content
	err := os.WriteFile(filePath, []byte("some content"), 0644)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Truncate
	err = capturer.TruncateFile(filePath)
	if err != nil {
		t.Fatalf("TruncateFile() error = %v", err)
	}

	// Verify file is empty
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("file should be empty, got %d bytes", len(data))
	}
}

func TestCapturerReset(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_reset.txt")

	capturer := output.NewCapturer()

	// First write (truncate)
	err := capturer.WriteToFile(filePath, "first\n")
	if err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	// Reset truncate state
	capturer.ResetTruncateState(filePath)

	// Next write should truncate again
	err = capturer.WriteToFile(filePath, "after reset\n")
	if err != nil {
		t.Fatalf("WriteToFile() error = %v", err)
	}

	// Verify only "after reset" is in file
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := "after reset\n"
	if string(data) != expected {
		t.Errorf("file content = %q, want %q", string(data), expected)
	}
}

func TestGlobalCapturer(t *testing.T) {
	capturer := output.GetCapturer()
	if capturer == nil {
		t.Error("GetCapturer() should not return nil")
	}

	// Multiple calls should return same instance
	capturer2 := output.GetCapturer()
	if capturer != capturer2 {
		t.Error("GetCapturer() should return same instance")
	}
}

func TestTeeCapture(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "original.txt")
	capturePath := filepath.Join(tmpDir, "capture.txt")

	original, err := os.Create(originalPath)
	if err != nil {
		t.Fatalf("failed to create original file: %v", err)
	}
	defer original.Close()

	capture, err := os.Create(capturePath)
	if err != nil {
		t.Fatalf("failed to create capture file: %v", err)
	}
	defer capture.Close()

	tee := output.NewTeeCapture(original, capture)

	// Write through tee
	_, err = tee.Write([]byte("test data"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	// Flush files
	original.Close()
	capture.Close()

	// Verify both files have the data
	originalData, _ := os.ReadFile(originalPath)
	captureData, _ := os.ReadFile(capturePath)

	if string(originalData) != "test data" {
		t.Errorf("original file = %q, want %q", string(originalData), "test data")
	}
	if string(captureData) != "test data" {
		t.Errorf("capture file = %q, want %q", string(captureData), "test data")
	}
}
