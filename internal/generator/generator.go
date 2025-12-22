package generator

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	
	"github.com/LiboWorks/llm-compiler/internal/workflow"
)

// Generate builds a single Go program that runs one or more workflows in
// parallel. Workflows may coordinate via step-level `wait_for` values that
// reference `workflowName.stepName` keys.
func Generate(wfs []workflow.Workflow) (string, error) {
	var sb strings.Builder

	// Pre-scan to determine which imports are needed
	needLog := false
	for _, wf := range wfs {
		for _, step := range wf.Steps {
			if step.WaitFor != "" {
				needLog = true
				break
			}
		}
		if needLog {
			break
		}
	}

	// Header with conditional imports
	sb.WriteString(`package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"path/filepath"
`)
	if needLog {
		sb.WriteString(`	"log"
`)
	}
	sb.WriteString(`	"sync"
	"time"
	"encoding/json"

	"github.com/LiboWorks/llm-compiler/internal/config"
	"github.com/LiboWorks/llm-compiler/internal/runtime"
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
	// coordination channels for cross-workflow step outputs
	type signalMsg struct { Val string; Err string }
	signals := make(map[string]chan signalMsg)
	var signalsMu sync.Mutex
	mk := func(k string) chan signalMsg {
		signalsMu.Lock()
		ch, ok := signals[k]
		if !ok {
			ch = make(chan signalMsg, 1)
			signals[k] = ch
		}
		signalsMu.Unlock()
		return ch
	}

	// contexts collects the final ctx.Vars for each workflow so we can
	// persist them after all workflows complete for debugging.
	contexts := make(map[string]map[string]string)
	var contextsMu sync.Mutex

	var wg sync.WaitGroup

	// create files next to the executable: one for Go fmt output, one for native C output
	cfg := config.Get()
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	fmtOutPath := filepath.Join(exeDir, cfg.FmtOutputFile)
	llamaOutPath := filepath.Join(exeDir, cfg.LlamaOutputFile)
	// Truncate existing files on program start so this run starts fresh.
	fmtFile, _ := os.OpenFile(fmtOutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	llamaFile, _ := os.OpenFile(llamaOutPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	// Duplicate the fmt file descriptor to fd 3 so subprocess workers can
	// write their fmt/log messages directly to fd 3 (no extra pipes needed).
	_ = syscall.Dup2(int(fmtFile.Fd()), 3)
	// write a small run header to mark the run
	header := fmt.Sprintf("=== RUN START: %s pid=%d ===\n", time.Now().Format(time.RFC3339), os.Getpid())
	_, _ = fmtFile.WriteString(header)
	_ = fmtFile.Sync()
	_, _ = llamaFile.WriteString(header)
	_ = llamaFile.Sync()

	// save original terminal fds so we can continue to write to the terminal
	savedStdoutFd, _ := syscall.Dup(int(os.Stdout.Fd()))
	savedStderrFd, _ := syscall.Dup(int(os.Stderr.Fd()))
	savedStdout := os.NewFile(uintptr(savedStdoutFd), "saved_stdout")
	savedStderr := os.NewFile(uintptr(savedStderrFd), "saved_stderr")

	// --- Go-level output capture: replace os.Stdout/os.Stderr with pipe writers
	rGoOut, wGoOut, _ := os.Pipe()
	rGoErr, wGoErr, _ := os.Pipe()
	os.Stdout = wGoOut
	os.Stderr = wGoErr

	// copy Go-level output into (terminal + fmtFile)
	go func() {
		defer rGoOut.Close()
		io.Copy(io.MultiWriter(savedStdout, fmtFile), rGoOut)
	}()
	go func() {
		defer rGoErr.Close()
		io.Copy(io.MultiWriter(savedStderr, fmtFile), rGoErr)
	}()

	// --- Native-level output capture: redirect fd 1/2 to separate pipes
	rCOut, wCOut, _ := os.Pipe()
	rCErr, wCErr, _ := os.Pipe()
	_ = syscall.Dup2(int(wCOut.Fd()), 1)
	_ = syscall.Dup2(int(wCErr.Fd()), 2)

	// copy native-level output into (terminal + llamaFile)
	go func() {
		defer rCOut.Close()
		io.Copy(io.MultiWriter(savedStdout, llamaFile), rCOut)
	}()
	go func() {
		defer rCErr.Close()
		io.Copy(io.MultiWriter(savedStderr, llamaFile), rCErr)
	}()

	// restore and close on exit
	defer func() {
		// restore original fds
		_ = syscall.Dup2(savedStdoutFd, int(os.Stdout.Fd()))
		_ = syscall.Dup2(savedStderrFd, int(os.Stderr.Fd()))
		// close pipe writers (this will end the readers)
		wGoOut.Close()
		wGoErr.Close()
		wCOut.Close()
		wCErr.Close()
		// flush and close files
		fmtFile.Sync()
		fmtFile.Close()
		llamaFile.Sync()
		llamaFile.Close()
		savedStdout.Close()
		savedStderr.Close()
	}()
`)

	// Determine which runtimes are required by the workflows.
	// We separate `llm` and `local_llm` detection so we only emit a top-level
	// `llm` when real remote/managed LLM steps exist. `local_llm` is
	// instantiated per-workflow below to avoid sharing a non-thread-safe
	// llama.cpp-backed runtime.
	needShell := false
	needLLM := false
	for _, wf := range wfs {
		for _, step := range wf.Steps {
			if step.Type == "shell" || step.Command != "" {
				needShell = true
			}
			if step.Type == "llm" {
				needLLM = true
			}
			// If a prompt is present and it's not explicitly a local_llm step,
			// treat it as a regular llm usage.
			if step.Prompt != "" && step.Type != "local_llm" {
				needLLM = true
			}
		}
	}

	if needShell {
		sb.WriteString("    shell := runtime.NewShellRuntime()\n")
	}
	if needLLM {
		sb.WriteString("    llm := runtime.NewLLMRuntime()\n")
	}
	// NOTE: do not create a shared `localLlama` here. `local_llm` runtimes
	// (backed by llama.cpp) are not guaranteed to be goroutine-safe. We
	// instantiate per-workflow `localLlama` instances below inside each
	// workflow goroutine when a workflow actually needs it.

	sb.WriteString("\n")

	// Launch each workflow in its own goroutine
	for _, wf := range wfs {
		// detect per-workflow needs to avoid declaring unused variables
		hasShell := false
		hasLLM := false
		hasLocal := false
		for _, s := range wf.Steps {
			if s.Type == "shell" || s.Command != "" {
				hasShell = true
			}
			if s.Type == "llm" || s.Prompt != "" {
				hasLLM = true
			}
			if s.Type == "local_llm" {
				hasLLM = true
				hasLocal = true
			}
		}

		sb.WriteString(fmt.Sprintf("    // Workflow: %s\n", wf.Name))
		sb.WriteString("    wg.Add(1)\n")
		sb.WriteString("    go func() {\n")
		sb.WriteString("        defer wg.Done()\n")
		sb.WriteString("        ctx := NewContext()\n")
		if hasLLM {
			sb.WriteString("        var result string\n")
			sb.WriteString("        var maxTokens int\n")
		}
		if hasShell || hasLLM {
			sb.WriteString("        var out string\n")
			sb.WriteString("        var err error\n")
		}
		if hasShell {
			sb.WriteString("        var cmd string\n")
		}
		sb.WriteString("\n")
		if hasLocal {
			sb.WriteString("        localLlama := runtime.NewLocalLlamaRuntime()\n")
			sb.WriteString("\n")
		}

		for _, step := range wf.Steps {
			sb.WriteString(fmt.Sprintf("        // Step: %s\n", step.Name))

			// Wait-for handling
			if step.WaitFor != "" {
				keyQ := strconv.Quote(step.WaitFor)
				if step.WaitTimeout > 0 {
					sb.WriteString("        select {\n")
					sb.WriteString("            case msg := <-mk(" + keyQ + "):\n")
					sb.WriteString("                if msg.Err != \"\" {\n")
					sb.WriteString("                    log.Fatalf(\"producer %s failed: %s\", " + keyQ + ", msg.Err)\n")
					sb.WriteString("                }\n")
					sb.WriteString("                ctx.Set(" + keyQ + ", msg.Val)\n")
					sb.WriteString("            case <-time.After(" + strconv.Itoa(step.WaitTimeout) + " * time.Second):\n")
					sb.WriteString("                log.Fatalf(\"wait_for timed out waiting for " + step.WaitFor + "\")\n")
					sb.WriteString("        }\n")
				} else {
					sb.WriteString("        msg := <-mk(" + keyQ + ")\n")
					sb.WriteString("        if msg.Err != \"\" {\n")
					sb.WriteString("            log.Fatalf(\"producer %s failed: %s\", " + keyQ + ", msg.Err)\n")
					sb.WriteString("        }\n")
					sb.WriteString("        ctx.Set(" + keyQ + ", msg.Val)\n")
				}
			}

			// Conditional execution
			if step.If != "" {
				sb.WriteString(fmt.Sprintf("        if runtime.EvalCondition(ctx, %q) {\n", step.If))
			}

			// Shell steps
			if step.Type == "shell" || step.Command != "" {
				sb.WriteString(fmt.Sprintf("            cmd, _ = runtime.RenderTemplate(%q, ctx.Vars)\n", step.Command))
				if step.Output != "" {
					sb.WriteString("            out, err = shell.Run(cmd)\n")
					sb.WriteString("            if err != nil {\n")
					sb.WriteString(fmt.Sprintf("                select { case mk(%q) <- signalMsg{Err: err.Error()}: default: }\n", wf.Name+"."+step.Name))
					sb.WriteString("                return\n")
					sb.WriteString("            }\n")
					sb.WriteString(fmt.Sprintf("            ctx.Set(%q, out)\n", step.Output))
					// send to signals
					sb.WriteString(fmt.Sprintf("            select { case mk(%q) <- signalMsg{Val: out}: default: }\n", wf.Name+"."+step.Name))
				} else {
					sb.WriteString("            out, err = shell.Run(cmd)\n")
					sb.WriteString("            if err != nil {\n")
					sb.WriteString(fmt.Sprintf("                select { case mk(%q) <- signalMsg{Err: err.Error()}: default: }\n", wf.Name+"."+step.Name))
					sb.WriteString("                return\n")
					sb.WriteString("            }\n")
					sb.WriteString("            if len(out) > 0 {\n")
					sb.WriteString("                fmt.Print(out)\n")
					sb.WriteString("            }\n")
				}
			}

			// LLM steps
			if step.Type == "llm" || step.Type == "local_llm" || step.Prompt != "" {
				runtimeVar := "llm"
				if step.Type == "local_llm" {
					runtimeVar = "localLlama"
				}
				if step.MaxTokens != 0 {
					sb.WriteString(fmt.Sprintf("            maxTokens = %d\n", step.MaxTokens))
				} else {
					sb.WriteString("            maxTokens = 256\n")
				}
				// Emit the prompt as a raw backtick string literal in the generated
				// program to preserve multi-line prompts safely (prompts should not
				// contain backticks). Use a sanitized variable name per step and
				// render it with `runtime.RenderTemplate` at runtime using the
				// workflow `ctx.Vars` so earlier step outputs are substituted.
				varName := fmt.Sprintf("prompt_%s_%s", wf.Name, step.Name)
				// sanitize common characters
				varName = strings.ReplaceAll(varName, "-", "_")
				varName = strings.ReplaceAll(varName, ".", "_")
				varName = strings.ReplaceAll(varName, " ", "_")
				sb.WriteString(fmt.Sprintf("            %s := `%s`\n", varName, step.Prompt))
				rendered := varName + "_rendered"
				sb.WriteString(fmt.Sprintf("            %s, _ := runtime.RenderTemplate(%s, ctx.Vars)\n", rendered, varName))
				qModel := strconv.Quote(step.Model)
				sb.WriteString(fmt.Sprintf("            result, err = %s.Generate(%s, %s, maxTokens)\n", runtimeVar, rendered, qModel))
				sb.WriteString("            if err != nil {\n")
				sb.WriteString(fmt.Sprintf("                select { case mk(%q) <- signalMsg{Err: err.Error()}: default: }\n", wf.Name+"."+step.Name))
				sb.WriteString("                return\n")
				sb.WriteString("            }\n")
				if step.Output != "" {
					sb.WriteString(fmt.Sprintf("            out = runtime.SanitizeForShell(result)\n            ctx.Set(%q, out)\n", step.Output))
					sb.WriteString(fmt.Sprintf("            select { case mk(%q) <- signalMsg{Val: out}: default: }\n", wf.Name+"."+step.Name))
				}
			}

			if step.If != "" {
				sb.WriteString("        }\n")
			}

			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("            contextsMu.Lock()\n            contexts[%q] = ctx.Vars\n            contextsMu.Unlock()\n", wf.Name))
		sb.WriteString("    }()\n\n")
	}

	sb.WriteString("    wg.Wait()\n")
	sb.WriteString("    // Dump contexts and channel values as JSON for debugging\n")
	sb.WriteString("    dump := map[string]interface{}{}\n")
	sb.WriteString("    dump[\"contexts\"] = contexts\n")
	sb.WriteString("    chans := make(map[string]map[string]string)\n")
	sb.WriteString("    signalsMu.Lock()\n")
	sb.WriteString("    for k, ch := range signals {\n")
	sb.WriteString("        m := map[string]string{}\n")
	sb.WriteString("        select {\n")
	sb.WriteString("        case msg := <-ch:\n")
	sb.WriteString("            m[\"val\"] = msg.Val\n")
	sb.WriteString("            m[\"err\"] = msg.Err\n")
	sb.WriteString("        default:\n")
	sb.WriteString("            m[\"val\"] = \"\"\n")
	sb.WriteString("            m[\"err\"] = \"\"\n")
	sb.WriteString("        }\n")
	sb.WriteString("        chans[k] = m\n")
	sb.WriteString("    }\n")
	sb.WriteString("    signalsMu.Unlock()\n")
	sb.WriteString("    dump[\"channels\"] = chans\n")
	sb.WriteString("    b, _ := json.MarshalIndent(dump, \"\", \"  \")\n")
	sb.WriteString("    exe, _ = os.Executable()\n")
	sb.WriteString("    exeDir = filepath.Dir(exe)\n")
	sb.WriteString("    outPath := filepath.Join(exeDir, \"contexts_and_signals.json\")\n")
	sb.WriteString("    _ = os.WriteFile(outPath, b, 0644)\n")
	sb.WriteString("    fmt.Println(\"\\n✅ Workflows completed\")\n")
	sb.WriteString("}\n")

	return sb.String(), nil
}

func SaveToFile(output string, program string) error {
	return os.WriteFile(output, []byte(program), 0644)
}
