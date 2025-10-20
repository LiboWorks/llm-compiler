package generator

import (
	"fmt"
	"os"
	"strings"

	"github.com/libochen/llm-compiler/internal/workflow"
)

func Generate(wf workflow.Workflow) (string, error) {
	var sb strings.Builder

	// Header
	sb.WriteString(`package main

import (
	"fmt"
	"log"

	"github.com/libochen/llm-compiler/internal/runtime"
)

type Context struct {
	Vars map[string]string
}

func NewContext() *Context {
	return &Context{Vars: make(map[string]string)}
}

func (c *Context) Set(key, value string) {
	c.Vars[key] = value
}

func (c *Context) Get(key string) string {
	return c.Vars[key]
}

func main() {
	ctx := NewContext()
	shell := runtime.NewShellRuntime()
	llm := runtime.NewLLMRuntime()

`)

	// Steps
	for _, step := range wf.Steps {
		sb.WriteString(fmt.Sprintf("    // Step: %s\n", step.Name))

		// Wrap condition if exists
		if step.If != "" {
			sb.WriteString(fmt.Sprintf(`    if runtime.EvalCondition(ctx, %q) {
`, step.If))
		}

		// Shell step
		if step.Command != "" {
			sb.WriteString(fmt.Sprintf(`        cmd, _ := runtime.RenderTemplate(%q, ctx.Vars)
`, step.Command))

			if step.Output != "" {
				sb.WriteString(fmt.Sprintf(`        out, err := shell.Run(cmd)
        if err != nil {
            log.Fatalf("shell step '%s' failed: %%v", err)
        }
        ctx.Set("%s", out)
`, step.Name, step.Output))
			} else {
				sb.WriteString(fmt.Sprintf(`        _, err = shell.Run(cmd)
        if err != nil {
            log.Fatalf("shell step '%s' failed: %%v", err)
        }
`, step.Name))
			}
		}

		// LLM step
		if step.Prompt != "" {
			sb.WriteString(fmt.Sprintf(`        result, err := llm.Generate(%q, "%s")
        if err != nil {
            log.Fatalf("llm step '%s' failed: %%v", err)
        }
`, step.Prompt, step.Model, step.Name))

			if step.Output != "" {
				sb.WriteString(fmt.Sprintf(`        ctx.Set(%q, result)
`, step.Output))
			}
		}

		// Close condition block if used
		if step.If != "" {
			sb.WriteString("    }\n\n")
		} else {
			sb.WriteString("\n")
		}
	}

	// Footer
	sb.WriteString(`    fmt.Println("\n✅ Workflow completed")
}
`)

	return sb.String(), nil
}

func SaveToFile(output string, program string) error {
	return os.WriteFile(output, []byte(program), 0644)
}
