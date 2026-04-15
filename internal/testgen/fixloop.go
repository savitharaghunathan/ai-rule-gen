package testgen

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// FixLoopResult holds the result of generate + test + fix loop.
type FixLoopResult struct {
	GenerateOutput *GenerateOutput `json:"generate"`
	TestResult     *TestResult     `json:"test_result"`
	Iterations     int             `json:"iterations"`
}

// RunWithTests generates test data, runs kantra, and iterates on failures using code hints.
func (g *Generator) RunWithTests(ctx context.Context, input GenerateInput, fixTmpl *template.Template, maxIterations int) (*FixLoopResult, error) {
	if maxIterations <= 0 {
		maxIterations = 3
	}

	loopStart := time.Now()

	// Step 1: Generate initial test data
	stepStart := time.Now()
	slog.Info("generating initial test data")
	genOutput, err := g.Generate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("generating test data: %w", err)
	}
	slog.Info("test data generation complete", "groups", len(genOutput.DataDirs), "rules", genOutput.RulesTested, "duration", time.Since(stepStart).Round(time.Millisecond))

	testsDir := filepath.Join(input.OutputDir, "tests")

	// Step 2: Run kantra tests, iterate with fix hints on failures
	var testResult *TestResult
	var iterationsRun int
	for iteration := 1; iteration <= maxIterations; iteration++ {
		iterationsRun = iteration
		stepStart = time.Now()
		slog.Info("running kantra tests", "iteration", iteration, "max_iterations", maxIterations)
		testResult, err = RunKantraTests(ctx, testsDir, 900)
		if err != nil {
			return nil, fmt.Errorf("running kantra (iteration %d): %w", iteration, err)
		}
		slog.Info("kantra tests complete", "iteration", iteration, "passed", testResult.Passed, "total", testResult.Total, "pass_rate", fmt.Sprintf("%.0f%%", testResult.PassRate), "duration", time.Since(stepStart).Round(time.Millisecond))

		// All passed — done
		if testResult.Passed == testResult.Total && testResult.Total > 0 {
			slog.Info("all tests passed")
			break
		}

		// No failures parsed but tests didn't all pass
		if len(testResult.Failures) == 0 && testResult.Passed < testResult.Total {
			slog.Warn("kantra reported failures but could not parse which rules failed", "raw_output", testResult.RawOutput)
			break
		}

		// Last iteration — report and stop
		if iteration == maxIterations {
			failedIDs := make([]string, len(testResult.Failures))
			for i, f := range testResult.Failures {
				failedIDs[i] = f.RuleID
			}
			slog.Warn("rules still failing after max iterations", "failed_rules", failedIDs, "iterations", maxIterations)
			break
		}

		// Fix failing rules via code hints and regenerate
		stepStart = time.Now()
		slog.Info("fixing failing rules", "count", len(testResult.Failures))
		codeHints, err := g.generateFixHints(ctx, testResult.Failures, fixTmpl, input)
		if err != nil {
			slog.Warn("fix hints failed", "error", err)
			continue
		}

		err = g.regenerateWithHints(ctx, input, codeHints)
		if err != nil {
			slog.Warn("regeneration failed", "error", err)
			continue
		}
		slog.Info("fix iteration complete", "iteration", iteration, "hints_generated", len(codeHints), "duration", time.Since(stepStart).Round(time.Millisecond))
	}

	slog.Info("test loop complete", "total_duration", time.Since(loopStart).Round(time.Millisecond))

	return &FixLoopResult{
		GenerateOutput: genOutput,
		TestResult:     testResult,
		Iterations:     iterationsRun,
	}, nil
}

// generateFixHints asks the LLM for code hints for each failing rule.
func (g *Generator) generateFixHints(ctx context.Context, failures []Failure, fixTmpl *template.Template, input GenerateInput) (map[string]string, error) {
	hints := make(map[string]string)

	ruleList, err := rules.ReadRulesDir(input.RulesDir)
	if err != nil {
		return nil, fmt.Errorf("reading rules: %w", err)
	}
	ruleMap := make(map[string]rules.Rule)
	for _, r := range ruleList {
		ruleMap[r.RuleID] = r
	}

	language := input.Language
	if language == "" {
		language = detectLanguage(ruleList)
		if language == "" {
			language = "go"
		}
	}

	for _, f := range failures {
		r, ok := ruleMap[f.RuleID]
		if !ok {
			continue
		}

		pattern, provider := extractRulePattern(r)
		if pattern == "" {
			continue
		}

		hint, err := g.getCodeHint(ctx, fixTmpl, f.RuleID, pattern, provider, language)
		if err != nil {
			slog.Warn("could not get fix hint", "rule_id", f.RuleID, "error", err)
			continue
		}
		hints[f.RuleID] = hint
		slog.Info("generated fix hint", "rule_id", f.RuleID, "hint", hint)
	}

	return hints, nil
}

// getCodeHint calls the LLM with the fix_hint template.
func (g *Generator) getCodeHint(ctx context.Context, fixTmpl *template.Template, ruleID, pattern, provider, language string) (string, error) {
	var buf bytes.Buffer
	data := map[string]string{
		"RuleID":   ruleID,
		"Pattern":  pattern,
		"Provider": provider,
		"Language": language,
	}
	if err := fixTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering fix template: %w", err)
	}

	response, err := g.completer.Complete(ctx, buf.String())
	if err != nil {
		return "", fmt.Errorf("LLM fix hint: %w", err)
	}

	// Clean up: remove markdown fencing, trim
	hint := strings.TrimSpace(response)
	hint = strings.TrimPrefix(hint, "```")
	hint = strings.TrimSuffix(hint, "```")
	hint = strings.TrimSpace(hint)
	if idx := strings.Index(hint, "\n"); idx >= 0 && idx < 10 {
		first := hint[:idx]
		if !strings.Contains(first, " ") && !strings.Contains(first, "(") {
			hint = strings.TrimSpace(hint[idx+1:])
		}
	}

	return hint, nil
}

// regenerateWithHints regenerates test data, injecting code hints for failing rules.
// It creates a temporary Generator with a wrapped completer to avoid mutating shared state.
func (g *Generator) regenerateWithHints(ctx context.Context, input GenerateInput, hints map[string]string) error {
	if len(hints) == 0 {
		return nil
	}

	var hintBlock strings.Builder
	hintBlock.WriteString("\n\nIMPORTANT: The following rules FAILED in kantra tests. For each one, use the EXACT code snippet provided:\n")
	for ruleID, hint := range hints {
		fmt.Fprintf(&hintBlock, "- %s: %s\n", ruleID, hint)
	}

	wrapped := &hintCompleter{
		inner: g.completer,
		extra: hintBlock.String(),
	}

	// Create a new Generator with the wrapped completer instead of mutating g.completer.
	hintGen := New(wrapped, g.tmpl)
	_, err := hintGen.Generate(ctx, input)
	return err
}

// hintCompleter wraps a Completer and appends extra context to prompts.
type hintCompleter struct {
	inner llm.Completer
	extra string
}

func (h *hintCompleter) Complete(ctx context.Context, prompt string) (string, error) {
	return h.inner.Complete(ctx, prompt+h.extra)
}

// extractRulePattern extracts the primary pattern and provider from a rule's condition.
func extractRulePattern(r rules.Rule) (pattern, provider string) {
	c := r.When
	if c.GoReferenced != nil {
		return c.GoReferenced.Pattern, "go.referenced"
	}
	if c.JavaReferenced != nil {
		return c.JavaReferenced.Pattern, "java.referenced"
	}
	if c.NodejsReferenced != nil {
		return c.NodejsReferenced.Pattern, "nodejs.referenced"
	}
	if c.CSharpReferenced != nil {
		return c.CSharpReferenced.Pattern, "csharp.referenced"
	}
	if c.BuiltinFilecontent != nil {
		return c.BuiltinFilecontent.Pattern, "builtin.filecontent"
	}
	for _, entry := range c.Or {
		if p, prov := extractRulePattern(rules.Rule{When: entry.Condition}); p != "" {
			return p, prov
		}
	}
	for _, entry := range c.And {
		if p, prov := extractRulePattern(rules.Rule{When: entry.Condition}); p != "" {
			return p, prov
		}
	}
	return "", ""
}

