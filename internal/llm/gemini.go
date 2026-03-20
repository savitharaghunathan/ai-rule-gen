package llm

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// GeminiProvider implements Provider using the Google Gemini API.
type GeminiProvider struct {
	client *genai.Client
	model  string
}

// NewGeminiProvider creates a Gemini provider from GEMINI_API_KEY env var.
func NewGeminiProvider() (*GeminiProvider, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: key,
	})
	if err != nil {
		return nil, fmt.Errorf("creating gemini client: %w", err)
	}

	return &GeminiProvider{client: client, model: model}, nil
}

func (p *GeminiProvider) Complete(ctx context.Context, prompt string) (string, error) {
	result, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(prompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini API: %w", err)
	}

	text := result.Text()
	if text == "" {
		return "", fmt.Errorf("gemini API: no text content in response")
	}

	return text, nil
}
