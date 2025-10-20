package generator

import (
	"fmt"
	"os/exec"
)

func BuildGoFile(sourcePath string) error {
	outputPath := sourcePath[:len(sourcePath)-3] // remove .go
	cmd := exec.Command("go", "build", "-o", outputPath, sourcePath)
	cmd.Stdout = nil
	cmd.Stderr = nil

	fmt.Printf("🔨 Building %s...\n", outputPath)
	return cmd.Run()
}
