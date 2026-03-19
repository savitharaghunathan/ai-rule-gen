package llm

import (
	"context"
	"fmt"
	"os"
)

// OllamaProvider implements Provider using a local Ollama server.
type OllamaProvider struct {
	host  string
	model string
}

// NewOllamaProvider creates an Ollama provider from OLLAMA_HOST and OLLAMA_MODEL env vars.
func NewOllamaProvider() (*OllamaProvider, error) {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{host: host, model: model}, nil
}

func (p *OllamaProvider) Complete(ctx context.Context, prompt string) (string, error) {
	return "", fmt.Errorf("ollama provider not yet implemented")
}
