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
