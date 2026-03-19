package extraction

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/konveyor/ai-rule-gen/internal/llm"
)

// Extractor extracts migration patterns from content using an LLM.
type Extractor struct {
	completer llm.Completer
	tmpl      *template.Template
}

// New creates an Extractor with the given LLM completer and prompt template.
func New(completer llm.Completer, tmpl *template.Template) *Extractor {
	return &Extractor{completer: completer, tmpl: tmpl}
}

// ExtractInput holds the template data for pattern extraction.
type ExtractInput struct {
	Content  string
	Source   string
	Target   string
	Language string
}

// Extract runs LLM-driven pattern extraction on each content chunk,
// deduplicates, and returns the combined patterns.
func (e *Extractor) Extract(ctx context.Context, chunks []string, source, target, language string) ([]MigrationPattern, error) {
	var allPatterns []MigrationPattern

	for i, chunk := range chunks {
		patterns, err := e.extractChunk(ctx, chunk, source, target, language)
		if err != nil {
			return nil, fmt.Errorf("chunk %d: %w", i, err)
		}
		allPatterns = append(allPatterns, patterns...)
	}

	return Deduplicate(allPatterns), nil
}

func (e *Extractor) extractChunk(ctx context.Context, content, source, target, language string) ([]MigrationPattern, error) {
	var buf bytes.Buffer
	err := e.tmpl.Execute(&buf, ExtractInput{
		Content:  content,
		Source:   source,
		Target:   target,
		Language: language,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering extraction template: %w", err)
	}

	response, err := e.completer.Complete(ctx, buf.String())
	if err != nil {
		return nil, fmt.Errorf("LLM extraction: %w", err)
	}

	return parsePatterns(response)
}

// parsePatterns extracts a JSON array of MigrationPattern from an LLM response.
// It handles responses that may include markdown fences or surrounding text.
func parsePatterns(response string) ([]MigrationPattern, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON array found in LLM response")
	}

	var patterns []MigrationPattern
	if err := json.Unmarshal([]byte(jsonStr), &patterns); err != nil {
		return nil, fmt.Errorf("parsing LLM response as JSON: %w", err)
	}

	if len(patterns) == 0 {
		return nil, fmt.Errorf("no migration patterns found")
	}

	return patterns, nil
}

// extractJSON finds a JSON array in the response, handling markdown fences.
func extractJSON(s string) string {
	// Strip markdown code fences if present
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.SplitN(s, "\n", 2)
		if len(lines) == 2 {
			s = lines[1]
		}
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}

	// Find the JSON array
	start := strings.Index(s, "[")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(s, "]")
	if end < 0 || end <= start {
		return ""
	}
	return s[start : end+1]
}
