package runtime

import (
	"context"
	"fmt"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type LLMRuntime struct {
	client *openai.Client
}

func NewLLMRuntime() *LLMRuntime {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️ OPENAI_API_KEY not set, LLM won't work")
	}
	return &LLMRuntime{
		client: openai.NewClient(apiKey),
	}
}

func (r *LLMRuntime) Generate(prompt string, model string) (string, error) {
	resp, err := r.client.CreateChatCompletion(context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: "user", Content: prompt},
			},
		})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
