package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
)

// OpenAIProvider implements Provider using the OpenAI API.
type OpenAIProvider struct {
	client openai.Client
	model  openai.ChatModel
}

// NewOpenAIProvider creates an OpenAI provider from OPENAI_API_KEY env var.
func NewOpenAIProvider() (*OpenAIProvider, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = string(openai.ChatModelGPT4o)
	}

	client := openai.NewClient()
	return &OpenAIProvider{client: client, model: openai.ChatModel(model)}, nil
}

func (p *OpenAIProvider) Complete(ctx context.Context, prompt string) (string, error) {
	completion, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		Model: p.model,
	})
	if err != nil {
		return "", fmt.Errorf("openai API: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("openai API: no choices in response")
	}

	return completion.Choices[0].Message.Content, nil
}
