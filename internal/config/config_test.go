package config_test

import (
	"os"
	"testing"

	"github.com/LiboWorks/llm-compiler/internal/config"
)

func TestGetConfig(t *testing.T) {
	// Reset to get fresh config
	config.Reset()

	cfg := config.Get()
	if cfg == nil {
		t.Fatal("config should not be nil")
	}

	// Check defaults
	if cfg.OpenAIModel != config.DefaultOpenAIModel {
		t.Errorf("expected default OpenAI model %q, got %q", config.DefaultOpenAIModel, cfg.OpenAIModel)
	}

	if cfg.LlamaThreads != config.DefaultLlamaThreads {
		t.Errorf("expected default llama threads %d, got %d", config.DefaultLlamaThreads, cfg.LlamaThreads)
	}

	if cfg.FmtOutputFile != config.DefaultFmtOutputFile {
		t.Errorf("expected default fmt output file %q, got %q", config.DefaultFmtOutputFile, cfg.FmtOutputFile)
	}
}

func TestConfigFromEnv(t *testing.T) {
	// Reset and set env vars
	config.Reset()

	os.Setenv("LLMC_VERBOSE", "true")
	os.Setenv("LLMC_DEBUG", "1")
	os.Setenv("LLMC_SUBPROCESS", "1")
	defer func() {
		os.Unsetenv("LLMC_VERBOSE")
		os.Unsetenv("LLMC_DEBUG")
		os.Unsetenv("LLMC_SUBPROCESS")
	}()

	cfg := config.Get()

	if !cfg.Verbose {
		t.Error("expected Verbose to be true")
	}

	if !cfg.DebugMode {
		t.Error("expected DebugMode to be true")
	}

	if !cfg.UseSubprocess {
		t.Error("expected UseSubprocess to be true")
	}
}

func TestNewConfigBuilder(t *testing.T) {
	cfg := config.NewConfig().
		WithOpenAI("test-key", "https://custom.api", "gpt-4").
		WithLlama("/path/to/model.gguf", 8).
		WithSubprocess(true).
		WithOutput("custom_fmt.txt", "custom_llama.txt").
		WithDebug(true, true)

	if cfg.OpenAIAPIKey != "test-key" {
		t.Errorf("expected API key 'test-key', got %q", cfg.OpenAIAPIKey)
	}

	if cfg.OpenAIBaseURL != "https://custom.api" {
		t.Errorf("expected base URL 'https://custom.api', got %q", cfg.OpenAIBaseURL)
	}

	if cfg.OpenAIModel != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", cfg.OpenAIModel)
	}

	if cfg.LlamaModelPath != "/path/to/model.gguf" {
		t.Errorf("expected model path '/path/to/model.gguf', got %q", cfg.LlamaModelPath)
	}

	if cfg.LlamaThreads != 8 {
		t.Errorf("expected 8 threads, got %d", cfg.LlamaThreads)
	}

	if !cfg.UseSubprocess {
		t.Error("expected UseSubprocess to be true")
	}

	if cfg.FmtOutputFile != "custom_fmt.txt" {
		t.Errorf("expected fmt output 'custom_fmt.txt', got %q", cfg.FmtOutputFile)
	}

	if cfg.LlamaOutputFile != "custom_llama.txt" {
		t.Errorf("expected llama output 'custom_llama.txt', got %q", cfg.LlamaOutputFile)
	}

	if !cfg.DebugMode || !cfg.Verbose {
		t.Error("expected debug and verbose to be true")
	}
}

func TestConfigSingleton(t *testing.T) {
	config.Reset()

	cfg1 := config.Get()
	cfg2 := config.Get()

	if cfg1 != cfg2 {
		t.Error("Get() should return the same instance")
	}
}
