package tests

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Detect accidental imports/references to the private pro module in the public repo.
func TestNoPrivateImports(t *testing.T) {
	var found []string
	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// skip vendor and .git and build
			if d.Name() == "vendor" || d.Name() == ".git" || d.Name() == "build" {
				return fs.SkipDir
			}
			return nil
		}
		// skip the test file itself which contains an example reference
		if strings.HasSuffix(path, "no_private_imports_test.go") {
			return nil
		}

		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "go.mod") {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(b), "github.com/libochen/llm-compiler-pro") {
				found = append(found, path)
			}
		}
		return nil
	})
	if len(found) > 0 {
		t.Fatalf("found references to private module in public repo: %v", found)
	}
}

// fpFS is a tiny wrapper to use os.ReadFile via fs.ReadFile semantics.
// no additional helper required
