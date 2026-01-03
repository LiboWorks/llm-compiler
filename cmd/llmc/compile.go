package main

import (
	"fmt"

	"github.com/LiboWorks/llm-compiler/pkg/llmc"
	"github.com/spf13/cobra"
)

var (
	inputFile  string
	outputDir  string
	keepSource bool
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile",
	Short: "Compile a workflow file into a runnable agent pipeline",
	Long: `Compile transforms a YAML workflow definition into a standalone
Go binary with embedded LLM inference capabilities.

Examples:
  llmc compile -i workflow.yaml -o ./build
  llmc compile -i example.yaml
  llmc compile --input multi-workflow.yaml --output ./dist
  llmc compile -i workflow.yaml --keep-source`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Support both -i flag and positional argument for backwards compatibility
		workflowFile := inputFile
		if workflowFile == "" && len(args) > 0 {
			workflowFile = args[0]
		}
		if workflowFile == "" {
			return fmt.Errorf("workflow file required: use -i <file> or provide as argument")
		}

		fmt.Println("🔧 Starting compilation...")

		result, err := llmc.CompileFile(workflowFile, &llmc.CompileOptions{
			OutputDir:  outputDir,
			KeepSource: keepSource,
		})
		if err != nil {
			return err
		}

		// Print workflow info
		for _, wf := range result.Workflows {
			fmt.Printf("📋 Workflow loaded: %s\n", wf.Name)
			fmt.Printf("🧩 Steps: %d\n", len(wf.Steps))
			fmt.Println("✅ Workflow validated")
		}

		if result.SourcePath != "" {
			fmt.Printf("📄 Source saved at %s\n", result.SourcePath)
		}
		fmt.Printf("✅ Build complete! Run with: %s\n", result.BinaryPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input workflow YAML file (required)")
	compileCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory for generated binary")
	compileCmd.Flags().BoolVar(&keepSource, "keep-source", false, "Save generated .go source file alongside binary")
}
