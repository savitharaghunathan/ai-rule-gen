package llm

import (
	"context"
	"fmt"
	"os"
)

// Completer abstracts LLM inference for both MCP sampling and server-side LLM paths.
type Completer interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// Provider is a server-side LLM API client (Anthropic, OpenAI, Gemini, Ollama).
type Provider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// LLMCompleter calls a server-side LLM provider directly.
type LLMCompleter struct {
	provider Provider
}

// NewLLMCompleter creates a Completer that uses a server-side LLM provider.
func NewLLMCompleter(provider Provider) *LLMCompleter {
	return &LLMCompleter{provider: provider}
}

func (c *LLMCompleter) Complete(ctx context.Context, prompt string) (string, error) {
	return c.provider.Complete(ctx, prompt)
}

// NewCompleter creates a Completer for the given provider name.
// Returns nil if providerName is empty (deterministic-only mode).
// Returns an error if the provider is unknown or misconfigured (e.g., missing API key).
func NewCompleter(providerName string) (*LLMCompleter, error) {
	if providerName == "" {
		return nil, nil
	}

	var provider Provider
	var err error

	switch providerName {
	case "anthropic":
		provider, err = NewAnthropicProvider()
	case "openai":
		provider, err = NewOpenAIProvider()
	case "gemini":
		provider, err = NewGeminiProvider()
	case "ollama":
		provider, err = NewOllamaProvider()
	default:
		return nil, fmt.Errorf("unknown LLM provider %q. Valid: anthropic, openai, gemini, ollama", providerName)
	}
	if err != nil {
		return nil, fmt.Errorf("configuring %s provider: %w", providerName, err)
	}

	return NewLLMCompleter(provider), nil
}

// NewCompleterFromEnv creates a Completer based on the RULEGEN_LLM_PROVIDER env var.
// Returns nil if no provider is configured (deterministic-only mode).
func NewCompleterFromEnv() (*LLMCompleter, error) {
	return NewCompleter(os.Getenv("RULEGEN_LLM_PROVIDER"))
}
