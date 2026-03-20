package confidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	// Detect providers from test files — if Go, use --run-local directly
	// TODO: Remove once kantra ships Go toolchain in the container image.
	_, _, providers := parseTestFilesPaths(testsDir, testFiles)

	var passed, failed int
	var output string

	if needsLocalRun(providers) {
		fmt.Println("  Go provider detected — using kantra analyze --run-local (container lacks Go toolchain)")
		passed, failed, output, err = s.runKantraLocal(ctx, testsDir, testFiles, allRuleIDs)
		if err != nil {
			// Fall back to kantra test if --run-local fails
			fmt.Printf("  Warning: --run-local failed: %v — falling back to kantra test\n", err)
			passed, failed, output, err = s.runKantra(ctx, testFiles)
			if err != nil {
				return nil, fmt.Errorf("running kantra: %w", err)
			}
		}
	} else {
		fmt.Printf("  Running kantra test on %d test file(s)...\n", len(testFiles))
		passed, failed, output, err = s.runKantra(ctx, testFiles)
		if err != nil {
			return nil, fmt.Errorf("running kantra: %w", err)
		}

		// Safety net: if all rules failed unexpectedly, try --run-local
		if passed == 0 && failed > 0 {
			fmt.Println("  All rules failed in container — trying kantra analyze --run-local...")
			localPassed, localFailed, localOutput, localErr := s.runKantraLocal(ctx, testsDir, testFiles, allRuleIDs)
			if localErr != nil {
				fmt.Printf("  Warning: --run-local fallback failed: %v\n", localErr)
			} else if localPassed > passed {
				fmt.Printf("  Fallback succeeded: %d/%d passed\n", localPassed, localPassed+localFailed)
				passed = localPassed
				failed = localFailed
				output = localOutput
			}
		}
	}
	fmt.Printf("  kantra result: %d/%d passed\n", passed, passed+failed)

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
		fmt.Println("  Running LLM judge on rules...")
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
			fmt.Printf("  Judging rule %d/%d: %s...\n", i+1, len(scores), sc.RuleID)
			judgeScore, judgeVerdict, reasoning, err := s.judgeRule(ctx, r)
			if err != nil {
				fmt.Printf("  Warning: judge failed for %s: %v\n", sc.RuleID, err)
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

// runKantraLocal runs `kantra analyze --run-local` and compares violations
// against expected rules to produce pass/fail counts.
//
// TODO: Remove once kantra ships Go toolchain in the container image.
func (s *Scorer) runKantraLocal(ctx context.Context, testsDir string, testFiles []string, expectedRules []string) (passed, failed int, output string, err error) {
	rulesDir, dataDirs, providers := parseTestFilesPaths(testsDir, testFiles)
	if rulesDir == "" || len(dataDirs) == 0 {
		return 0, 0, "", fmt.Errorf("could not parse rules/data paths from test files")
	}

	outputDir, err := os.MkdirTemp("", "kantra-score-*")
	if err != nil {
		return 0, 0, "", fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(outputDir)

	args := []string{"analyze",
		"--input", dataDirs[0],
		"--rules", rulesDir,
		"--run-local",
		"--output", outputDir,
		"--overwrite",
	}
	for _, p := range providers {
		args = append(args, "--provider", p)
	}

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.kantraPath, args...)
	out, runErr := cmd.CombinedOutput()
	output = string(out)

	if ctx.Err() != nil {
		return 0, 0, output, fmt.Errorf("kantra analyze timed out")
	}
	if runErr != nil {
		return 0, 0, output, fmt.Errorf("kantra analyze failed: %w\noutput: %s", runErr, output)
	}

	// Parse output.yaml for violations
	outputFile := filepath.Join(outputDir, "output.yaml")
	matched := parseAnalyzeViolations(outputFile)

	total := len(expectedRules)
	for _, ruleID := range expectedRules {
		if matched[ruleID] {
			passed++
		}
	}
	failed = total - passed

	// Build synthetic output for parseFailedRules compatibility
	var synth strings.Builder
	for _, ruleID := range expectedRules {
		if matched[ruleID] {
			fmt.Fprintf(&synth, "%s  1/1  PASSED\n", ruleID)
		} else {
			fmt.Fprintf(&synth, "%s  0/1  PASSED\n", ruleID)
		}
	}
	fmt.Fprintf(&synth, "Rules Summary: %d/%d PASSED\n", passed, total)
	output = synth.String()

	return passed, failed, output, nil
}

// parseTestFilesPaths extracts the rules directory and data directories from test files.
func parseTestFilesPaths(testsDir string, testFiles []string) (rulesDir string, dataDirs []string, providers []string) {
	type provider struct {
		Name     string `yaml:"name"`
		DataPath string `yaml:"dataPath"`
	}
	type testFileLayout struct {
		RulesPath string     `yaml:"rulesPath"`
		Providers []provider `yaml:"providers"`
	}

	for _, f := range testFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var tf testFileLayout
		if err := yaml.Unmarshal(data, &tf); err != nil {
			continue
		}
		if tf.RulesPath != "" && rulesDir == "" {
			absRules := filepath.Join(testsDir, tf.RulesPath)
			rulesDir = filepath.Dir(absRules)
		}
		for _, p := range tf.Providers {
			if p.DataPath != "" {
				absData := filepath.Join(testsDir, p.DataPath)
				dataDirs = append(dataDirs, absData)
			}
			if p.Name != "" {
				providers = append(providers, p.Name)
			}
		}
	}
	return
}

// parseAnalyzeViolations reads kantra analyze output.yaml and returns which rule IDs had violations.
func parseAnalyzeViolations(outputFile string) map[string]bool {
	matched := make(map[string]bool)

	data, err := os.ReadFile(outputFile)
	if err != nil {
		return matched
	}

	var rulesets []struct {
		Violations map[string]interface{} `yaml:"violations"`
	}
	if err := yaml.Unmarshal(data, &rulesets); err != nil {
		return matched
	}

	for _, rs := range rulesets {
		for ruleID := range rs.Violations {
			matched[ruleID] = true
		}
	}
	return matched
}

// needsLocalRun returns true if any provider requires a local toolchain that
// the kantra container doesn't have (currently just Go).
// TODO: Remove once kantra ships Go toolchain in the container image.
func needsLocalRun(providers []string) bool {
	for _, p := range providers {
		if p == "go" {
			return true
		}
	}
	return false
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
