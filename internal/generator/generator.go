package generator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/LiboWorks/llm-compiler/internal/workflow"
)

// sanitizeIdentifier converts an arbitrary string to a valid Go identifier.
// Replaces invalid characters (-, /, ., space) with underscores and prefixes
// with '_' if the name starts with a digit.
func sanitizeIdentifier(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, " ", "_")
	if len(s) > 0 && unicode.IsDigit(rune(s[0])) {
		s = "_" + s
	}
	return s
}

// prefixedWorkflowName returns "a_name" where a is the 1-indexed workflow order.
func prefixedWorkflowName(wfIdx int, name string) string {
	return fmt.Sprintf("%d_%s", wfIdx+1, name)
}

// prefixedStepKey returns "wfKey.a_b/c_stepName" where:
//   - wfKey = prefixed workflow name
//   - a = 1-indexed workflow order
//   - b = 1-indexed step order within workflow
//   - c = total steps in the workflow
func prefixedStepKey(wfKey string, wfIdx int, stepIdx int, totalSteps int, stepName string) string {
	return fmt.Sprintf("%s.%d_%d/%d_%s", wfKey, wfIdx+1, stepIdx+1, totalSteps, stepName)
}

// GenerateOptions configures code generation.
type GenerateOptions struct {
	// OutputName is used for the JSON output filename (e.g., "example" -> "example_run.json")
	// If empty, defaults to "contexts_and_signals"
	OutputName string
}

// Generate builds a single Go program that runs one or more workflows in
// parallel. Workflows may coordinate via step-level `wait_for` values that
// reference `workflowName.stepName` keys.
func Generate(wfs []workflow.Workflow, opts *GenerateOptions) (string, error) {
	if opts == nil {
		opts = &GenerateOptions{}
	}
	jsonOutputName := "contexts_and_signals.json"
	if opts.OutputName != "" {
		jsonOutputName = opts.OutputName + "_run.json"
	}

	var sb strings.Builder

	// Build a mapping from original "workflow.step" keys to prefixed keys
	// so that wait_for references can be resolved.
	stepKeyMap := make(map[string]string)
	for wfIdx, wf := range wfs {
		wfKey := prefixedWorkflowName(wfIdx, wf.Name)
		totalSteps := len(wf.Steps)
		for stepIdx, step := range wf.Steps {
			originalKey := wf.Name + "." + step.Name
			prefixedKey := prefixedStepKey(wfKey, wfIdx, stepIdx, totalSteps, step.Name)
			stepKeyMap[originalKey] = prefixedKey
		}
	}

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
	"os"
`)
	if needLog {
		sb.WriteString(`	"log"
	"time"
`)
	}
	sb.WriteString(`	"sync"
	"encoding/json"
	"path/filepath"

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
	// signalValues stores the last sent value for each signal key for JSON dump
	// (channels may be consumed by wait_for before dump runs)
	signalValues := make(map[string]signalMsg)
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
	// send stores the value and sends to channel
	send := func(k string, msg signalMsg) {
		signalsMu.Lock()
		signalValues[k] = msg
		signalsMu.Unlock()
		select { case mk(k) <- msg: default: }
	}

	// contexts collects the final ctx.Vars for each workflow so we can
	// persist them after all workflows complete for debugging.
	contexts := make(map[string]map[string]string)
	var contextsMu sync.Mutex

	var wg sync.WaitGroup

	// Set up output capture (platform-aware)
	capture := runtime.NewOutputCapture()
	savedStdout, savedStderr, err := capture.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to set up output capture: %v\n", err)
		savedStdout = os.Stdout
		savedStderr = os.Stderr
	}
	defer capture.Stop()
	// Use savedStdout/savedStderr to avoid unused variable warnings
	_ = savedStdout
	_ = savedStderr
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
	// Track local_llm runtimes that need to be closed
	// We need to detect if any workflow uses local_llm to set up tracking
	needLocalLlama := false
	for _, wf := range wfs {
		for _, step := range wf.Steps {
			if step.Type == "local_llm" {
				needLocalLlama = true
				break
			}
		}
		if needLocalLlama {
			break
		}
	}
	if needLocalLlama {
		sb.WriteString("    // Track local_llm runtimes for cleanup\n")
		sb.WriteString("    var localLlamasMu sync.Mutex\n")
		sb.WriteString("    var localLlamas []*runtime.LocalLlamaRuntime\n")
	}

	sb.WriteString("\n")

	// Launch each workflow in its own goroutine
	for wfIdx, wf := range wfs {
		totalSteps := len(wf.Steps)
		wfKey := prefixedWorkflowName(wfIdx, wf.Name)
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
			sb.WriteString("        localLlamasMu.Lock()\n")
			sb.WriteString("        localLlamas = append(localLlamas, localLlama)\n")
			sb.WriteString("        localLlamasMu.Unlock()\n")
			sb.WriteString("\n")
		}

		for stepIdx, step := range wf.Steps {
			stepKey := prefixedStepKey(wfKey, wfIdx, stepIdx, totalSteps, step.Name)
			sb.WriteString(fmt.Sprintf("        // Step: %s\n", step.Name))

			// Wait-for handling
			if step.WaitFor != "" {
				// Resolve the wait_for reference to its prefixed key
				waitForKey := step.WaitFor
				if mapped, ok := stepKeyMap[step.WaitFor]; ok {
					waitForKey = mapped
				}
				keyQ := strconv.Quote(waitForKey)
				// Store the received value with the original wait_for key so the user
				// can access it via {{producer.final_output}} (the key they wrote in YAML)
				originalKeyQ := strconv.Quote(step.WaitFor)
				if step.WaitTimeout > 0 {
					sb.WriteString("        select {\n")
					sb.WriteString("            case msg := <-mk(" + keyQ + "):\n")
					sb.WriteString("                if msg.Err != \"\" {\n")
					sb.WriteString("                    log.Fatalf(\"producer %s failed: %s\", " + keyQ + ", msg.Err)\n")
					sb.WriteString("                }\n")
					sb.WriteString("                ctx.Set(" + originalKeyQ + ", msg.Val)\n")
					sb.WriteString("            case <-time.After(" + strconv.Itoa(step.WaitTimeout) + " * time.Second):\n")
					sb.WriteString("                log.Fatalf(\"wait_for timed out waiting for " + waitForKey + "\")\n")
					sb.WriteString("        }\n")
				} else {
					sb.WriteString("        msg := <-mk(" + keyQ + ")\n")
					sb.WriteString("        if msg.Err != \"\" {\n")
					sb.WriteString("            log.Fatalf(\"producer %s failed: %s\", " + keyQ + ", msg.Err)\n")
					sb.WriteString("        }\n")
					sb.WriteString("        ctx.Set(" + originalKeyQ + ", msg.Val)\n")
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
					sb.WriteString(fmt.Sprintf("                send(%q, signalMsg{Err: err.Error()})\n", stepKey))
					sb.WriteString("                return\n")
					sb.WriteString("            }\n")
					sb.WriteString(fmt.Sprintf("            ctx.Set(%q, out)\n", step.Output))
					// send to signals
					sb.WriteString(fmt.Sprintf("            send(%q, signalMsg{Val: out})\n", stepKey))
				} else {
					sb.WriteString("            out, err = shell.Run(cmd)\n")
					sb.WriteString("            if err != nil {\n")
					sb.WriteString(fmt.Sprintf("                send(%q, signalMsg{Err: err.Error()})\n", stepKey))
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
				varName := sanitizeIdentifier(fmt.Sprintf("prompt_%s_%s", wf.Name, step.Name))
				sb.WriteString(fmt.Sprintf("            %s := `%s`\n", varName, step.Prompt))
				rendered := varName + "_rendered"
				sb.WriteString(fmt.Sprintf("            %s, _ := runtime.RenderTemplate(%s, ctx.Vars)\n", rendered, varName))
				qModel := strconv.Quote(step.Model)
				sb.WriteString(fmt.Sprintf("            result, err = %s.Generate(%s, %s, maxTokens)\n", runtimeVar, rendered, qModel))
				sb.WriteString("            if err != nil {\n")
				sb.WriteString(fmt.Sprintf("                send(%q, signalMsg{Err: err.Error()})\n", stepKey))
				sb.WriteString("                return\n")
				sb.WriteString("            }\n")
				if step.Output != "" {
					sb.WriteString(fmt.Sprintf("            out = runtime.SanitizeForShell(result)\n            ctx.Set(%q, out)\n", step.Output))
					sb.WriteString(fmt.Sprintf("            send(%q, signalMsg{Val: out})\n", stepKey))
				}
			}

			if step.If != "" {
				sb.WriteString("        }\n")
			}

			sb.WriteString("\n")
		}

		sb.WriteString(fmt.Sprintf("            contextsMu.Lock()\n            contexts[%q] = ctx.Vars\n            contextsMu.Unlock()\n", wfKey))
		sb.WriteString("    }()\n\n")
	}

	sb.WriteString("    wg.Wait()\n")
	// Close local_llm runtimes to shut down worker subprocesses
	if needLocalLlama {
		sb.WriteString("    // Close local_llm runtimes (shuts down worker subprocesses)\n")
		sb.WriteString("    for _, ll := range localLlamas {\n")
		sb.WriteString("        ll.Close()\n")
		sb.WriteString("    }\n")
	}
	sb.WriteString("    // Dump contexts and channel values as JSON for debugging\n")
	sb.WriteString("    dump := map[string]interface{}{}\n")
	sb.WriteString("    dump[\"contexts\"] = contexts\n")
	sb.WriteString("    chans := make(map[string]map[string]interface{})\n")
	sb.WriteString("    signalsMu.Lock()\n")
	sb.WriteString("    for k, msg := range signalValues {\n")
	sb.WriteString("        m := map[string]interface{}{}\n")
	sb.WriteString("        m[\"val\"] = msg.Val\n")
	sb.WriteString("        if msg.Err == \"\" { m[\"err\"] = nil } else { m[\"err\"] = msg.Err }\n")
	sb.WriteString("        chans[k] = m\n")
	sb.WriteString("    }\n")
	sb.WriteString("    signalsMu.Unlock()\n")
	sb.WriteString("    dump[\"channels\"] = chans\n")
	sb.WriteString("    b, _ := json.MarshalIndent(dump, \"\", \"  \")\n")
	sb.WriteString("    exe, _ := os.Executable()\n")
	sb.WriteString("    exeDir := filepath.Dir(exe)\n")
	sb.WriteString(fmt.Sprintf("    outPath := filepath.Join(exeDir, %q)\n", jsonOutputName))
	sb.WriteString("    _ = os.WriteFile(outPath, b, 0644)\n")
	sb.WriteString("    fmt.Println(\"\\n✅ Workflows completed\")\n")
	sb.WriteString("}\n")

	return sb.String(), nil
}
