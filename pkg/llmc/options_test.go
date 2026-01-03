package llmc_test

import (
	"testing"

	"github.com/LiboWorks/llm-compiler/pkg/llmc"
)

func TestDefaultOptions(t *testing.T) {
	opts := llmc.DefaultOptions()

	if opts.OutputDir != "." {
		t.Errorf("expected default OutputDir '.', got %s", opts.OutputDir)
	}
	if opts.OutputName != "" {
		t.Errorf("expected empty OutputName, got %s", opts.OutputName)
	}
	if opts.SkipBuild {
		t.Error("SkipBuild should be false by default")
	}
	if opts.KeepSource {
		t.Error("KeepSource should be false by default")
	}
	if opts.Verbose {
		t.Error("Verbose should be false by default")
	}
}

func TestWithOutputDir(t *testing.T) {
	opts := llmc.ApplyOptions(llmc.WithOutputDir("/custom/path"))

	if opts.OutputDir != "/custom/path" {
		t.Errorf("expected OutputDir '/custom/path', got %s", opts.OutputDir)
	}
}

func TestWithOutputName(t *testing.T) {
	opts := llmc.ApplyOptions(llmc.WithOutputName("my-binary"))

	if opts.OutputName != "my-binary" {
		t.Errorf("expected OutputName 'my-binary', got %s", opts.OutputName)
	}
}

func TestWithSkipBuild(t *testing.T) {
	opts := llmc.ApplyOptions(llmc.WithSkipBuild())

	if !opts.SkipBuild {
		t.Error("SkipBuild should be true")
	}
}

func TestWithKeepSource(t *testing.T) {
	opts := llmc.ApplyOptions(llmc.WithKeepSource())

	if !opts.KeepSource {
		t.Error("KeepSource should be true")
	}
}

func TestWithVerbose(t *testing.T) {
	opts := llmc.ApplyOptions(llmc.WithVerbose())

	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
}

func TestApplyOptionsChaining(t *testing.T) {
	opts := llmc.ApplyOptions(
		llmc.WithOutputDir("./dist"),
		llmc.WithOutputName("agent"),
		llmc.WithKeepSource(),
		llmc.WithVerbose(),
	)

	if opts.OutputDir != "./dist" {
		t.Errorf("expected OutputDir './dist', got %s", opts.OutputDir)
	}
	if opts.OutputName != "agent" {
		t.Errorf("expected OutputName 'agent', got %s", opts.OutputName)
	}
	if !opts.KeepSource {
		t.Error("KeepSource should be true")
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
}

func TestApplyOptionsOverride(t *testing.T) {
	// Later options should override earlier ones
	opts := llmc.ApplyOptions(
		llmc.WithOutputDir("./first"),
		llmc.WithOutputDir("./second"),
	)

	if opts.OutputDir != "./second" {
		t.Errorf("expected OutputDir './second', got %s", opts.OutputDir)
	}
}

func TestVersion(t *testing.T) {
	if llmc.Version == "" {
		t.Error("Version should not be empty")
	}
	if llmc.MinGoVersion == "" {
		t.Error("MinGoVersion should not be empty")
	}
}
