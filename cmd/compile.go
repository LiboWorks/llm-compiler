package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/libochen/llm-compiler/internal/generator"
	"github.com/libochen/llm-compiler/internal/workflow"
	"github.com/spf13/cobra"
)

var output string

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile <workflow-file>",
	Short: "Compile a workflow file into a runnable agent pipeline",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workflowFile := args[0]
		fmt.Println("🔧 Starting compilation...")

		// Check file exists
		if _, err := os.Stat(workflowFile); os.IsNotExist(err) {
			fmt.Printf("❌ Workflow file not found: %s\n", workflowFile)
			os.Exit(1)
		}

		fmt.Printf("✅ Workflow file: %s\n", workflowFile)
		fmt.Printf("📦 Output target folder: %s\n", output)

		// Load workflow
		wf, err := workflow.LoadWorkflow(workflowFile)
		if err != nil {
			fmt.Printf("❌ Failed to parse workflow: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("📋 Workflow loaded: %s\n", wf.Name)
		fmt.Printf("🧩 Steps: %d\n", len(wf.Steps))

		// Validate workflow
		if err := wf.Validate(); err != nil {
			fmt.Printf("❌ Validation error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Workflow validated")

		// Ensure output folder exists
		if err := os.MkdirAll(output, 0755); err != nil {
			fmt.Printf("❌ Failed to create output folder: %v\n", err)
			os.Exit(1)
		}

		// Generate code
		program, err := generator.Generate(*wf)
		if err != nil {
			fmt.Printf("❌ Code generation failed: %v\n", err)
			os.Exit(1)
		}

		// Timestamped output file
		timestamp := time.Now().Format("20060102_150405")
		outputFile := filepath.Join(output, fmt.Sprintf("%s_%s.go", wf.Name, timestamp))

		if err := generator.SaveToFile(outputFile, program); err != nil {
			fmt.Printf("❌ Failed to save generated file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Code generated at %s\n", outputFile)

		// Build Go file
		if err := generator.BuildGoFile(outputFile); err != nil {
			fmt.Printf("❌ Build failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Build complete! Run with: %s\n", outputFile[:len(outputFile)-3])
	},
}

func init() {
	rootCmd.AddCommand(compileCmd)
	compileCmd.Flags().StringVarP(&output, "output", "o", "build/", "Output folder")
}
