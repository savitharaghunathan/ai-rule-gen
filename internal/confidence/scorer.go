package confidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

// Score holds the confidence score for a single rule.
type Score struct {
	RuleID               string  `json:"rule_id" yaml:"ruleID"`
	PatternCorrectness   float64 `json:"pattern_correctness" yaml:"patternCorrectness"`
	MessageQuality       float64 `json:"message_quality" yaml:"messageQuality"`
	CategoryAppropriate  float64 `json:"category_appropriateness" yaml:"categoryAppropriateness"`
	EffortAccuracy       float64 `json:"effort_accuracy" yaml:"effortAccuracy"`
	FalsePositiveRisk    float64 `json:"false_positive_risk" yaml:"falsePositiveRisk"`
	Overall              float64 `json:"overall" yaml:"overall"`
	Verdict              string  `json:"verdict" yaml:"verdict"`
	Reasoning            string  `json:"reasoning" yaml:"reasoning"`
}

// ScoreReport is the full confidence report for a set of rules.
type ScoreReport struct {
	Scores  []Score `json:"scores" yaml:"scores"`
	Summary Summary `json:"summary" yaml:"summary"`
}

// Summary provides aggregate statistics.
type Summary struct {
	TotalRules    int     `json:"total_rules" yaml:"totalRules"`
	Accepted      int     `json:"accepted" yaml:"accepted"`
	NeedsReview   int     `json:"needs_review" yaml:"needsReview"`
	Rejected      int     `json:"rejected" yaml:"rejected"`
	AverageScore  float64 `json:"average_score" yaml:"averageScore"`
}

// Scorer evaluates rule quality using LLM-as-judge.
type Scorer struct {
	completer llm.Completer
	tmpl      *template.Template
}

// New creates a Scorer.
func New(completer llm.Completer, tmpl *template.Template) *Scorer {
	return &Scorer{completer: completer, tmpl: tmpl}
}

// ScoreRules evaluates each rule independently and returns a report.
func (s *Scorer) ScoreRules(ctx context.Context, ruleList []rules.Rule) (*ScoreReport, error) {
	var scores []Score

	for i, r := range ruleList {
		fmt.Printf("  Scoring rule %d/%d: %s\n", i+1, len(ruleList), r.RuleID)

		score, err := s.scoreRule(ctx, r)
		if err != nil {
			return nil, fmt.Errorf("scoring %s: %w", r.RuleID, err)
		}
		scores = append(scores, score)
	}

	report := &ScoreReport{
		Scores:  scores,
		Summary: computeSummary(scores),
	}

	return report, nil
}

func (s *Scorer) scoreRule(ctx context.Context, r rules.Rule) (Score, error) {
	// Marshal rule to YAML for the judge prompt
	ruleYAML, err := yaml.Marshal([]rules.Rule{r})
	if err != nil {
		return Score{}, fmt.Errorf("marshaling rule: %w", err)
	}

	var buf bytes.Buffer
	data := map[string]string{
		"RuleYAML": string(ruleYAML),
	}
	if err := s.tmpl.Execute(&buf, data); err != nil {
		return Score{}, fmt.Errorf("rendering template: %w", err)
	}

	response, err := s.completer.Complete(ctx, buf.String())
	if err != nil {
		return Score{}, fmt.Errorf("LLM scoring: %w", err)
	}

	score, err := parseScore(response, r.RuleID)
	if err != nil {
		return Score{}, fmt.Errorf("parsing score: %w", err)
	}

	return score, nil
}

func parseScore(response, ruleID string) (Score, error) {
	// Extract JSON from response
	response = strings.TrimSpace(response)
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end < 0 || end <= start {
		return Score{}, fmt.Errorf("no JSON object found in response")
	}
	jsonStr := response[start : end+1]

	var raw struct {
		PatternCorrectness  float64 `json:"pattern_correctness"`
		MessageQuality      float64 `json:"message_quality"`
		CategoryAppropriate float64 `json:"category_appropriateness"`
		EffortAccuracy      float64 `json:"effort_accuracy"`
		FalsePositiveRisk   float64 `json:"false_positive_risk"`
		Reasoning           string  `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return Score{}, fmt.Errorf("parsing JSON: %w (response: %s)", err, jsonStr)
	}

	overall := (raw.PatternCorrectness + raw.MessageQuality + raw.CategoryAppropriate +
		raw.EffortAccuracy + raw.FalsePositiveRisk) / 5.0

	verdict := "review"
	if overall >= 4.0 {
		verdict = "accept"
	} else if overall < 2.5 {
		verdict = "reject"
	}

	return Score{
		RuleID:              ruleID,
		PatternCorrectness:  raw.PatternCorrectness,
		MessageQuality:      raw.MessageQuality,
		CategoryAppropriate: raw.CategoryAppropriate,
		EffortAccuracy:      raw.EffortAccuracy,
		FalsePositiveRisk:   raw.FalsePositiveRisk,
		Overall:             overall,
		Verdict:             verdict,
		Reasoning:           raw.Reasoning,
	}, nil
}

func computeSummary(scores []Score) Summary {
	s := Summary{TotalRules: len(scores)}
	var total float64
	for _, sc := range scores {
		total += sc.Overall
		switch sc.Verdict {
		case "accept":
			s.Accepted++
		case "review":
			s.NeedsReview++
		case "reject":
			s.Rejected++
		}
	}
	if len(scores) > 0 {
		s.AverageScore = total / float64(len(scores))
	}
	return s
}
