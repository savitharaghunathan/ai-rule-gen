package eval

import (
	"fmt"
	"os"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// RunEval executes the deterministic eval checks.
func RunEval(cfg Config) (*EvalResult, error) {
	ruleList, err := rules.ReadRulesDir(cfg.RulesDir)
	if err != nil {
		return nil, fmt.Errorf("loading rules: %w", err)
	}

	if len(ruleList) == 0 {
		return nil, fmt.Errorf("no rules found in %s", cfg.RulesDir)
	}

	quality, details := ScoreAll(ruleList)

	result := &EvalResult{
		RuleCount:   len(ruleList),
		Quality:     quality,
		RuleDetails: details,
	}

	if cfg.AppDir != "" {
		outputDir, err := os.MkdirTemp("", "eval-kantra-output-*")
		if err != nil {
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
		defer os.RemoveAll(outputDir)

		cov, err := RunKantraAnalyze(cfg.RulesDir, cfg.AppDir, outputDir)
		if err != nil {
			return nil, fmt.Errorf("app coverage: %w", err)
		}
		FillNotFired(cov, ruleList)

		if len(cov.NotFired) > 0 {
			cov.Unmatched = CrossRefNotFired(ruleList, cov.NotFired, cfg.AppDir)
		}

		notInAppCount := 0
		for _, u := range cov.Unmatched {
			if !u.InApp {
				notInAppCount++
			}
		}
		cov.EffectiveTotal = cov.TotalRules - notInAppCount
		cov.EffectiveFired = cov.RulesFired
		if cov.EffectiveTotal > 0 {
			cov.EffectivePct = 100 * cov.EffectiveFired / cov.EffectiveTotal
		}

		for i, d := range result.RuleDetails {
			if v, ok := cov.Violations[d.RuleID]; ok {
				result.RuleDetails[i].AppIncidents = v.Incidents
				result.RuleDetails[i].AppFiles = v.Files
			}
		}

		result.AppCoverage = cov
	}

	return result, nil
}

// PrintReport writes a human-readable summary to stderr.
func PrintReport(r *EvalResult) {
	fmt.Fprintf(os.Stderr, "======================================================================\n")
	fmt.Fprintf(os.Stderr, "EVAL REPORT\n")
	fmt.Fprintf(os.Stderr, "======================================================================\n")

	fmt.Fprintf(os.Stderr, "\n## Rules: %d\n", r.RuleCount)

	q := r.Quality
	fmt.Fprintf(os.Stderr, "\n## Quality (avg %.1f/%d)\n", q.AvgScore, q.MaxScore)
	fmt.Fprintf(os.Stderr, "   Messages:           %d/%d\n", q.HasMessage, q.TotalRules)
	fmt.Fprintf(os.Stderr, "   Links:              %d/%d\n", q.HasLinks, q.TotalRules)
	fmt.Fprintf(os.Stderr, "   Effort rating:      %d/%d\n", q.HasEffort, q.TotalRules)
	fmt.Fprintf(os.Stderr, "   Before/after:       %d/%d\n", q.HasBeforeAfter, q.TotalRules)

	for _, d := range r.RuleDetails {
		if len(d.Missing) > 0 {
			fmt.Fprintf(os.Stderr, "   %s: missing %v\n", d.RuleID, d.Missing)
		}
	}

	if r.AppCoverage != nil {
		c := r.AppCoverage
		pct := 0
		if c.TotalRules > 0 {
			pct = 100 * c.RulesFired / c.TotalRules
		}
		fmt.Fprintf(os.Stderr, "\n## App Coverage\n")
		fmt.Fprintf(os.Stderr, "   Rules fired:      %d/%d (%d%%)\n", c.RulesFired, c.TotalRules, pct)
		fmt.Fprintf(os.Stderr, "   Effective match:  %d/%d (%d%%)  — excludes rules where API is absent from app\n", c.EffectiveFired, c.EffectiveTotal, c.EffectivePct)
		fmt.Fprintf(os.Stderr, "   Incidents:        %d\n", c.TotalIncidents)

		if len(c.Unmatched) > 0 {
			var inApp, notInApp []UnmatchedRule
			for _, u := range c.Unmatched {
				if u.InApp {
					inApp = append(inApp, u)
				} else {
					notInApp = append(notInApp, u)
				}
			}

			if len(inApp) > 0 {
				fmt.Fprintf(os.Stderr, "\n   In app but unmatched (%d rules):\n", len(inApp))
				for _, u := range inApp {
					files := strings.Join(u.AppFiles, ", ")
					fmt.Fprintf(os.Stderr, "     - %s (%s) → %s\n", u.RuleID, u.Pattern, files)
				}
			}

			if len(notInApp) > 0 {
				fmt.Fprintf(os.Stderr, "\n   Not in app (%d rules):\n", len(notInApp))
				for _, u := range notInApp {
					fmt.Fprintf(os.Stderr, "     - %s (%s)\n", u.RuleID, u.Pattern)
				}
			}
		} else if len(c.NotFired) > 0 {
			fmt.Fprintf(os.Stderr, "\n   Not fired:\n")
			for _, id := range c.NotFired {
				fmt.Fprintf(os.Stderr, "     - %s\n", id)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n======================================================================\n")
}
