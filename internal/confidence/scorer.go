package confidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

// Score holds the confidence result for a single rule.
type Score struct {
	RuleID        string `json:"rule_id" yaml:"ruleID"`
	TestPassed    bool   `json:"test_passed" yaml:"testPassed"`
	Verdict       string `json:"verdict" yaml:"verdict"`
	FailureReason string `json:"failure_reason,omitempty" yaml:"failureReason,omitempty"`
	// LLM judge fields (populated when --provider is set)
	JudgeScore    float64 `json:"judge_score,omitempty" yaml:"judgeScore,omitempty"`
	JudgeVerdict  string  `json:"judge_verdict,omitempty" yaml:"judgeVerdict,omitempty"`
	JudgeReasoning string `json:"judge_reasoning,omitempty" yaml:"judgeReasoning,omitempty"`
}

// ScoreReport is the full confidence report.
type ScoreReport struct {
	Scores  []Score `json:"scores" yaml:"scores"`
	Summary Summary `json:"summary" yaml:"summary"`
}

// Summary provides aggregate statistics.
type Summary struct {
	TotalRules int     `json:"total_rules" yaml:"totalRules"`
	Passed     int     `json:"passed" yaml:"passed"`
	Failed     int     `json:"failed" yaml:"failed"`
	PassRate   float64 `json:"pass_rate" yaml:"passRate"`
}

// Scorer runs kantra tests and optionally uses LLM-as-judge to evaluate rules.
type Scorer struct {
	kantraPath string
	timeout    time.Duration
	completer  llm.Completer
	judgeTmpl  *template.Template
}

// New creates a Scorer. kantraPath defaults to "kantra" on PATH.
// completer and judgeTmpl are optional — if provided, LLM-as-judge runs after kantra.
func New(kantraPath string, timeoutSeconds int, completer llm.Completer, judgeTmpl *template.Template) *Scorer {
	if kantraPath == "" {
		kantraPath = "kantra"
	}
	if timeoutSeconds <= 0 {
		timeoutSeconds = 900
	}
	return &Scorer{
		kantraPath: kantraPath,
		timeout:    time.Duration(timeoutSeconds) * time.Second,
		completer:  completer,
		judgeTmpl:  judgeTmpl,
	}
}

// ScoreRules runs kantra test on .test.yaml files in testsDir, optionally runs LLM judge,
// and returns a report. rulesDir is needed only if completer is set (for LLM judge).
func (s *Scorer) ScoreRules(ctx context.Context, testsDir string, rulesDir string) (*ScoreReport, error) {
	testFiles, err := findTestFiles(testsDir)
	if err != nil {
		return nil, fmt.Errorf("finding test files: %w", err)
	}
	if len(testFiles) == 0 {
		return nil, fmt.Errorf("no .test.yaml files found in %s", testsDir)
	}

	allRuleIDs, err := collectRuleIDs(testFiles)
	if err != nil {
		return nil, fmt.Errorf("collecting rule IDs: %w", err)
	}

	slog.Info("running kantra tests", "test_files", len(testFiles))
	var passed, failed int
	var output string
	passed, failed, output, err = s.runKantra(ctx, testFiles)
	if err != nil {
		return nil, fmt.Errorf("running kantra: %w", err)
	}
	slog.Info("kantra tests complete", "passed", passed, "total", passed+failed)

	failedRules := parseFailedRules(output)

	// Build per-rule scores from kantra results
	scores := make([]Score, 0, len(allRuleIDs))
	for _, ruleID := range allRuleIDs {
		reason, isFailed := failedRules[ruleID]
		score := Score{
			RuleID:     ruleID,
			TestPassed: !isFailed,
			Verdict:    "accept",
		}
		if isFailed {
			score.Verdict = "reject"
			score.FailureReason = reason
		}
		scores = append(scores, score)
	}

	// Run LLM-as-judge (secondary signal, optional)
	if s.completer != nil && s.judgeTmpl != nil && rulesDir != "" {
		slog.Info("running LLM judge")
		ruleList, err := rules.ReadRulesDir(rulesDir)
		if err != nil {
			return nil, fmt.Errorf("reading rules for judge: %w", err)
		}
		ruleMap := make(map[string]rules.Rule)
		for _, r := range ruleList {
			ruleMap[r.RuleID] = r
		}

		for i, sc := range scores {
			r, ok := ruleMap[sc.RuleID]
			if !ok {
				continue
			}
			slog.Info("judging rule", "index", i+1, "total", len(scores), "rule_id", sc.RuleID)
			judgeScore, judgeVerdict, reasoning, err := s.judgeRule(ctx, r)
			if err != nil {
				slog.Warn("judge failed", "rule_id", sc.RuleID, "error", err)
				continue
			}
			scores[i].JudgeScore = judgeScore
			scores[i].JudgeVerdict = judgeVerdict
			scores[i].JudgeReasoning = reasoning

			// If kantra passed but judge says reject, downgrade to review
			if scores[i].TestPassed && judgeVerdict == "reject" {
				scores[i].Verdict = "review"
			}
		}
	}

	report := &ScoreReport{
		Scores:  scores,
		Summary: computeSummary(scores),
	}
	return report, nil
}

// judgeRule uses LLM-as-judge to evaluate a single rule's quality.
func (s *Scorer) judgeRule(ctx context.Context, r rules.Rule) (score float64, verdict, reasoning string, err error) {
	ruleYAML, err := yaml.Marshal([]rules.Rule{r})
	if err != nil {
		return 0, "", "", fmt.Errorf("marshaling rule: %w", err)
	}

	var buf bytes.Buffer
	data := map[string]string{"RuleYAML": string(ruleYAML)}
	if err := s.judgeTmpl.Execute(&buf, data); err != nil {
		return 0, "", "", fmt.Errorf("rendering template: %w", err)
	}

	response, err := s.completer.Complete(ctx, buf.String())
	if err != nil {
		return 0, "", "", fmt.Errorf("LLM scoring: %w", err)
	}

	return parseJudgeResponse(response)
}

func parseJudgeResponse(response string) (score float64, verdict, reasoning string, err error) {
	response = strings.TrimSpace(response)
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start < 0 || end < 0 || end <= start {
		return 0, "", "", fmt.Errorf("no JSON object found in response")
	}

	var raw struct {
		PatternCorrectness  float64 `json:"pattern_correctness"`
		MessageQuality      float64 `json:"message_quality"`
		CategoryAppropriate float64 `json:"category_appropriateness"`
		EffortAccuracy      float64 `json:"effort_accuracy"`
		FalsePositiveRisk   float64 `json:"false_positive_risk"`
		Reasoning           string  `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(response[start:end+1]), &raw); err != nil {
		return 0, "", "", fmt.Errorf("parsing JSON: %w", err)
	}

	score = (raw.PatternCorrectness + raw.MessageQuality + raw.CategoryAppropriate +
		raw.EffortAccuracy + raw.FalsePositiveRisk) / 5.0

	verdict = "review"
	if score >= 4.0 {
		verdict = "accept"
	} else if score < 2.5 {
		verdict = "reject"
	}

	return score, verdict, raw.Reasoning, nil
}

// --- kantra integration ---

func findTestFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".test.yaml") || strings.HasSuffix(e.Name(), ".test.yml") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

type testFileSpec struct {
	Tests []struct {
		RuleID string `yaml:"ruleID"`
	} `yaml:"tests"`
}

func collectRuleIDs(testFiles []string) ([]string, error) {
	var ids []string
	seen := make(map[string]bool)
	for _, f := range testFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f, err)
		}
		var spec testFileSpec
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", f, err)
		}
		for _, t := range spec.Tests {
			if t.RuleID != "" && !seen[t.RuleID] {
				ids = append(ids, t.RuleID)
				seen[t.RuleID] = true
			}
		}
	}
	return ids, nil
}

func (s *Scorer) runKantra(ctx context.Context, testFiles []string) (passed, failed int, output string, err error) {
	args := []string{"test"}
	args = append(args, testFiles...)

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.kantraPath, args...)
	out, runErr := cmd.CombinedOutput()
	output = string(out)

	passed, failed = parseSummary(output)

	if ctx.Err() != nil {
		return passed, failed, output, fmt.Errorf("kantra timed out after %s", s.timeout)
	}

	// kantra returns non-zero if tests fail — that's expected, not an error
	if runErr != nil && passed == 0 && failed == 0 {
		return 0, 0, output, fmt.Errorf("kantra failed: %w\noutput: %s", runErr, output)
	}

	return passed, failed, output, nil
}

var reSummary = regexp.MustCompile(`Rules Summary:\s+(\d+)/(\d+)`)
var reRuleFail = regexp.MustCompile(`([\w-]+-\d{5})\s+0/\d+\s+PASSED`)

func parseSummary(output string) (passed, failed int) {
	m := reSummary.FindStringSubmatch(output)
	if len(m) == 3 {
		fmt.Sscanf(m[1], "%d", &passed)
		var total int
		fmt.Sscanf(m[2], "%d", &total)
		failed = total - passed
	}
	return
}

func parseFailedRules(output string) map[string]string {
	failed := make(map[string]string)
	matches := reRuleFail.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		failed[m[1]] = "rule did not match any incidents in test data"
	}
	return failed
}

func computeSummary(scores []Score) Summary {
	s := Summary{TotalRules: len(scores)}
	for _, sc := range scores {
		if sc.TestPassed {
			s.Passed++
		} else {
			s.Failed++
		}
	}
	if s.TotalRules > 0 {
		s.PassRate = float64(s.Passed) / float64(s.TotalRules) * 100
	}
	return s
}
