package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/konveyor/ai-rule-gen/internal/llm"
)

// Metadata holds auto-detected source, target, and language.
type Metadata struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Language string `json:"language"`
}

// DetectMetadata uses the LLM to detect source, target, and language from content.
func DetectMetadata(ctx context.Context, completer llm.Completer, tmpl *template.Template, content string) (*Metadata, error) {
	// Use at most 4000 chars for detection — no need to send the whole doc
	if len(content) > 4000 {
		content = content[:4000]
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]string{"Content": content}); err != nil {
		return nil, fmt.Errorf("rendering detect template: %w", err)
	}

	resp, err := completer.Complete(ctx, buf.String())
	if err != nil {
		return nil, fmt.Errorf("LLM detection: %w", err)
	}

	// Extract JSON from response (handle possible markdown fencing)
	resp = strings.TrimSpace(resp)
	if idx := strings.Index(resp, "{"); idx >= 0 {
		if end := strings.LastIndex(resp, "}"); end > idx {
			resp = resp[idx : end+1]
		}
	}

	var meta Metadata
	if err := json.Unmarshal([]byte(resp), &meta); err != nil {
		return nil, fmt.Errorf("parsing detection response: %w (response: %s)", err, resp)
	}

	if meta.Source == "" || meta.Target == "" {
		return nil, fmt.Errorf("could not detect source/target from content")
	}

	return &meta, nil
}
