package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider implements Provider using the Anthropic API.
type AnthropicProvider struct {
	client anthropic.Client
	model  anthropic.Model
}

// NewAnthropicProvider creates an Anthropic provider from ANTHROPIC_API_KEY env var.
func NewAnthropicProvider() (*AnthropicProvider, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}

	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = string(anthropic.ModelClaudeSonnet4_5)
	}

	client := anthropic.NewClient(option.WithAPIKey(key))
	return &AnthropicProvider{client: client, model: anthropic.Model(model)}, nil
}

func (p *AnthropicProvider) Complete(ctx context.Context, prompt string) (string, error) {
	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 8192,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
		Model: p.model,
	})
	if err != nil {
		return "", fmt.Errorf("anthropic API: %w", err)
	}

	for _, block := range message.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("anthropic API: no text content in response")
}
