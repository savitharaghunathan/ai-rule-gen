package llm

import (
	"context"
	"fmt"
	"os"
)

// OpenAIProvider implements Provider using the OpenAI API.
type OpenAIProvider struct {
	apiKey string
}

// NewOpenAIProvider creates an OpenAI provider from OPENAI_API_KEY env var.
func NewOpenAIProvider() (*OpenAIProvider, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}
	return &OpenAIProvider{apiKey: key}, nil
}

func (p *OpenAIProvider) Complete(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("openai provider not yet implemented")
}
