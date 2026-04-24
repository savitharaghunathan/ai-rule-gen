package eval

import (
	"fmt"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
)

// CheckResult holds the outcome of a single eval check.
type CheckResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Priority string `json:"priority"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	Details  any    `json:"details,omitempty"`
}

// EvalSummary holds aggregate pass/fail counts.
type EvalSummary struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	Failed   int     `json:"failed"`
	PassRate float64 `json:"pass_rate"`
}

// EvalReport is the top-level eval output.
type EvalReport struct {
	Timestamp string        `json:"timestamp"`
	GoldenSet string        `json:"golden_set"`
	Checks    []CheckResult `json:"checks"`
	Summary   EvalSummary   `json:"summary"`
}

// EvalContext holds all inputs needed by grader functions.
type EvalContext struct {
	Golden   *GoldenSet
	Patterns *rules.ExtractOutput
	Rules    []rules.Rule
	Report   *workspace.Report
}

// CheckRequiredFields (p0-004) checks each pattern has source_pattern, rationale,
// complexity, and category.
func CheckRequiredFields(ctx *EvalContext) CheckResult {
	if ctx.Patterns == nil {
		return CheckResult{
			ID: "p0-004", Name: "Required fields in patterns.json",
			Priority: "P0", Passed: false,
			Message: "No patterns.json loaded",
		}
	}
	var missing []string
	for i, p := range ctx.Patterns.Patterns {
		var fields []string
		if p.SourcePattern == "" {
			fields = append(fields, "source_pattern")
		}
		if p.Rationale == "" {
			fields = append(fields, "rationale")
		}
		if p.Complexity == "" {
			fields = append(fields, "complexity")
		}
		if p.Category == "" {
			fields = append(fields, "category")
		}
		if len(fields) > 0 {
			missing = append(missing, fmt.Sprintf("pattern[%d]: missing %s", i, strings.Join(fields, ", ")))
		}
	}
	if len(missing) > 0 {
		return CheckResult{
			ID: "p0-004", Name: "Required fields in patterns.json",
			Priority: "P0", Passed: false,
			Message:  fmt.Sprintf("%d patterns missing required fields", len(missing)),
			Details:  missing,
		}
	}
	return CheckResult{
		ID: "p0-004", Name: "Required fields in patterns.json",
		Priority: "P0", Passed: true,
		Message: fmt.Sprintf("All %d patterns have required fields", len(ctx.Patterns.Patterns)),
	}
}

// CheckRuleValidity (p0-001) validates rule YAML using rules.Validate.
func CheckRuleValidity(ctx *EvalContext) CheckResult {
	if len(ctx.Rules) == 0 {
		return CheckResult{
			ID: "p0-001", Name: "Rule YAML validity",
			Priority: "P0", Passed: false,
			Message: "No rules loaded",
		}
	}
	result := rules.Validate(ctx.Rules)
	if result.Valid {
		return CheckResult{
			ID: "p0-001", Name: "Rule YAML validity",
			Priority: "P0", Passed: true,
			Message: fmt.Sprintf("All %d rules are valid", result.RuleCount),
		}
	}
	return CheckResult{
		ID: "p0-001", Name: "Rule YAML validity",
		Priority: "P0", Passed: false,
		Message:  fmt.Sprintf("%d validation errors", len(result.Errors)),
		Details:  result.Errors,
	}
}

// ConditionType returns the provider condition type string for a Condition.
func ConditionType(c rules.Condition) string {
	switch {
	case c.JavaReferenced != nil:
		return "java.referenced"
	case c.JavaDependency != nil:
		return "java.dependency"
	case c.GoReferenced != nil:
		return "go.referenced"
	case c.GoDependency != nil:
		return "go.dependency"
	case c.NodejsReferenced != nil:
		return "nodejs.referenced"
	case c.CSharpReferenced != nil:
		return "csharp.referenced"
	case c.BuiltinFilecontent != nil:
		return "builtin.filecontent"
	case c.BuiltinFile != nil:
		return "builtin.file"
	case c.BuiltinXML != nil:
		return "builtin.xml"
	case c.BuiltinJSON != nil:
		return "builtin.json"
	case c.BuiltinXMLPublicID != nil:
		return "builtin.xmlPublicID"
	case len(c.Or) > 0:
		return "or"
	case len(c.And) > 0:
		return "and"
	default:
		return "unknown"
	}
}

// CheckConditionTypes (p1-006) verifies each golden pattern's expected condition type
// matches the actual condition type in the generated rules.
func CheckConditionTypes(ctx *EvalContext) CheckResult {
	if ctx.Golden == nil || len(ctx.Rules) == 0 {
		return CheckResult{
			ID: "p1-006", Name: "Correct condition types",
			Priority: "P1", Passed: false,
			Message: "Missing golden set or rules",
		}
	}
	var mismatches []string
	for _, gp := range ctx.Golden.Patterns {
		r := findRuleByGolden(ctx.Rules, gp)
		if r == nil {
			continue
		}
		actual := ConditionType(r.When)
		if actual != gp.ConditionType {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected %s, got %s", gp.ID, gp.ConditionType, actual))
		}
	}
	if len(mismatches) > 0 {
		return CheckResult{
			ID: "p1-006", Name: "Correct condition types",
			Priority: "P1", Passed: false,
			Message:  fmt.Sprintf("%d condition type mismatches", len(mismatches)),
			Details:  mismatches,
		}
	}
	return CheckResult{
		ID: "p1-006", Name: "Correct condition types",
		Priority: "P1", Passed: true,
		Message: "All golden patterns have correct condition types",
	}
}

// findRuleByGolden locates the rule matching a golden pattern by source_fqn or
// dependency_name, including conditions inside or-combinators.
func findRuleByGolden(ruleList []rules.Rule, gp GoldenPattern) *rules.Rule {
	for i, r := range ruleList {
		c := r.When
		if gp.SourceFQN != "" {
			if c.JavaReferenced != nil && c.JavaReferenced.Pattern == gp.SourceFQN {
				return &ruleList[i]
			}
			if c.GoReferenced != nil && c.GoReferenced.Pattern == gp.SourceFQN {
				return &ruleList[i]
			}
			if c.NodejsReferenced != nil && c.NodejsReferenced.Pattern == gp.SourceFQN {
				return &ruleList[i]
			}
			if c.CSharpReferenced != nil && c.CSharpReferenced.Pattern == gp.SourceFQN {
				return &ruleList[i]
			}
			if c.BuiltinFilecontent != nil && c.BuiltinFilecontent.Pattern == gp.SourceFQN {
				return &ruleList[i]
			}
		}
		if gp.DependencyName != "" {
			if c.JavaDependency != nil && c.JavaDependency.Name == gp.DependencyName {
				return &ruleList[i]
			}
			if c.GoDependency != nil && c.GoDependency.Name == gp.DependencyName {
				return &ruleList[i]
			}
			// Also match referenced conditions by dependency name to detect
			// condition type mismatches (e.g., golden expects java.dependency
			// but rule uses java.referenced with the same identifier).
			if c.JavaReferenced != nil && c.JavaReferenced.Pattern == gp.DependencyName {
				return &ruleList[i]
			}
			if c.GoReferenced != nil && c.GoReferenced.Pattern == gp.DependencyName {
				return &ruleList[i]
			}
			if c.NodejsReferenced != nil && c.NodejsReferenced.Pattern == gp.DependencyName {
				return &ruleList[i]
			}
			if c.CSharpReferenced != nil && c.CSharpReferenced.Pattern == gp.DependencyName {
				return &ruleList[i]
			}
			if c.BuiltinFilecontent != nil && c.BuiltinFilecontent.Pattern == gp.DependencyName {
				return &ruleList[i]
			}
		}
		for _, entry := range c.Or {
			if gp.SourceFQN != "" {
				if entry.JavaReferenced != nil && entry.JavaReferenced.Pattern == gp.SourceFQN {
					return &ruleList[i]
				}
			}
			if gp.DependencyName != "" {
				if entry.JavaDependency != nil && entry.JavaDependency.Name == gp.DependencyName {
					return &ruleList[i]
				}
			}
		}
	}
	return nil
}

// CheckPassRate (p1-002) verifies the post-fix pass rate meets the threshold.
func CheckPassRate(ctx *EvalContext) CheckResult {
	if ctx.Report == nil {
		return CheckResult{
			ID: "p1-002", Name: "Pass rate (post-fix)",
			Priority: "P1", Passed: false,
			Message: "No report.yaml loaded",
		}
	}
	threshold := 95.0
	if ctx.Golden != nil && ctx.Golden.Thresholds.PassRatePostFix > 0 {
		threshold = ctx.Golden.Thresholds.PassRatePostFix
	}
	passed := ctx.Report.PassRate >= threshold
	return CheckResult{
		ID: "p1-002", Name: "Pass rate (post-fix)",
		Priority: "P1", Passed: passed,
		Message: fmt.Sprintf("Pass rate: %.1f%% (threshold: %.1f%%)", ctx.Report.PassRate, threshold),
	}
}

// CheckDeduplication (p1-009) ensures no duplicate source_fqn or dependency_name
// values exist in patterns.json.
func CheckDeduplication(ctx *EvalContext) CheckResult {
	if ctx.Patterns == nil {
		return CheckResult{
			ID: "p1-009", Name: "No duplicate patterns",
			Priority: "P1", Passed: false,
			Message: "No patterns.json loaded",
		}
	}
	seenFQN := make(map[string]int)
	seenDep := make(map[string]int)
	var dupes []string
	for i, p := range ctx.Patterns.Patterns {
		if p.SourceFQN != "" {
			if prev, ok := seenFQN[p.SourceFQN]; ok {
				dupes = append(dupes, fmt.Sprintf("source_fqn %q: pattern[%d] and pattern[%d]", p.SourceFQN, prev, i))
			}
			seenFQN[p.SourceFQN] = i
		}
		if p.DependencyName != "" {
			if prev, ok := seenDep[p.DependencyName]; ok {
				dupes = append(dupes, fmt.Sprintf("dependency_name %q: pattern[%d] and pattern[%d]", p.DependencyName, prev, i))
			}
			seenDep[p.DependencyName] = i
		}
	}
	if len(dupes) > 0 {
		return CheckResult{
			ID: "p1-009", Name: "No duplicate patterns",
			Priority: "P1", Passed: false,
			Message:  fmt.Sprintf("%d duplicate patterns found", len(dupes)),
			Details:  dupes,
		}
	}
	return CheckResult{
		ID: "p1-009", Name: "No duplicate patterns",
		Priority: "P1", Passed: true,
		Message: fmt.Sprintf("No duplicates among %d patterns", len(ctx.Patterns.Patterns)),
	}
}

// CheckKnownPatterns (p1-005) verifies all golden patterns were extracted.
func CheckKnownPatterns(ctx *EvalContext) CheckResult {
	if ctx.Golden == nil || ctx.Patterns == nil {
		return CheckResult{
			ID: "p1-005", Name: "Known patterns extracted",
			Priority: "P1", Passed: false,
			Message: "Missing golden set or patterns.json",
		}
	}
	fqnSet := make(map[string]bool)
	depSet := make(map[string]bool)
	for _, p := range ctx.Patterns.Patterns {
		if p.SourceFQN != "" {
			fqnSet[p.SourceFQN] = true
		}
		if p.DependencyName != "" {
			depSet[p.DependencyName] = true
		}
	}
	var missing []string
	for _, gp := range ctx.Golden.Patterns {
		found := false
		if gp.SourceFQN != "" && fqnSet[gp.SourceFQN] {
			found = true
		}
		if gp.DependencyName != "" && depSet[gp.DependencyName] {
			found = true
		}
		if !found {
			missing = append(missing, gp.ID)
		}
	}
	if len(missing) > 0 {
		return CheckResult{
			ID: "p1-005", Name: "Known patterns extracted",
			Priority: "P1", Passed: false,
			Message:  fmt.Sprintf("%d/%d golden patterns missing", len(missing), len(ctx.Golden.Patterns)),
			Details:  missing,
		}
	}
	return CheckResult{
		ID: "p1-005", Name: "Known patterns extracted",
		Priority: "P1", Passed: true,
		Message: fmt.Sprintf("All %d golden patterns found", len(ctx.Golden.Patterns)),
	}
}
