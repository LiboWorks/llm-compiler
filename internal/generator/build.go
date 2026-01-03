package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// getModuleRoot finds the llm-compiler source directory using runtime caller info.
func getModuleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	// file is .../llm-compiler/internal/generator/build.go
	dir := filepath.Dir(file)     // internal/generator
	dir = filepath.Dir(dir)       // internal
	return filepath.Dir(dir)      // llm-compiler root
}

// BuildOptions configures the build process.
type BuildOptions struct {
	// OutputDir is where the final binary will be placed.
	OutputDir string

	// OutputName is the name of the binary (without extension).
	OutputName string

	// KeepSource keeps the generated .go file for debugging.
	// If false (default), the source is deleted after build.
	KeepSource bool

	// SourceDir overrides where to save the source file when KeepSource is true.
	// Defaults to OutputDir.
	SourceDir string
}

// BuildResult contains the paths to generated artifacts.
type BuildResult struct {
	// BinaryPath is the path to the compiled binary.
	BinaryPath string

	// SourcePath is the path to the generated source (only if KeepSource was true).
	SourcePath string
}

// BuildFromCode compiles generated Go code into a standalone binary.
// The code is compiled within the llm-compiler module context, allowing
// direct import of internal packages.
func BuildFromCode(code string, opts *BuildOptions) (*BuildResult, error) {
	if opts == nil {
		opts = &BuildOptions{}
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if opts.OutputName == "" {
		opts.OutputName = "workflow"
	}

	moduleRoot := getModuleRoot()
	if moduleRoot == "" {
		return nil, fmt.Errorf("could not determine llm-compiler module root")
	}

	// Create temp directory inside the module for compilation
	// Using internal/generated to keep it clearly within the module
	tempDir := filepath.Join(moduleRoot, "internal", "generated")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp build dir: %w", err)
	}

	// Write generated code
	sourceFile := filepath.Join(tempDir, opts.OutputName+".go")
	if err := os.WriteFile(sourceFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write generated code: %w", err)
	}

	// Ensure output directory exists
	absOutputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("invalid output dir: %w", err)
	}
	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output dir: %w", err)
	}

	// Build binary with output path pointing to user's directory
	binaryPath := filepath.Join(absOutputDir, opts.OutputName)
	
	fmt.Printf("🔨 Building %s...\n", binaryPath)
	
	cmd := exec.Command("go", "build", "-o", binaryPath, sourceFile)
	cmd.Dir = moduleRoot // Build from module root so internal imports work
	
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up on failure
		os.Remove(sourceFile)
		return nil, fmt.Errorf("build error: %v\n%s", err, string(out))
	}

	result := &BuildResult{
		BinaryPath: binaryPath,
	}

	// Handle source file
	if opts.KeepSource {
		// Move source to output dir (or specified source dir)
		destDir := opts.SourceDir
		if destDir == "" {
			destDir = opts.OutputDir
		}
		destPath := filepath.Join(destDir, opts.OutputName+".go")
		
		// Copy if different location
		if destPath != sourceFile {
			if err := copyFile(sourceFile, destPath); err != nil {
				fmt.Printf("⚠️  Warning: failed to save source: %v\n", err)
			} else {
				result.SourcePath = destPath
			}
		} else {
			result.SourcePath = sourceFile
		}
	}
	
	// Clean up temp source file
	os.Remove(sourceFile)

	return result, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// SaveToFile writes generated code to a file (for --keep-source or debugging).
func SaveToFile(path string, code string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(path, []byte(code), 0644)
}

// Deprecated: BuildGoFile is kept for backward compatibility.
// Use BuildFromCode instead.
func BuildGoFile(sourcePath string) error {
	code, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	
	dir := filepath.Dir(sourcePath)
	name := strings.TrimSuffix(filepath.Base(sourcePath), ".go")
	
	_, err = BuildFromCode(string(code), &BuildOptions{
		OutputDir:  dir,
		OutputName: name,
		KeepSource: true,
		SourceDir:  dir,
	})
	return err
}
