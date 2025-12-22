// Package runtime provides runtime helpers for generated llm-compiler programs.
// This file contains the app context and initialization helpers.
package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/LiboWorks/llm-compiler/internal/config"
)

// App represents the application runtime with all required state
type App struct {
	// Configuration
	Config *config.Config

	// Output files
	FmtFile   *os.File
	LlamaFile *os.File
	ExeDir    string

	// Saved terminal descriptors for restoration
	SavedStdout *os.File
	SavedStderr *os.File

	// Signal coordination for cross-workflow step outputs
	Signals   map[string]chan SignalMsg
	SignalsMu sync.Mutex

	// Contexts for each workflow
	Contexts   map[string]map[string]string
	ContextsMu sync.Mutex

	// WaitGroup for workflow coordination
	WG sync.WaitGroup

	// Runtimes (lazily initialized)
	shell      *ShellRuntime
	llm        *LLMRuntime
	localLlama *LocalLlamaRuntime
	runtimeMu  sync.Mutex
}

// SignalMsg is used for cross-workflow step coordination
type SignalMsg struct {
	Val string
	Err string
}

// NewApp creates a new application runtime
func NewApp() *App {
	cfg := config.Get()
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	return &App{
		Config:    cfg,
		ExeDir:    exeDir,
		Signals:   make(map[string]chan SignalMsg),
		Contexts:  make(map[string]map[string]string),
	}
}

// MakeSignal returns or creates a signal channel for the given key
func (a *App) MakeSignal(key string) chan SignalMsg {
	a.SignalsMu.Lock()
	defer a.SignalsMu.Unlock()

	ch, ok := a.Signals[key]
	if !ok {
		ch = make(chan SignalMsg, 1)
		a.Signals[key] = ch
	}
	return ch
}

// SendSignal sends a value to a signal channel (non-blocking)
func (a *App) SendSignal(key, val string) {
	ch := a.MakeSignal(key)
	select {
	case ch <- SignalMsg{Val: val}:
	default:
	}
}

// SendSignalError sends an error to a signal channel (non-blocking)
func (a *App) SendSignalError(key, err string) {
	ch := a.MakeSignal(key)
	select {
	case ch <- SignalMsg{Err: err}:
	default:
	}
}

// WaitForSignal waits for a signal with optional timeout
func (a *App) WaitForSignal(key string, timeout int) (SignalMsg, error) {
	ch := a.MakeSignal(key)

	if timeout > 0 {
		select {
		case msg := <-ch:
			return msg, nil
		case <-time.After(time.Duration(timeout) * time.Second):
			return SignalMsg{}, fmt.Errorf("timeout waiting for signal: %s", key)
		}
	}

	msg := <-ch
	return msg, nil
}

// SaveContext saves a workflow's context
func (a *App) SaveContext(workflowName string, vars map[string]string) {
	a.ContextsMu.Lock()
	defer a.ContextsMu.Unlock()
	a.Contexts[workflowName] = vars
}

// DumpContextsAndSignals dumps all contexts and signal values to a JSON file
func (a *App) DumpContextsAndSignals() error {
	dump := map[string]interface{}{}
	dump["contexts"] = a.Contexts

	chans := make(map[string]map[string]string)
	a.SignalsMu.Lock()
	for k, ch := range a.Signals {
		m := map[string]string{}
		select {
		case msg := <-ch:
			m["val"] = msg.Val
			m["err"] = msg.Err
		default:
			m["val"] = ""
			m["err"] = ""
		}
		chans[k] = m
	}
	a.SignalsMu.Unlock()
	dump["channels"] = chans

	b, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		return err
	}

	outPath := filepath.Join(a.ExeDir, "contexts_and_signals.json")
	return os.WriteFile(outPath, b, 0644)
}

// Shell returns the shell runtime, creating it if needed
func (a *App) Shell() *ShellRuntime {
	a.runtimeMu.Lock()
	defer a.runtimeMu.Unlock()

	if a.shell == nil {
		a.shell = NewShellRuntime()
	}
	return a.shell
}

// LLM returns the LLM runtime, creating it if needed
func (a *App) LLM() *LLMRuntime {
	a.runtimeMu.Lock()
	defer a.runtimeMu.Unlock()

	if a.llm == nil {
		a.llm = NewLLMRuntime()
	}
	return a.llm
}

// LocalLlama returns the local llama runtime, creating it if needed
// Note: This should be called per-workflow as llama.cpp may not be thread-safe
func (a *App) LocalLlama() *LocalLlamaRuntime {
	a.runtimeMu.Lock()
	defer a.runtimeMu.Unlock()

	if a.localLlama == nil {
		a.localLlama = NewLocalLlamaRuntime()
	}
	return a.localLlama
}
