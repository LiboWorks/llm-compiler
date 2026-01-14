package main

import (
	"fmt"
	"os"
	"log"
	"time"
	"sync"
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
    shell := runtime.NewShellRuntime()
    // Track local_llm runtimes for cleanup
    var localLlamasMu sync.Mutex
    var localLlamas []*runtime.LocalLlamaRuntime

    // Workflow: producer
    wg.Add(1)
    go func() {
        defer wg.Done()
        ctx := NewContext()
        var result string
        var maxTokens int
        var out string
        var err error
        var cmd string

        localLlama := runtime.NewLocalLlamaRuntime()
        localLlamasMu.Lock()
        localLlamas = append(localLlamas, localLlama)
        localLlamasMu.Unlock()

        // Step: generate_data
            cmd, _ = runtime.RenderTemplate("echo \"Hello from Producer\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("1_producer.1_1/4_generate_data", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("message", out)
            send("1_producer.1_1/4_generate_data", signalMsg{Val: out})

        // Step: check_status
            cmd, _ = runtime.RenderTemplate("echo \"ready\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("1_producer.1_2/4_check_status", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("status", out)
            send("1_producer.1_2/4_check_status", signalMsg{Val: out})

        // Step: final_output
            cmd, _ = runtime.RenderTemplate("echo \"Producer says: {{message}} | Status: {{status}}\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("1_producer.1_3/4_final_output", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("producer_result", out)
            send("1_producer.1_3/4_final_output", signalMsg{Val: out})

        // Step: prod_summarize
            maxTokens = 32
            prompt_producer_prod_summarize := `Summarize the following producer result in one short line:
{{producer_result}}
`
            prompt_producer_prod_summarize_rendered, _ := runtime.RenderTemplate(prompt_producer_prod_summarize, ctx.Vars)
            result, err = localLlama.Generate(prompt_producer_prod_summarize_rendered, "/Users/libochen/Downloads/meta-llama-3-8b-instruct.Q4_K_M.gguf", maxTokens)
            if err != nil {
                send("1_producer.1_4/4_prod_summarize", signalMsg{Err: err.Error()})
                return
            }
            out = runtime.SanitizeForShell(result)
            ctx.Set("producer_summary", out)
            send("1_producer.1_4/4_prod_summarize", signalMsg{Val: out})

            contextsMu.Lock()
            contexts["1_producer"] = ctx.Vars
            contextsMu.Unlock()
    }()

    // Workflow: consumer
    wg.Add(1)
    go func() {
        defer wg.Done()
        ctx := NewContext()
        var result string
        var maxTokens int
        var out string
        var err error
        var cmd string

        localLlama := runtime.NewLocalLlamaRuntime()
        localLlamasMu.Lock()
        localLlamas = append(localLlamas, localLlama)
        localLlamasMu.Unlock()

        // Step: wait_for_producer
        select {
            case msg := <-mk("1_producer.1_3/4_final_output"):
                if msg.Err != "" {
                    log.Fatalf("producer %s failed: %s", "1_producer.1_3/4_final_output", msg.Err)
                }
                ctx.Set("producer.final_output", msg.Val)
            case <-time.After(10 * time.Second):
                log.Fatalf("wait_for timed out waiting for 1_producer.1_3/4_final_output")
        }
            cmd, _ = runtime.RenderTemplate("echo \"Consumer received: {{producer.final_output}}\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("2_consumer.2_1/3_wait_for_producer", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("received_data", out)
            send("2_consumer.2_1/3_wait_for_producer", signalMsg{Val: out})

        // Step: process_data
            cmd, _ = runtime.RenderTemplate("echo \"Processed: {{received_data}}\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("2_consumer.2_2/3_process_data", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("consumer_result", out)
            send("2_consumer.2_2/3_process_data", signalMsg{Val: out})

        // Step: consumer_insight
            maxTokens = 32
            prompt_consumer_consumer_insight := `Provide a one-line insight based on: {{consumer_result}}`
            prompt_consumer_consumer_insight_rendered, _ := runtime.RenderTemplate(prompt_consumer_consumer_insight, ctx.Vars)
            result, err = localLlama.Generate(prompt_consumer_consumer_insight_rendered, "/Users/libochen/Downloads/meta-llama-3-8b-instruct.Q4_K_M.gguf", maxTokens)
            if err != nil {
                send("2_consumer.2_3/3_consumer_insight", signalMsg{Err: err.Error()})
                return
            }
            out = runtime.SanitizeForShell(result)
            ctx.Set("consumer_insight", out)
            send("2_consumer.2_3/3_consumer_insight", signalMsg{Val: out})

            contextsMu.Lock()
            contexts["2_consumer"] = ctx.Vars
            contextsMu.Unlock()
    }()

    // Workflow: conditional
    wg.Add(1)
    go func() {
        defer wg.Done()
        ctx := NewContext()
        var result string
        var maxTokens int
        var out string
        var err error
        var cmd string

        localLlama := runtime.NewLocalLlamaRuntime()
        localLlamasMu.Lock()
        localLlamas = append(localLlamas, localLlama)
        localLlamasMu.Unlock()

        // Step: set_mode
            cmd, _ = runtime.RenderTemplate("echo \"production\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("3_conditional.3_1/5_set_mode", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("mode", out)
            send("3_conditional.3_1/5_set_mode", signalMsg{Val: out})

        // Step: production_action
        if runtime.EvalCondition(ctx, "{{mode}} == 'production'") {
            cmd, _ = runtime.RenderTemplate("echo \"Running in PRODUCTION mode\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("3_conditional.3_2/5_production_action", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("action_result", out)
            send("3_conditional.3_2/5_production_action", signalMsg{Val: out})
        }

        // Step: debug_action
        if runtime.EvalCondition(ctx, "{{mode}} == 'debug'") {
            cmd, _ = runtime.RenderTemplate("echo \"Running in DEBUG mode\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("3_conditional.3_3/5_debug_action", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("debug_result", out)
            send("3_conditional.3_3/5_debug_action", signalMsg{Val: out})
        }

        // Step: summary
            cmd, _ = runtime.RenderTemplate("echo \"Mode={{mode}} Action={{action_result}}\"", ctx.Vars)
            out, err = shell.Run(cmd)
            if err != nil {
                send("3_conditional.3_4/5_summary", signalMsg{Err: err.Error()})
                return
            }
            ctx.Set("summary", out)
            send("3_conditional.3_4/5_summary", signalMsg{Val: out})

        // Step: generate_note
            maxTokens = 32
            prompt_conditional_generate_note := `Write a very short note about the run:
Mode={{mode}}; Action={{action_result}}
`
            prompt_conditional_generate_note_rendered, _ := runtime.RenderTemplate(prompt_conditional_generate_note, ctx.Vars)
            result, err = localLlama.Generate(prompt_conditional_generate_note_rendered, "/Users/libochen/Downloads/meta-llama-3-8b-instruct.Q4_K_M.gguf", maxTokens)
            if err != nil {
                send("3_conditional.3_5/5_generate_note", signalMsg{Err: err.Error()})
                return
            }
            out = runtime.SanitizeForShell(result)
            ctx.Set("note", out)
            send("3_conditional.3_5/5_generate_note", signalMsg{Val: out})

            contextsMu.Lock()
            contexts["3_conditional"] = ctx.Vars
            contextsMu.Unlock()
    }()

    wg.Wait()
    // Close local_llm runtimes (shuts down worker subprocesses)
    for _, ll := range localLlamas {
        ll.Close()
    }
    // Dump contexts and channel values as JSON for debugging
    dump := map[string]interface{}{}
    dump["contexts"] = contexts
    chans := make(map[string]map[string]interface{})
    signalsMu.Lock()
    for k, msg := range signalValues {
        m := map[string]interface{}{}
        m["val"] = msg.Val
        if msg.Err == "" { m["err"] = nil } else { m["err"] = msg.Err }
        chans[k] = m
    }
    signalsMu.Unlock()
    dump["channels"] = chans
    b, _ := json.MarshalIndent(dump, "", "  ")
    exe, _ := os.Executable()
    exeDir := filepath.Dir(exe)
    outPath := filepath.Join(exeDir, "example_run.json")
    _ = os.WriteFile(outPath, b, 0644)
    fmt.Println("\n✅ Workflows completed")
}
