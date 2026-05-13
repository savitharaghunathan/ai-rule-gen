package verify_test

import (
	"path/filepath"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/construct"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/verify"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
)

func TestEndToEnd_VerifyToReport(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	rulesDir := filepath.Join(dir, "rules")

	extract := &rules.ExtractOutput{
		Sources:  []string{"lib-v1"},
		Targets:  []string{"lib-v2"},
		Language: "java",
		Patterns: []rules.MigrationPattern{
			{
				SourcePattern: "A",
				SourceFQN:     "com.example.RealClass",
				Rationale:     "class moved",
				Complexity:    "low",
				Category:      "mandatory",
				ProviderType:  "java",
				LocationType:  "IMPORT",
			},
			{
				SourcePattern:  "B",
				DependencyName: "org.example.dep",
				Rationale:      "dep removed",
				Complexity:     "low",
				Category:       "mandatory",
				ProviderType:   "java",
			},
		},
	}

	// Step 1: Verify patterns
	results, err := verify.Run(extract, cacheDir)
	if err != nil {
		t.Fatalf("verify.Run: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}
	if results[0].Status != verify.StatusSkipped {
		t.Errorf("pattern 0 status = %q, want skipped", results[0].Status)
	}
	if results[1].Status != verify.StatusVerified {
		t.Errorf("pattern 1 status = %q, want verified", results[1].Status)
	}

	// Step 2: Construct rules
	constructResult, err := construct.Run(extract, rulesDir)
	if err != nil {
		t.Fatalf("construct.Run: %v", err)
	}

	// Step 3: Map verification results to rule IDs
	var verifiedIDs, notFoundIDs []string
	for _, r := range results {
		ruleID, ok := constructResult.PatternRuleMap[r.PatternIndex]
		if !ok {
			continue
		}
		switch r.Status {
		case verify.StatusVerified:
			verifiedIDs = append(verifiedIDs, ruleID)
		case verify.StatusNotFound:
			notFoundIDs = append(notFoundIDs, ruleID)
		}
	}

	// Step 4: Build report with per-rule status
	report := workspace.BuildReport([]string{"lib-v1"}, []string{"lib-v2"}, 2, 0, 0, 0, nil, nil, nil, verifiedIDs, notFoundIDs)

	if len(report.Rules) != 1 {
		t.Fatalf("report rules count = %d, want 1 (only verified/not-found rules included)", len(report.Rules))
	}

	depRuleID := constructResult.PatternRuleMap[1]
	found := false
	for _, rs := range report.Rules {
		if rs.RuleID == depRuleID {
			found = true
			if rs.SourceVerified != "true" {
				t.Errorf("dependency rule %s: source_verified = %q, want true", depRuleID, rs.SourceVerified)
			}
			if rs.TestStatus != "untested" {
				t.Errorf("dependency rule %s: test_status = %q, want untested", depRuleID, rs.TestStatus)
			}
		}
	}
	if !found {
		t.Errorf("dependency rule %s not found in report", depRuleID)
	}

	// Verify rules don't have pipeline metadata labels
	allRules, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("ReadRulesDir: %v", err)
	}
	for _, r := range allRules {
		for _, l := range r.Labels {
			if l == "konveyor.io/test-result=untested" || l == "konveyor.io/review=unreviewed" {
				t.Errorf("rule %s has pipeline metadata label %q — should only be in report", r.RuleID, l)
			}
		}
	}
}
