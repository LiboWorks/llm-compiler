package generator

import (
	"fmt"
	"os/exec"
)

func BuildGoFile(sourcePath string) error {
	outputPath := sourcePath[:len(sourcePath)-3] // remove .go
	cmd := exec.Command("go", "build", "-o", outputPath, sourcePath)

	fmt.Printf("🔨 Building %s...\n", outputPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build error: %v\n%s", err, string(out))
	}
	return nil
}
