package llm

import (
	"context"
	"fmt"
	"os"
)

// AnthropicProvider implements Provider using the Anthropic API.
type AnthropicProvider struct {
	apiKey string
}

// NewAnthropicProvider creates an Anthropic provider from ANTHROPIC_API_KEY env var.
func NewAnthropicProvider() (*AnthropicProvider, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is required")
	}
	return &AnthropicProvider{apiKey: key}, nil
}

func (p *AnthropicProvider) Complete(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("anthropic provider not yet implemented")
}
