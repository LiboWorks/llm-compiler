package main

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

    // Step: ask
        result, err := llm.Generate("Give me a random programming language", "gpt-4")
        if err != nil {
            log.Fatalf("llm step 'ask' failed: %v", err)
        }
        ctx.Set("lang", result)

    // Step: say
        cmd, _ := runtime.RenderTemplate("echo \"You chose {{lang}}\"", ctx.Vars)
        _, err = shell.Run(cmd)
        if err != nil {
            log.Fatalf("shell step 'say' failed: %v", err)
        }

    fmt.Println("\n✅ Workflow completed")
}
