package llm

import (
	"context"
	"fmt"
	"os"
	"time"
)

const llmCallTimeout = 5 * time.Minute

// Completer abstracts LLM inference for both MCP sampling and server-side LLM paths.
type Completer interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

// NewCompleter creates a Completer for the given provider name.
// Returns nil if providerName is empty (deterministic-only mode).
// Returns an error if the provider is unknown or misconfigured (e.g., missing API key).
func NewCompleter(providerName string) (Completer, error) {
	if providerName == "" {
		return nil, nil
	}

	var c Completer
	var err error
	switch providerName {
	case "anthropic":
		c, err = NewAnthropicProvider()
	case "openai":
		c, err = NewOpenAIProvider()
	case "gemini":
		c, err = NewGeminiProvider()
	case "ollama":
		c, err = NewOllamaProvider()
	default:
		return nil, fmt.Errorf("unknown LLM provider %q. Valid: anthropic, openai, gemini, ollama", providerName)
	}
	if err != nil {
		return nil, err
	}
	return &timeoutCompleter{inner: c, timeout: llmCallTimeout}, nil
}

// timeoutCompleter wraps a Completer with a per-call deadline.
type timeoutCompleter struct {
	inner   Completer
	timeout time.Duration
}

func (t *timeoutCompleter) Complete(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	return t.inner.Complete(ctx, prompt)
}

// NewCompleterFromEnv creates a Completer based on the RULEGEN_LLM_PROVIDER env var.
// Returns nil if no provider is configured (deterministic-only mode).
func NewCompleterFromEnv() (Completer, error) {
	return NewCompleter(os.Getenv("RULEGEN_LLM_PROVIDER"))
}
