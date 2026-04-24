package eval

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
)

func TestCheckRequiredFields_AllPresent(t *testing.T) {
	ctx := &EvalContext{
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{SourcePattern: "foo", Rationale: "r", Complexity: "low", Category: "mandatory"},
				{SourcePattern: "bar", Rationale: "r", Complexity: "high", Category: "optional"},
			},
		},
	}
	result := CheckRequiredFields(ctx)
	if !result.Passed {
		t.Errorf("expected pass, got fail: %s", result.Message)
	}
	if result.Agent != "rule-writer" {
		t.Errorf("Agent = %q, want rule-writer", result.Agent)
	}
}

func TestCheckRequiredFields_MissingFields(t *testing.T) {
	ctx := &EvalContext{
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{SourcePattern: "foo", Rationale: "r", Complexity: "low", Category: "mandatory"},
				{Rationale: "r", Complexity: "low", Category: "mandatory"},
				{SourcePattern: "baz", Complexity: "low", Category: "mandatory"},
			},
		},
	}
	result := CheckRequiredFields(ctx)
	if result.Passed {
		t.Error("expected fail for missing fields")
	}
	if result.Priority != "P0" {
		t.Errorf("Priority = %q, want P0", result.Priority)
	}
}

func TestCheckRequiredFields_NilPatterns(t *testing.T) {
	ctx := &EvalContext{}
	result := CheckRequiredFields(ctx)
	if result.Passed {
		t.Error("expected fail for nil patterns")
	}
}

func TestCheckRuleValidity_Valid(t *testing.T) {
	ctx := &EvalContext{
		Rules: []rules.Rule{
			{
				RuleID: "rule-001", Message: "test",
				When: rules.Condition{
					JavaReferenced: &rules.JavaReferenced{Pattern: "com.example.Foo", Location: "IMPORT"},
				},
			},
		},
	}
	result := CheckRuleValidity(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
	if result.Agent != "rule-writer" {
		t.Errorf("Agent = %q, want rule-writer", result.Agent)
	}
}

func TestCheckRuleValidity_Invalid(t *testing.T) {
	ctx := &EvalContext{
		Rules: []rules.Rule{
			{RuleID: "", Message: "no id"},
		},
	}
	result := CheckRuleValidity(ctx)
	if result.Passed {
		t.Error("expected fail for missing ruleID")
	}
}

func TestCheckRuleValidity_NilRules(t *testing.T) {
	ctx := &EvalContext{}
	result := CheckRuleValidity(ctx)
	if result.Passed {
		t.Error("expected fail for nil rules")
	}
}

func TestConditionType(t *testing.T) {
	tests := []struct {
		name string
		cond rules.Condition
		want string
	}{
		{"java.referenced", rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "x"}}, "java.referenced"},
		{"java.dependency", rules.Condition{JavaDependency: &rules.Dependency{Name: "x"}}, "java.dependency"},
		{"go.referenced", rules.Condition{GoReferenced: &rules.GoReferenced{Pattern: "x"}}, "go.referenced"},
		{"builtin.filecontent", rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{Pattern: "x"}}, "builtin.filecontent"},
		{"builtin.xml", rules.Condition{BuiltinXML: &rules.BuiltinXML{XPath: "//x"}}, "builtin.xml"},
		{"or", rules.Condition{Or: []rules.ConditionEntry{{}}}, "or"},
		{"empty", rules.Condition{}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConditionType(tt.cond)
			if got != tt.want {
				t.Errorf("ConditionType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckConditionTypes_Match(t *testing.T) {
	ctx := &EvalContext{
		Golden: &GoldenSet{
			Patterns: []GoldenPattern{
				{ID: "dep-1", DependencyName: "org.example.foo", ConditionType: "java.dependency"},
				{ID: "ref-1", SourceFQN: "com.example.Bar", ConditionType: "java.referenced"},
			},
		},
		Rules: []rules.Rule{
			{RuleID: "r1", When: rules.Condition{JavaDependency: &rules.Dependency{Name: "org.example.foo"}}},
			{RuleID: "r2", When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "com.example.Bar"}}},
		},
	}
	result := CheckConditionTypes(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
}

func TestCheckConditionTypes_Mismatch(t *testing.T) {
	ctx := &EvalContext{
		Golden: &GoldenSet{
			Patterns: []GoldenPattern{
				{ID: "dep-1", DependencyName: "org.example.foo", ConditionType: "java.dependency"},
			},
		},
		Rules: []rules.Rule{
			{RuleID: "r1", When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.example.foo"}}},
		},
	}
	result := CheckConditionTypes(ctx)
	if result.Passed {
		t.Error("expected fail for condition type mismatch")
	}
}

func TestCheckPassRate_Above(t *testing.T) {
	ctx := &EvalContext{
		Golden: &GoldenSet{Thresholds: Thresholds{PassRatePostFix: 95.0}},
		Report: &workspace.Report{PassRate: 100.0, TestsPassed: 62, TestsFailed: 0},
	}
	result := CheckPassRate(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
	if result.Agent != "pipeline" {
		t.Errorf("Agent = %q, want pipeline", result.Agent)
	}
}

func TestCheckPassRate_Below(t *testing.T) {
	ctx := &EvalContext{
		Golden: &GoldenSet{Thresholds: Thresholds{PassRatePostFix: 95.0}},
		Report: &workspace.Report{PassRate: 80.0, TestsPassed: 40, TestsFailed: 10},
	}
	result := CheckPassRate(ctx)
	if result.Passed {
		t.Error("expected fail for low pass rate")
	}
}

func TestCheckPassRate_DefaultThreshold(t *testing.T) {
	ctx := &EvalContext{
		Report: &workspace.Report{PassRate: 96.0},
	}
	result := CheckPassRate(ctx)
	if !result.Passed {
		t.Errorf("expected pass with default threshold: %s", result.Message)
	}
}

func TestCheckDeduplication_NoDupes(t *testing.T) {
	ctx := &EvalContext{
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{SourceFQN: "com.example.Foo"},
				{SourceFQN: "com.example.Bar"},
				{DependencyName: "org.example.dep"},
			},
		},
	}
	result := CheckDeduplication(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
}

func TestCheckDeduplication_DuplicateFQN(t *testing.T) {
	ctx := &EvalContext{
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{SourceFQN: "com.example.Foo", SourcePattern: "a"},
				{SourceFQN: "com.example.Foo", SourcePattern: "b"},
			},
		},
	}
	result := CheckDeduplication(ctx)
	if result.Passed {
		t.Error("expected fail for duplicate source_fqn")
	}
}

func TestCheckDeduplication_DuplicateDependency(t *testing.T) {
	ctx := &EvalContext{
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{DependencyName: "org.example.foo", SourcePattern: "a"},
				{DependencyName: "org.example.foo", SourcePattern: "b"},
			},
		},
	}
	result := CheckDeduplication(ctx)
	if result.Passed {
		t.Error("expected fail for duplicate dependency_name")
	}
}

func TestCheckKnownPatterns_AllFound(t *testing.T) {
	ctx := &EvalContext{
		Golden: &GoldenSet{
			Patterns: []GoldenPattern{
				{ID: "dep-1", DependencyName: "org.example.foo"},
				{ID: "ref-1", SourceFQN: "com.example.Bar"},
			},
		},
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{DependencyName: "org.example.foo", SourcePattern: "foo"},
				{SourceFQN: "com.example.Bar", SourcePattern: "bar"},
				{SourceFQN: "com.example.Extra", SourcePattern: "extra"},
			},
		},
	}
	result := CheckKnownPatterns(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
}

func TestCheckKnownPatterns_SomeMissing(t *testing.T) {
	ctx := &EvalContext{
		Golden: &GoldenSet{
			Patterns: []GoldenPattern{
				{ID: "dep-1", DependencyName: "org.example.foo"},
				{ID: "ref-1", SourceFQN: "com.example.Bar"},
			},
		},
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{DependencyName: "org.example.foo", SourcePattern: "foo"},
			},
		},
	}
	result := CheckKnownPatterns(ctx)
	if result.Passed {
		t.Error("expected fail for missing golden pattern")
	}
	details, ok := result.Details.([]string)
	if !ok || len(details) != 1 || details[0] != "ref-1" {
		t.Errorf("expected [ref-1] in details, got %v", result.Details)
	}
}

func TestRunAll_RuleWriterOnly(t *testing.T) {
	ctx := &EvalContext{
		Golden: &GoldenSet{
			Patterns: []GoldenPattern{
				{ID: "dep-1", DependencyName: "org.example.foo", ConditionType: "java.dependency"},
			},
			Thresholds: Thresholds{PassRatePostFix: 95.0},
		},
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{SourcePattern: "foo", DependencyName: "org.example.foo", Rationale: "r", Complexity: "low", Category: "mandatory"},
			},
		},
		Rules: []rules.Rule{
			{
				RuleID: "r1", Message: "test",
				When: rules.Condition{JavaDependency: &rules.Dependency{Name: "org.example.foo"}},
			},
		},
	}
	report := RunAll(ctx)
	rw, ok := report.Agents["rule-writer"]
	if !ok {
		t.Fatal("expected rule-writer agent in report")
	}
	if rw.Summary.Failed != 0 {
		t.Errorf("expected 0 rule-writer failures, got %d", rw.Summary.Failed)
		for _, c := range rw.Checks {
			if !c.Passed {
				t.Errorf("  FAIL: %s — %s", c.ID, c.Message)
			}
		}
	}
	if rw.Summary.Total != 5 {
		t.Errorf("expected 5 rule-writer checks, got %d", rw.Summary.Total)
	}
	if _, ok := report.Agents["test-generator"]; ok {
		t.Error("test-generator should not appear without PreFixReport")
	}
	if _, ok := report.Agents["validator"]; ok {
		t.Error("validator should not appear without RulesSnapshot")
	}
	if _, ok := report.Agents["pipeline"]; ok {
		t.Error("pipeline should not appear without Report")
	}
}

func TestRunAll_AllAgents(t *testing.T) {
	baseRules := []rules.Rule{
		{
			RuleID: "r1", Message: "test",
			When: rules.Condition{JavaDependency: &rules.Dependency{Name: "org.example.foo"}},
		},
	}
	ctx := &EvalContext{
		Golden: &GoldenSet{
			Patterns: []GoldenPattern{
				{ID: "dep-1", DependencyName: "org.example.foo", ConditionType: "java.dependency"},
			},
			Thresholds: Thresholds{PassRatePostFix: 95.0, PreFixPassRate: 80.0},
		},
		Patterns: &rules.ExtractOutput{
			Patterns: []rules.MigrationPattern{
				{SourcePattern: "foo", DependencyName: "org.example.foo", Rationale: "r", Complexity: "low", Category: "mandatory"},
			},
		},
		Rules:         baseRules,
		RulesSnapshot: baseRules,
		PreFixReport:  &workspace.Report{PassRate: 90.0, TestsPassed: 9, TestsFailed: 1, FailedRules: []string{"r2"}},
		Report:        &workspace.Report{PassRate: 100.0, TestsPassed: 10, TestsFailed: 0},
	}
	report := RunAll(ctx)

	expectedAgents := []string{"rule-writer", "test-generator", "validator", "pipeline"}
	for _, agent := range expectedAgents {
		if _, ok := report.Agents[agent]; !ok {
			t.Errorf("expected %s agent in report", agent)
		}
	}
	if report.Summary.Failed != 0 {
		t.Errorf("expected 0 total failures, got %d", report.Summary.Failed)
		for agent, ae := range report.Agents {
			for _, c := range ae.Checks {
				if !c.Passed {
					t.Errorf("  FAIL [%s]: %s — %s", agent, c.ID, c.Message)
				}
			}
		}
	}
}

func TestRunAll_MinimalContext(t *testing.T) {
	ctx := &EvalContext{}
	report := RunAll(ctx)
	if len(report.Agents) != 0 {
		t.Errorf("expected empty agents map with no data, got %d agents", len(report.Agents))
	}
	if report.Summary.Total != 0 {
		t.Errorf("expected 0 total checks, got %d", report.Summary.Total)
	}
}

func TestCheckPreFixPassRate_Above(t *testing.T) {
	ctx := &EvalContext{
		Golden:       &GoldenSet{Thresholds: Thresholds{PreFixPassRate: 80.0}},
		PreFixReport: &workspace.Report{PassRate: 90.0, TestsPassed: 9, TestsFailed: 1},
	}
	result := CheckPreFixPassRate(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
	if result.Agent != "test-generator" {
		t.Errorf("Agent = %q, want test-generator", result.Agent)
	}
}

func TestCheckPreFixPassRate_Below(t *testing.T) {
	ctx := &EvalContext{
		Golden:       &GoldenSet{Thresholds: Thresholds{PreFixPassRate: 80.0}},
		PreFixReport: &workspace.Report{PassRate: 50.0, TestsPassed: 5, TestsFailed: 5},
	}
	result := CheckPreFixPassRate(ctx)
	if result.Passed {
		t.Error("expected fail for low pre-fix pass rate")
	}
}

func TestCheckPreFixPassRate_DefaultThreshold(t *testing.T) {
	ctx := &EvalContext{
		PreFixReport: &workspace.Report{PassRate: 85.0},
	}
	result := CheckPreFixPassRate(ctx)
	if !result.Passed {
		t.Errorf("expected pass with default 80%% threshold: %s", result.Message)
	}
}

func TestCheckPreFixPassRate_Nil(t *testing.T) {
	ctx := &EvalContext{}
	result := CheckPreFixPassRate(ctx)
	if result.Passed {
		t.Error("expected fail for nil pre-fix report")
	}
}

func TestCheckRuleIntegrity_Unchanged(t *testing.T) {
	r := []rules.Rule{
		{
			RuleID: "r1", Message: "test",
			When: rules.Condition{JavaDependency: &rules.Dependency{Name: "org.example.foo"}},
		},
	}
	ctx := &EvalContext{Rules: r, RulesSnapshot: r}
	result := CheckRuleIntegrity(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
	if result.Agent != "validator" {
		t.Errorf("Agent = %q, want validator", result.Agent)
	}
}

func TestCheckRuleIntegrity_Changed(t *testing.T) {
	snapshot := []rules.Rule{
		{
			RuleID: "r1", Message: "test",
			When: rules.Condition{JavaDependency: &rules.Dependency{Name: "org.example.foo"}},
		},
	}
	modified := []rules.Rule{
		{
			RuleID: "r1", Message: "test",
			When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "org.example.foo"}},
		},
	}
	ctx := &EvalContext{Rules: modified, RulesSnapshot: snapshot}
	result := CheckRuleIntegrity(ctx)
	if result.Passed {
		t.Error("expected fail for modified rule condition")
	}
	details, ok := result.Details.([]string)
	if !ok || len(details) != 1 || details[0] != "r1" {
		t.Errorf("expected [r1] in details, got %v", result.Details)
	}
}

func TestCheckRuleIntegrity_Nil(t *testing.T) {
	ctx := &EvalContext{}
	result := CheckRuleIntegrity(ctx)
	if result.Passed {
		t.Error("expected fail for nil snapshot")
	}
}

func TestCheckFixEffectiveness_Improved(t *testing.T) {
	ctx := &EvalContext{
		PreFixReport: &workspace.Report{PassRate: 80.0, TestsPassed: 8, TestsFailed: 2},
		Report:       &workspace.Report{PassRate: 100.0, TestsPassed: 10, TestsFailed: 0},
	}
	result := CheckFixEffectiveness(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
	if result.Agent != "validator" {
		t.Errorf("Agent = %q, want validator", result.Agent)
	}
}

func TestCheckFixEffectiveness_NoImprovement(t *testing.T) {
	ctx := &EvalContext{
		PreFixReport: &workspace.Report{PassRate: 80.0, TestsPassed: 8, TestsFailed: 2},
		Report:       &workspace.Report{PassRate: 80.0, TestsPassed: 8, TestsFailed: 2},
	}
	result := CheckFixEffectiveness(ctx)
	if result.Passed {
		t.Error("expected fail when no failures were fixed")
	}
}

func TestCheckFixEffectiveness_NoFailures(t *testing.T) {
	ctx := &EvalContext{
		PreFixReport: &workspace.Report{PassRate: 100.0, TestsPassed: 10, TestsFailed: 0},
		Report:       &workspace.Report{PassRate: 100.0, TestsPassed: 10, TestsFailed: 0},
	}
	result := CheckFixEffectiveness(ctx)
	if !result.Passed {
		t.Errorf("expected pass when no failures to fix: %s", result.Message)
	}
}

func TestCheckNoRegressions_Clean(t *testing.T) {
	ctx := &EvalContext{
		PreFixReport: &workspace.Report{FailedRules: []string{"r1", "r2"}},
		Report:       &workspace.Report{FailedRules: []string{"r1"}},
	}
	result := CheckNoRegressions(ctx)
	if !result.Passed {
		t.Errorf("expected pass: %s", result.Message)
	}
	if result.Agent != "validator" {
		t.Errorf("Agent = %q, want validator", result.Agent)
	}
}

func TestCheckNoRegressions_Regression(t *testing.T) {
	ctx := &EvalContext{
		PreFixReport: &workspace.Report{FailedRules: []string{"r1"}},
		Report:       &workspace.Report{FailedRules: []string{"r3"}},
	}
	result := CheckNoRegressions(ctx)
	if result.Passed {
		t.Error("expected fail for regression")
	}
	details, ok := result.Details.([]string)
	if !ok || len(details) != 1 || details[0] != "r3" {
		t.Errorf("expected [r3] in details, got %v", result.Details)
	}
}

func TestCheckNoRegressions_AllFixed(t *testing.T) {
	ctx := &EvalContext{
		PreFixReport: &workspace.Report{FailedRules: []string{"r1", "r2"}},
		Report:       &workspace.Report{},
	}
	result := CheckNoRegressions(ctx)
	if !result.Passed {
		t.Errorf("expected pass when all fixed: %s", result.Message)
	}
}
