package llm

import (
	"context"
	"fmt"
	"os"
)

// GeminiProvider implements Provider using the Google Gemini API.
type GeminiProvider struct {
	apiKey string
}

// NewGeminiProvider creates a Gemini provider from GEMINI_API_KEY env var.
func NewGeminiProvider() (*GeminiProvider, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}
	return &GeminiProvider{apiKey: key}, nil
}

func (p *GeminiProvider) Complete(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("gemini provider not yet implemented")
}
