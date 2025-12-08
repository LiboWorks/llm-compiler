//go:build pro

package main

import (
	"github.com/LiboWorks/llm-compiler/cmd"
	_ "github.com/libochen/llm-compiler-pro/register"
)

func main() {
	cmd.Execute()
}
