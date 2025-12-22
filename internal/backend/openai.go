package backend

import (
	"context"
	"fmt"

	"github.com/LiboWorks/llm-compiler/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

// OpenAIBackend implements LLMBackend using the OpenAI API.
type OpenAIBackend struct {
	client       *openai.Client
	defaultModel string
}

// OpenAIConfig holds configuration for the OpenAI backend.
type OpenAIConfig struct {
	APIKey       string
	BaseURL      string // Optional: for Azure or compatible APIs
	DefaultModel string
}

// NewOpenAIBackend creates a new OpenAI backend.
func NewOpenAIBackend(cfg OpenAIConfig) (*OpenAIBackend, error) {
	globalCfg := config.Get()

	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = globalCfg.OpenAIAPIKey
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not provided (set OPENAI_API_KEY or pass in config)")
	}

	clientCfg := openai.DefaultConfig(apiKey)
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = globalCfg.OpenAIBaseURL
	}
	if baseURL != "" {
		clientCfg.BaseURL = baseURL
	}

	defaultModel := cfg.DefaultModel
	if defaultModel == "" {
		defaultModel = globalCfg.OpenAIModel
	}

	return &OpenAIBackend{
		client:       openai.NewClientWithConfig(clientCfg),
		defaultModel: defaultModel,
	}, nil
}

// Generate implements LLMBackend.
func (b *OpenAIBackend) Generate(ctx context.Context, prompt string, model string, maxTokens int) (string, error) {
	if model == "" {
		model = b.defaultModel
	}

	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	}

	if maxTokens > 0 {
		req.MaxTokens = maxTokens
	}

	resp, err := b.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return resp.Choices[0].Message.Content, nil
}

// Name implements LLMBackend.
func (b *OpenAIBackend) Name() string {
	return "openai"
}

// Close implements LLMBackend.
func (b *OpenAIBackend) Close() error {
	return nil
}
