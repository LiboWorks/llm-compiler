package main

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "llmc",
	Short: "Compile YAML workflows into native Go binaries with LLM support",
	Long: `llmc transforms declarative YAML workflow definitions into
standalone Go binaries that can orchestrate shell commands and LLM inference.

Features:
  - Define workflows in YAML with shell and LLM steps
  - Compile to native binaries with embedded llama.cpp
  - Run multiple workflows in parallel with cross-workflow communication
  - Support for local GGUF models via llama.cpp

Examples:
  llmc compile -i workflow.yaml -o ./build
  llmc compile -i example.yaml`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
