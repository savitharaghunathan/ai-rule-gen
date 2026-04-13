package testgen

import (
	"testing"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func TestGroupRulesForTestgen_SmallSet(t *testing.T) {
	ruleList := []rules.Rule{
		{RuleID: "r-00010", When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateless"}}},
		{RuleID: "r-00020", When: rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.ejb.Stateful"}}},
	}

	groups := groupRulesForTestgen(ruleList, "rules.yaml")
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if _, ok := groups["rules"]; !ok {
		t.Error("expected group named 'rules'")
	}
	if len(groups["rules"]) != 2 {
		t.Errorf("expected 2 rules, got %d", len(groups["rules"]))
	}
}

func TestGroupRulesForTestgen_SplitByConditionType(t *testing.T) {
	ruleList := make([]rules.Rule, 0, 12)
	// 7 java.referenced rules
	for i := range 7 {
		ruleList = append(ruleList, rules.Rule{
			RuleID: ruleID(i),
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.some.Class"}},
		})
	}
	// 5 java.dependency rules
	for i := range 5 {
		ruleList = append(ruleList, rules.Rule{
			RuleID: ruleID(i + 10),
			When:   rules.Condition{JavaDependency: &rules.Dependency{Name: "org.example.dep"}},
		})
	}

	groups := groupRulesForTestgen(ruleList, "rules.yaml")

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d: %v", len(groups), groupNames(groups))
	}

	refGroup, ok := groups["rules-java-referenced"]
	if !ok {
		t.Fatal("expected group 'rules-java-referenced'")
	}
	if len(refGroup) != 7 {
		t.Errorf("java-referenced group: expected 7 rules, got %d", len(refGroup))
	}

	depGroup, ok := groups["rules-java-dependency"]
	if !ok {
		t.Fatal("expected group 'rules-java-dependency'")
	}
	if len(depGroup) != 5 {
		t.Errorf("java-dependency group: expected 5 rules, got %d", len(depGroup))
	}
}

func TestGroupRulesForTestgen_SplitBySizeWhenSameType(t *testing.T) {
	ruleList := make([]rules.Rule, 0, 20)
	for i := range 20 {
		ruleList = append(ruleList, rules.Rule{
			RuleID: ruleID(i),
			When:   rules.Condition{JavaReferenced: &rules.JavaReferenced{Pattern: "javax.some.Class"}},
		})
	}

	groups := groupRulesForTestgen(ruleList, "rules.yaml")

	// 20 rules / 8 per group = 3 groups
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %v", len(groups), groupNames(groups))
	}

	total := 0
	for _, g := range groups {
		total += len(g)
	}
	if total != 20 {
		t.Errorf("total rules = %d, want 20", total)
	}
}

func TestRuleConditionType(t *testing.T) {
	tests := []struct {
		name string
		rule rules.Rule
		want string
	}{
		{"java.referenced", rules.Rule{When: rules.Condition{JavaReferenced: &rules.JavaReferenced{}}}, "java-referenced"},
		{"java.dependency", rules.Rule{When: rules.Condition{JavaDependency: &rules.Dependency{}}}, "java-dependency"},
		{"go.referenced", rules.Rule{When: rules.Condition{GoReferenced: &rules.GoReferenced{}}}, "go-referenced"},
		{"builtin.filecontent", rules.Rule{When: rules.Condition{BuiltinFilecontent: &rules.BuiltinFilecontent{}}}, "builtin-filecontent"},
		{"or combinator", rules.Rule{When: rules.Condition{Or: []rules.ConditionEntry{{}}}}, "combinator"},
		{"empty", rules.Rule{When: rules.Condition{}}, "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ruleConditionType(tt.rule)
			if got != tt.want {
				t.Errorf("ruleConditionType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitBySize(t *testing.T) {
	ruleList := make([]rules.Rule, 10)
	for i := range ruleList {
		ruleList[i] = rules.Rule{RuleID: ruleID(i)}
	}

	groups := splitBySize(ruleList, "test", 4)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if len(groups["test-part1"]) != 4 {
		t.Errorf("part1: expected 4, got %d", len(groups["test-part1"]))
	}
	if len(groups["test-part2"]) != 4 {
		t.Errorf("part2: expected 4, got %d", len(groups["test-part2"]))
	}
	if len(groups["test-part3"]) != 2 {
		t.Errorf("part3: expected 2, got %d", len(groups["test-part3"]))
	}
}

func ruleID(i int) string {
	return "r-" + string(rune('a'+i))
}

func groupNames(groups map[string][]rules.Rule) []string {
	names := make([]string, 0, len(groups))
	for n := range groups {
		names = append(names, n)
	}
	return names
}
