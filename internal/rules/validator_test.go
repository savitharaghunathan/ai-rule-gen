package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate_ValidRule(t *testing.T) {
	rules := []Rule{{
		RuleID:   "test-00010",
		Category: CategoryMandatory,
		Effort:   5,
		Labels:   []string{"konveyor.io/source=java-ee", "konveyor.io/target=quarkus"},
		Message:  "Migrate this API",
		When:     NewJavaReferenced("javax.ejb.Stateless", LocationAnnotation),
	}}

	result := Validate(rules)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
	if result.RuleCount != 1 {
		t.Errorf("rule_count: got %d, want 1", result.RuleCount)
	}
}

func TestValidate_MissingRuleID(t *testing.T) {
	rules := []Rule{{
		Message: "test",
		When:    NewJavaReferenced("foo", ""),
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "missing required field 'ruleID'")
}

func TestValidate_MissingMessageAndTag(t *testing.T) {
	rules := []Rule{{
		RuleID: "test-00010",
		When:   NewJavaReferenced("foo", ""),
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "either 'message' or 'tag' must be set")
}

func TestValidate_MissingWhen(t *testing.T) {
	rules := []Rule{{
		RuleID:  "test-00010",
		Message: "test",
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "missing required field 'when'")
}

func TestValidate_BadCategory(t *testing.T) {
	rules := []Rule{{
		RuleID:   "test-00010",
		Category: "invalid",
		Message:  "test",
		When:     NewJavaReferenced("foo", ""),
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "invalid category")
}

func TestValidate_BadRegex(t *testing.T) {
	rules := []Rule{{
		RuleID:  "test-00010",
		Message: "test",
		When:    NewBuiltinFilecontent("[invalid", ""),
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "invalid regex")
}

func TestValidate_DuplicateRuleIDs(t *testing.T) {
	rules := []Rule{
		{RuleID: "dup-00010", Message: "a", When: NewJavaReferenced("foo", "")},
		{RuleID: "dup-00010", Message: "b", When: NewJavaReferenced("bar", "")},
	}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "duplicate ruleID")
}

func TestValidate_BadJavaLocation(t *testing.T) {
	rules := []Rule{{
		RuleID:  "test-00010",
		Message: "test",
		When:    NewJavaReferenced("foo", "INVALID_LOCATION"),
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "invalid location")
}

func TestValidate_BadLabels(t *testing.T) {
	rules := []Rule{{
		RuleID:  "test-00010",
		Message: "test",
		Labels:  []string{"konveyor.io/source="},
		When:    NewJavaReferenced("foo", ""),
	}}

	result := Validate(rules)
	if len(result.Warnings) == 0 {
		t.Error("expected a warning about empty label value")
	}
}

func TestValidate_EffortOutOfRange(t *testing.T) {
	rules := []Rule{{
		RuleID:  "test-00010",
		Message: "test",
		Effort:  15,
		When:    NewJavaReferenced("foo", ""),
	}}

	result := Validate(rules)
	if len(result.Warnings) == 0 {
		t.Error("expected a warning about effort range")
	}
}

func TestValidate_TagOnly_Valid(t *testing.T) {
	rules := []Rule{{
		RuleID: "test-00010",
		Tag:    []string{"EJB"},
		When:   NewJavaReferenced("javax.ejb.*", ""),
	}}

	result := Validate(rules)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_RuleIDWithNewline(t *testing.T) {
	rules := []Rule{{
		RuleID:  "test\n00010",
		Message: "test",
		When:    NewJavaReferenced("foo", ""),
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid")
	}
	assertContains(t, result.Errors, "must not contain newlines or semicolons")
}

func TestValidate_OrCombinator(t *testing.T) {
	rules := []Rule{{
		RuleID:  "test-00010",
		Message: "test",
		When: NewOr(
			NewJavaReferenced("foo", ""),
			NewJavaReferenced("", ""), // missing pattern
		),
	}}

	result := Validate(rules)
	if result.Valid {
		t.Error("expected invalid due to missing pattern in or[1]")
	}
	assertContains(t, result.Errors, "or[1]")
}

func assertContains(t *testing.T, items []string, substr string) {
	t.Helper()
	for _, item := range items {
		if contains(item, substr) {
			return
		}
	}
	t.Errorf("expected item containing %q in %v", substr, items)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidateConsistency_AllMatched(t *testing.T) {
	rulesDir, testsDir := setupConsistencyDirs(t,
		[]Rule{
			{RuleID: "rule-00010", Message: "test", When: NewJavaReferenced("foo", "")},
			{RuleID: "rule-00020", Message: "test", When: NewJavaReferenced("bar", "")},
		},
		`tests:
  - ruleID: rule-00010
    testCases:
      - name: tc-1
  - ruleID: rule-00020
    testCases:
      - name: tc-1
`,
	)

	result, err := ValidateConsistency(rulesDir, testsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateConsistency_RuleWithoutTest(t *testing.T) {
	rulesDir, testsDir := setupConsistencyDirs(t,
		[]Rule{
			{RuleID: "rule-00010", Message: "test", When: NewJavaReferenced("foo", "")},
			{RuleID: "rule-00020", Message: "test", When: NewJavaReferenced("bar", "")},
		},
		`tests:
  - ruleID: rule-00010
    testCases:
      - name: tc-1
`,
	)

	result, err := ValidateConsistency(rulesDir, testsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid")
	}
	if len(result.RulesWithoutTests) != 1 || result.RulesWithoutTests[0] != "rule-00020" {
		t.Errorf("RulesWithoutTests = %v, want [rule-00020]", result.RulesWithoutTests)
	}
}

func TestValidateConsistency_TestWithoutRule(t *testing.T) {
	rulesDir, testsDir := setupConsistencyDirs(t,
		[]Rule{
			{RuleID: "rule-00010", Message: "test", When: NewJavaReferenced("foo", "")},
		},
		`tests:
  - ruleID: rule-00010
    testCases:
      - name: tc-1
  - ruleID: rule-00099
    testCases:
      - name: tc-1
`,
	)

	result, err := ValidateConsistency(rulesDir, testsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid")
	}
	if len(result.TestsWithoutRules) != 1 || result.TestsWithoutRules[0] != "rule-00099" {
		t.Errorf("TestsWithoutRules = %v, want [rule-00099]", result.TestsWithoutRules)
	}
}

func TestValidateConsistency_BothDirections(t *testing.T) {
	rulesDir, testsDir := setupConsistencyDirs(t,
		[]Rule{
			{RuleID: "rule-00010", Message: "test", When: NewJavaReferenced("foo", "")},
			{RuleID: "rule-00020", Message: "test", When: NewJavaReferenced("bar", "")},
		},
		`tests:
  - ruleID: rule-00010
    testCases:
      - name: tc-1
  - ruleID: rule-00099
    testCases:
      - name: tc-1
`,
	)

	result, err := ValidateConsistency(rulesDir, testsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid")
	}
	if len(result.RulesWithoutTests) != 1 {
		t.Errorf("RulesWithoutTests = %v, want 1 entry", result.RulesWithoutTests)
	}
	if len(result.TestsWithoutRules) != 1 {
		t.Errorf("TestsWithoutRules = %v, want 1 entry", result.TestsWithoutRules)
	}
}

func TestValidateConsistency_NoTestFiles(t *testing.T) {
	rulesDir, testsDir := setupConsistencyDirs(t,
		[]Rule{
			{RuleID: "rule-00010", Message: "test", When: NewJavaReferenced("foo", "")},
		},
		"", // no test file
	)

	result, err := ValidateConsistency(rulesDir, testsDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid — rule has no test")
	}
	if len(result.RulesWithoutTests) != 1 {
		t.Errorf("RulesWithoutTests = %v, want 1 entry", result.RulesWithoutTests)
	}
}

// setupConsistencyDirs creates temp rules and tests directories for testing.
func setupConsistencyDirs(t *testing.T, ruleList []Rule, testYAML string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	testsDir := filepath.Join(dir, "tests")
	os.MkdirAll(rulesDir, 0o755)
	os.MkdirAll(testsDir, 0o755)

	if len(ruleList) > 0 {
		if err := WriteRulesFile(filepath.Join(rulesDir, "web.yaml"), ruleList); err != nil {
			t.Fatalf("writing rules: %v", err)
		}
	}
	if testYAML != "" {
		os.WriteFile(filepath.Join(testsDir, "web.test.yaml"), []byte(testYAML), 0o644)
	}

	return rulesDir, testsDir
}
