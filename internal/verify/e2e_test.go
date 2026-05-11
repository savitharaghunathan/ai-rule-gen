package verify_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/construct"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/verify"
)

func TestEndToEnd_VerifyAndStamp(t *testing.T) {
	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "cache")
	rulesDir := filepath.Join(dir, "rules")

	extract := &rules.ExtractOutput{
		Source:   "lib-v1",
		Target:   "lib-v2",
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
				// No SourceArtifact — should be skipped
			},
			{
				SourcePattern:  "B",
				DependencyName: "org.example.dep",
				Rationale:      "dep removed",
				Complexity:     "low",
				Category:       "mandatory",
				ProviderType:   "java",
				// Dependency — auto-verified
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

	// Step 4: Stamp verification labels
	if err := rules.StampVerificationResults(rulesDir, verifiedIDs, notFoundIDs); err != nil {
		t.Fatalf("StampVerificationResults: %v", err)
	}

	// Step 5: Read back and check labels
	allRules, err := rules.ReadRulesDir(rulesDir)
	if err != nil {
		t.Fatalf("ReadRulesDir: %v", err)
	}

	for _, r := range allRules {
		ruleID := r.RuleID
		for _, l := range r.Labels {
			if strings.HasPrefix(l, "konveyor.io/source-verified=") {
				// The dependency rule should be verified
				if ruleID == constructResult.PatternRuleMap[1] {
					if l != "konveyor.io/source-verified=true" {
						t.Errorf("dependency rule %s: label = %q, want source-verified=true", ruleID, l)
					}
				}
			}
		}
	}
}
