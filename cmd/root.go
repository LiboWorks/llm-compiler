package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "llm-compiler",
	Short: "Compile YAML workflows into native Go binaries with LLM support",
	Long: `llm-compiler transforms declarative YAML workflow definitions into
standalone Go binaries that can orchestrate shell commands and LLM inference.

Features:
  - Define workflows in YAML with shell and LLM steps
  - Compile to native binaries with embedded llama.cpp
  - Run multiple workflows in parallel with cross-workflow communication
  - Support for local GGUF models via llama.cpp`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}
