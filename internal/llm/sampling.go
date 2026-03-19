package llm

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SamplingCompleter uses MCP sampling to ask the client's LLM.
// It requires a ServerSession obtained from the tool handler's request.
type SamplingCompleter struct {
	session *mcp.ServerSession
}

// NewSamplingCompleter creates a Completer that uses MCP sampling.
// Returns an error if the client does not support sampling.
func NewSamplingCompleter(session *mcp.ServerSession) (*SamplingCompleter, error) {
	params := session.InitializeParams()
	if params == nil {
		return nil, fmt.Errorf("session not initialized")
	}

	slog.Info("client capabilities",
		"client", params.ClientInfo.Name,
		"version", params.ClientInfo.Version,
		"sampling", params.Capabilities.Sampling != nil,
	)

	if params.Capabilities.Sampling == nil {
		return nil, fmt.Errorf("client %q does not support MCP sampling", params.ClientInfo.Name)
	}

	return &SamplingCompleter{session: session}, nil
}

func (c *SamplingCompleter) Complete(ctx context.Context, prompt string) (string, error) {
	result, err := c.session.CreateMessage(ctx, &mcp.CreateMessageParams{
		MaxTokens: 8192,
		Messages: []*mcp.SamplingMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: prompt},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("MCP sampling: %w", err)
	}

	text, ok := result.Content.(*mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("MCP sampling: unexpected content type %T", result.Content)
	}

	return text.Text, nil
}
