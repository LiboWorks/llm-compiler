package llmc

// Version information for the llm-compiler.
const (
	// Version is the current version of llm-compiler.
	Version = "0.1.0"

	// MinGoVersion is the minimum required Go version.
	MinGoVersion = "1.21"
)

// DefaultOptions returns a new CompileOptions with default values.
func DefaultOptions() *CompileOptions {
	return &CompileOptions{
		OutputDir: ".",
	}
}

// Option is a functional option for configuring compilation.
type Option func(*CompileOptions)

// WithOutputDir sets the output directory.
func WithOutputDir(dir string) Option {
	return func(o *CompileOptions) {
		o.OutputDir = dir
	}
}

// WithOutputName sets the output binary name.
func WithOutputName(name string) Option {
	return func(o *CompileOptions) {
		o.OutputName = name
	}
}

// WithSkipBuild skips the build step, only generating source.
func WithSkipBuild() Option {
	return func(o *CompileOptions) {
		o.SkipBuild = true
	}
}

// WithKeepSource saves the generated .go file alongside the binary.
func WithKeepSource() Option {
	return func(o *CompileOptions) {
		o.KeepSource = true
	}
}

// WithVerbose enables verbose output.
func WithVerbose() Option {
	return func(o *CompileOptions) {
		o.Verbose = true
	}
}

// ApplyOptions applies functional options to CompileOptions.
func ApplyOptions(opts ...Option) *CompileOptions {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// CompileFileWith compiles a workflow file with functional options.
//
// Example:
//
//	result, err := llmc.CompileFileWith("workflow.yaml",
//	    llmc.WithOutputDir("./dist"),
//	    llmc.WithVerbose(),
//	)
func CompileFileWith(inputPath string, opts ...Option) (*CompileResult, error) {
	return CompileFile(inputPath, ApplyOptions(opts...))
}

// CompileWith compiles workflows with functional options.
func CompileWith(workflows []*Workflow, opts ...Option) (*CompileResult, error) {
	return Compile(workflows, ApplyOptions(opts...))
}
