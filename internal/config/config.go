// Package config provides centralized configuration management for llm-compiler.
// It handles environment variables, default values, and configuration validation.
package config

import (
	"os"
	"strconv"
	"sync"
)

// Config holds all configuration settings for llm-compiler
type Config struct {
	// OpenAI settings
	OpenAIAPIKey   string
	OpenAIBaseURL  string
	OpenAIModel    string

	// Llama settings
	LlamaModelPath string
	LlamaThreads   int

	// Runtime settings
	UseSubprocess  bool
	WorkerTimeout  int // seconds
	MaxRetries     int

	// Output settings
	FmtOutputFile   string
	LlamaOutputFile string
	Verbose         bool
	DebugMode       bool
}

var (
	globalConfig *Config
	configOnce   sync.Once
)

// Default values
const (
	DefaultOpenAIModel    = "gpt-4"
	DefaultOpenAIBaseURL  = "https://api.openai.com/v1"
	DefaultLlamaThreads   = 4
	DefaultWorkerTimeout  = 300
	DefaultMaxRetries     = 3
	DefaultFmtOutputFile  = "fmt_output.txt"
	DefaultLlamaOutputFile = "llama_output.txt"
)

// Get returns the global configuration, loading from environment if not already loaded
func Get() *Config {
	configOnce.Do(func() {
		globalConfig = loadFromEnv()
	})
	return globalConfig
}

// Reset clears the global configuration, forcing reload on next Get()
// This is primarily useful for testing
func Reset() {
	configOnce = sync.Once{}
	globalConfig = nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv() *Config {
	return &Config{
		// OpenAI settings
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
		OpenAIBaseURL: getEnv("OPENAI_BASE_URL", DefaultOpenAIBaseURL),
		OpenAIModel:   getEnv("OPENAI_MODEL", DefaultOpenAIModel),

		// Llama settings
		LlamaModelPath: getEnv("LLAMA_MODEL_PATH", ""),
		LlamaThreads:   getEnvInt("LLAMA_THREADS", DefaultLlamaThreads),

		// Runtime settings
		UseSubprocess: getEnvBool("LLMC_SUBPROCESS", false),
		WorkerTimeout: getEnvInt("LLMC_WORKER_TIMEOUT", DefaultWorkerTimeout),
		MaxRetries:    getEnvInt("LLMC_MAX_RETRIES", DefaultMaxRetries),

		// Output settings
		FmtOutputFile:   getEnv("LLMC_FMT_OUTPUT", DefaultFmtOutputFile),
		LlamaOutputFile: getEnv("LLMC_LLAMA_OUTPUT", DefaultLlamaOutputFile),
		Verbose:         getEnvBool("LLMC_VERBOSE", false),
		DebugMode:       getEnvBool("LLMC_DEBUG", false),
	}
}

// NewConfig creates a new configuration with custom values
// This is useful for testing or programmatic configuration
func NewConfig() *Config {
	return &Config{
		OpenAIBaseURL:   DefaultOpenAIBaseURL,
		OpenAIModel:     DefaultOpenAIModel,
		LlamaThreads:    DefaultLlamaThreads,
		WorkerTimeout:   DefaultWorkerTimeout,
		MaxRetries:      DefaultMaxRetries,
		FmtOutputFile:   DefaultFmtOutputFile,
		LlamaOutputFile: DefaultLlamaOutputFile,
	}
}

// WithOpenAI configures OpenAI settings
func (c *Config) WithOpenAI(apiKey, baseURL, model string) *Config {
	c.OpenAIAPIKey = apiKey
	if baseURL != "" {
		c.OpenAIBaseURL = baseURL
	}
	if model != "" {
		c.OpenAIModel = model
	}
	return c
}

// WithLlama configures Llama settings
func (c *Config) WithLlama(modelPath string, threads int) *Config {
	c.LlamaModelPath = modelPath
	if threads > 0 {
		c.LlamaThreads = threads
	}
	return c
}

// WithSubprocess enables subprocess mode
func (c *Config) WithSubprocess(enabled bool) *Config {
	c.UseSubprocess = enabled
	return c
}

// WithOutput configures output file paths
func (c *Config) WithOutput(fmtOutput, llamaOutput string) *Config {
	if fmtOutput != "" {
		c.FmtOutputFile = fmtOutput
	}
	if llamaOutput != "" {
		c.LlamaOutputFile = llamaOutput
	}
	return c
}

// WithDebug enables debug and verbose modes
func (c *Config) WithDebug(debug, verbose bool) *Config {
	c.DebugMode = debug
	c.Verbose = verbose
	return c
}

// Validate checks if the configuration is valid for the intended use
func (c *Config) Validate() error {
	// Add validation logic here as needed
	return nil
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		// Also accept "1" as true
		if value == "1" {
			return true
		}
	}
	return defaultValue
}
