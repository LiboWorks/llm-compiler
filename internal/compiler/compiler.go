// Package compiler provides the core compilation logic for llm-compiler.
package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LiboWorks/llm-compiler/internal/generator"
	"github.com/LiboWorks/llm-compiler/internal/workflow"
)

// Options configures the compilation process.
type Options struct {
	// OutputDir is the directory where the binary will be placed.
	OutputDir string

	// OutputName overrides the output binary name.
	// If empty, uses the input filename (without extension).
	OutputName string

	// SkipBuild generates the Go code but skips compilation.
	// Use with KeepSource to inspect generated code.
	SkipBuild bool

	// KeepSource saves the generated .go file alongside the binary.
	// Useful for debugging or inspection.
	KeepSource bool

	// Verbose enables detailed output during compilation.
	Verbose bool
}

// Result contains the results of a successful compilation.
type Result struct {
	// SourceFile is the path to the generated Go source file.
	// Only set if KeepSource was true or SkipBuild was true.
	SourceFile string

	// BinaryFile is the path to the compiled binary.
	// Empty if SkipBuild was true.
	BinaryFile string

	// Workflows contains the parsed workflow definitions.
	Workflows []workflow.Workflow
}

// CompileFile compiles a YAML workflow file into a standalone binary.
func CompileFile(inputPath string, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}

	// Check file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("workflow file not found: %s", inputPath)
	}

	// Load workflows
	wfs, err := workflow.LoadWorkflows(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflows: %w", err)
	}

	// Validate workflows
	for _, wf := range wfs {
		if err := wf.Validate(); err != nil {
			return nil, fmt.Errorf("validation error in %s: %w", wf.Name, err)
		}
	}

	// Determine output name
	outputName := opts.OutputName
	if outputName == "" {
		baseName := filepath.Base(inputPath)
		outputName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	}

	// Generate code
	code, err := generator.Generate(wfs)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	result := &Result{
		Workflows: wfs,
	}

	// Handle SkipBuild case - just save source file
	if opts.SkipBuild {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output dir: %w", err)
		}
		sourcePath := filepath.Join(opts.OutputDir, outputName+".go")
		if err := generator.SaveToFile(sourcePath, code); err != nil {
			return nil, fmt.Errorf("failed to save generated file: %w", err)
		}
		result.SourceFile = sourcePath
		return result, nil
	}

	// Build binary (code is compiled in temp dir, binary goes to output dir)
	buildResult, err := generator.BuildFromCode(code, &generator.BuildOptions{
		OutputDir:  opts.OutputDir,
		OutputName: outputName,
		KeepSource: opts.KeepSource,
		SourceDir:  opts.OutputDir,
	})
	if err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	result.BinaryFile = buildResult.BinaryPath
	result.SourceFile = buildResult.SourcePath

	return result, nil
}

// Compile compiles workflow structs into a standalone binary.
func Compile(wfs []workflow.Workflow, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "."
	}
	if opts.OutputName == "" {
		if len(wfs) == 1 {
			opts.OutputName = wfs[0].Name
		} else {
			opts.OutputName = "workflows"
		}
	}

	// Validate workflows
	for _, wf := range wfs {
		if err := wf.Validate(); err != nil {
			return nil, fmt.Errorf("validation error in %s: %w", wf.Name, err)
		}
	}

	// Generate code
	code, err := generator.Generate(wfs)
	if err != nil {
		return nil, fmt.Errorf("code generation failed: %w", err)
	}

	result := &Result{
		Workflows: wfs,
	}

	// Handle SkipBuild case
	if opts.SkipBuild {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output dir: %w", err)
		}
		sourcePath := filepath.Join(opts.OutputDir, opts.OutputName+".go")
		if err := generator.SaveToFile(sourcePath, code); err != nil {
			return nil, fmt.Errorf("failed to save generated file: %w", err)
		}
		result.SourceFile = sourcePath
		return result, nil
	}

	// Build binary
	buildResult, err := generator.BuildFromCode(code, &generator.BuildOptions{
		OutputDir:  opts.OutputDir,
		OutputName: opts.OutputName,
		KeepSource: opts.KeepSource,
		SourceDir:  opts.OutputDir,
	})
	if err != nil {
		return nil, fmt.Errorf("build failed: %w", err)
	}

	result.BinaryFile = buildResult.BinaryPath
	result.SourceFile = buildResult.SourcePath

	return result, nil
}
