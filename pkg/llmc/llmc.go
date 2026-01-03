// Package llmc provides a public API for the llm-compiler.
//
// This package allows programmatic compilation of YAML workflow definitions
// into standalone Go binaries with embedded LLM inference capabilities.
//
// Basic usage:
//
//	result, err := llmc.CompileFile("workflow.yaml", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Binary:", result.BinaryPath)
//
// With options:
//
//	result, err := llmc.CompileFile("workflow.yaml", &llmc.CompileOptions{
//	    OutputDir:  "./dist",
//	    OutputName: "my-agent",
//	})
//
// Programmatic workflow construction:
//
//	wf := llmc.NewWorkflow("my-workflow")
//	wf.AddStep(llmc.ShellStep("greet", "echo 'Hello'").WithOutput("greeting"))
//	wf.AddStep(llmc.LLMStep("respond", "Respond to: {{greeting}}"))
//
//	result, err := llmc.Compile([]*llmc.Workflow{wf}, nil)
package llmc

import (
	"github.com/LiboWorks/llm-compiler/internal/compiler"
	"github.com/LiboWorks/llm-compiler/internal/workflow"
)

// CompileOptions configures the compilation process.
type CompileOptions struct {
	// OutputDir is the directory where the binary will be placed.
	// Defaults to current directory if not specified.
	OutputDir string

	// OutputName overrides the output binary name.
	// If empty, uses the input filename (without extension) for CompileFile,
	// or the workflow name for Compile.
	OutputName string

	// SkipBuild generates the Go source file but skips compilation to binary.
	// The source file will be saved to OutputDir.
	SkipBuild bool

	// KeepSource saves the generated .go file alongside the binary.
	// Useful for debugging or inspecting the generated code.
	// By default, only the binary is output.
	KeepSource bool

	// Verbose enables detailed output during compilation.
	Verbose bool
}

// CompileResult contains the results of a successful compilation.
type CompileResult struct {
	// SourcePath is the path to the generated Go source file.
	SourcePath string

	// BinaryPath is the path to the compiled binary.
	// Empty if SkipBuild was true.
	BinaryPath string

	// Workflows contains the compiled workflow definitions.
	Workflows []*Workflow
}

// CompileFile compiles a YAML workflow file into a standalone binary.
//
// The input file can contain single or multiple workflow definitions
// separated by YAML document markers (---).
//
// Example:
//
//	result, err := llmc.CompileFile("workflow.yaml", &llmc.CompileOptions{
//	    OutputDir: "./build",
//	})
func CompileFile(inputPath string, opts *CompileOptions) (*CompileResult, error) {
	internalOpts := toInternalOptions(opts)

	result, err := compiler.CompileFile(inputPath, internalOpts)
	if err != nil {
		return nil, err
	}

	return fromInternalResult(result), nil
}

// Compile compiles workflow definitions into a standalone binary.
//
// Use this for programmatically constructed workflows. For YAML files,
// use CompileFile instead.
//
// Example:
//
//	wf := llmc.NewWorkflow("my-workflow")
//	wf.AddStep(llmc.ShellStep("greet", "echo 'Hello'"))
//
//	result, err := llmc.Compile([]*llmc.Workflow{wf}, nil)
func Compile(workflows []*Workflow, opts *CompileOptions) (*CompileResult, error) {
	internalOpts := toInternalOptions(opts)

	// Convert public workflows to internal format
	internalWfs := make([]workflow.Workflow, len(workflows))
	for i, wf := range workflows {
		internalWfs[i] = wf.toInternal()
	}

	result, err := compiler.Compile(internalWfs, internalOpts)
	if err != nil {
		return nil, err
	}

	return fromInternalResult(result), nil
}

// LoadWorkflows loads and parses workflow definitions from a YAML file
// without compiling them. Useful for inspection or modification before
// compilation.
//
// Example:
//
//	workflows, err := llmc.LoadWorkflows("workflow.yaml")
//	for _, wf := range workflows {
//	    fmt.Printf("Workflow: %s (%d steps)\n", wf.Name, len(wf.Steps))
//	}
func LoadWorkflows(inputPath string) ([]*Workflow, error) {
	wfs, err := workflow.LoadWorkflows(inputPath)
	if err != nil {
		return nil, err
	}

	result := make([]*Workflow, len(wfs))
	for i, wf := range wfs {
		result[i] = fromInternalWorkflow(wf)
	}
	return result, nil
}

// Validate checks a workflow for errors without compiling it.
func Validate(wf *Workflow) error {
	internal := wf.toInternal()
	return internal.Validate()
}

// Helper functions for conversion

func toInternalOptions(opts *CompileOptions) *compiler.Options {
	if opts == nil {
		return nil
	}
	return &compiler.Options{
		OutputDir:  opts.OutputDir,
		OutputName: opts.OutputName,
		SkipBuild:  opts.SkipBuild,
		KeepSource: opts.KeepSource,
		Verbose:    opts.Verbose,
	}
}

func fromInternalResult(r *compiler.Result) *CompileResult {
	workflows := make([]*Workflow, len(r.Workflows))
	for i, wf := range r.Workflows {
		workflows[i] = fromInternalWorkflow(wf)
	}
	return &CompileResult{
		SourcePath: r.SourceFile,
		BinaryPath: r.BinaryFile,
		Workflows:  workflows,
	}
}
