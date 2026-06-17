package verify

import (
	"fmt"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

type Verifier interface {
	Verify(pattern rules.MigrationPattern) (Result, error)
	Language() string
}

func NewVerifier(language, cacheDir string) Verifier {
	switch language {
	case "java":
		return NewJavaVerifier(cacheDir)
	case "go":
		return NewGoVerifier(cacheDir)
	default:
		return nil
	}
}

func Run(extract *rules.ExtractOutput, cacheDir string) ([]Result, error) {
	v := NewVerifier(extract.Language, cacheDir)

	results := make([]Result, 0, len(extract.Patterns))
	for i, p := range extract.Patterns {
		if v == nil {
			results = append(results, Result{
				PatternIndex: i,
				SourceFQN:    p.SourceFQN,
				Status:       StatusSkipped,
				Reason:       fmt.Sprintf("no verifier for language %q", extract.Language),
			})
			continue
		}
		r, err := v.Verify(p)
		if err != nil {
			return nil, fmt.Errorf("verifying pattern %d (%s): %w", i, p.SourceFQN, err)
		}
		r.PatternIndex = i
		results = append(results, r)
	}
	return results, nil
}
