package eval

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
	"gopkg.in/yaml.v3"
)

type CheckResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Agent    string `json:"agent"`
	Priority string `json:"priority"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message"`
	Details  any    `json:"details,omitempty"`
}

type AgentEval struct {
	Checks  []CheckResult `json:"checks"`
	Summary EvalSummary   `json:"summary"`
}

type EvalSummary struct {
	Total    int     `json:"total"`
	Passed   int     `json:"passed"`
	Failed   int     `json:"failed"`
	PassRate float64 `json:"pass_rate"`
}

type EvalReport struct {
	Timestamp string                `json:"timestamp"`
	GoldenSet string                `json:"golden_set"`
	Agents    map[string]*AgentEval `json:"agents"`
	Summary   EvalSummary           `json:"summary"`
}

type EvalContext struct {
	Golden        *GoldenSet
	Patterns      *rules.ExtractOutput
	Rules         []rules.Rule
	Report        *workspace.Report
	PreFixReport  *workspace.Report
	RulesSnapshot []rules.Rule
}

// --- Rule Writer Graders ---

func CheckRequiredFields(ctx *EvalContext) CheckResult {
	if ctx.Patterns == nil {
		return CheckResult{
			ID: "rw-002", Name: "Required fields in patterns.json",
			Agent: "rule-writer", Priority: "P0", Passed: false,
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
			ID: "rw-002", Name: "Required fields in patterns.json",
			Agent: "rule-writer", Priority: "P0", Passed: false,
			Message:  fmt.Sprintf("%d patterns missing required fields", len(missing)),
			Details:  missing,
		}
	}
	return CheckResult{
		ID: "rw-002", Name: "Required fields in patterns.json",
		Agent: "rule-writer", Priority: "P0", Passed: true,
		Message: fmt.Sprintf("All %d patterns have required fields", len(ctx.Patterns.Patterns)),
	}
}

func CheckRuleValidity(ctx *EvalContext) CheckResult {
	if len(ctx.Rules) == 0 {
		return CheckResult{
			ID: "rw-001", Name: "Rule YAML validity",
			Agent: "rule-writer", Priority: "P0", Passed: false,
			Message: "No rules loaded",
		}
	}
	result := rules.Validate(ctx.Rules)
	if result.Valid {
		return CheckResult{
			ID: "rw-001", Name: "Rule YAML validity",
			Agent: "rule-writer", Priority: "P0", Passed: true,
			Message: fmt.Sprintf("All %d rules are valid", result.RuleCount),
		}
	}
	return CheckResult{
		ID: "rw-001", Name: "Rule YAML validity",
		Agent: "rule-writer", Priority: "P0", Passed: false,
		Message:  fmt.Sprintf("%d validation errors", len(result.Errors)),
		Details:  result.Errors,
	}
}

func CheckDeduplication(ctx *EvalContext) CheckResult {
	if ctx.Patterns == nil {
		return CheckResult{
			ID: "rw-005", Name: "No duplicate patterns",
			Agent: "rule-writer", Priority: "P1", Passed: false,
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
			ID: "rw-005", Name: "No duplicate patterns",
			Agent: "rule-writer", Priority: "P1", Passed: false,
			Message:  fmt.Sprintf("%d duplicate patterns found", len(dupes)),
			Details:  dupes,
		}
	}
	return CheckResult{
		ID: "rw-005", Name: "No duplicate patterns",
		Agent: "rule-writer", Priority: "P1", Passed: true,
		Message: fmt.Sprintf("No duplicates among %d patterns", len(ctx.Patterns.Patterns)),
	}
}

func CheckKnownPatterns(ctx *EvalContext) CheckResult {
	if ctx.Golden == nil || ctx.Patterns == nil {
		return CheckResult{
			ID: "rw-003", Name: "Known patterns extracted",
			Agent: "rule-writer", Priority: "P1", Passed: false,
			Message: "Missing golden set or patterns.json",
		}
	}
	fqnSet := make(map[string]bool)
	depSet := make(map[string]bool)
	xpathSet := make(map[string]bool)
	for _, p := range ctx.Patterns.Patterns {
		if p.SourceFQN != "" {
			fqnSet[p.SourceFQN] = true
		}
		if p.DependencyName != "" {
			depSet[p.DependencyName] = true
		}
		if p.XPath != "" {
			xpathSet[p.XPath] = true
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
		if gp.XPath != "" && xpathSet[gp.XPath] {
			found = true
		}
		if !found {
			missing = append(missing, gp.ID)
		}
	}
	if len(missing) > 0 {
		return CheckResult{
			ID: "rw-003", Name: "Known patterns extracted",
			Agent: "rule-writer", Priority: "P1", Passed: false,
			Message:  fmt.Sprintf("%d/%d golden patterns missing", len(missing), len(ctx.Golden.Patterns)),
			Details:  missing,
		}
	}
	return CheckResult{
		ID: "rw-003", Name: "Known patterns extracted",
		Agent: "rule-writer", Priority: "P1", Passed: true,
		Message: fmt.Sprintf("All %d golden patterns found", len(ctx.Golden.Patterns)),
	}
}

func CheckConditionTypes(ctx *EvalContext) CheckResult {
	if ctx.Golden == nil || len(ctx.Rules) == 0 {
		return CheckResult{
			ID: "rw-004", Name: "Correct condition types",
			Agent: "rule-writer", Priority: "P1", Passed: false,
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
			ID: "rw-004", Name: "Correct condition types",
			Agent: "rule-writer", Priority: "P1", Passed: false,
			Message:  fmt.Sprintf("%d condition type mismatches", len(mismatches)),
			Details:  mismatches,
		}
	}
	return CheckResult{
		ID: "rw-004", Name: "Correct condition types",
		Agent: "rule-writer", Priority: "P1", Passed: true,
		Message: "All golden patterns have correct condition types",
	}
}

// --- Test Generator Graders ---

func CheckPreFixPassRate(ctx *EvalContext) CheckResult {
	if ctx.PreFixReport == nil {
		return CheckResult{
			ID: "tg-001", Name: "Pre-fix pass rate",
			Agent: "test-generator", Priority: "P1", Passed: false,
			Message: "No pre-fix report loaded",
		}
	}
	threshold := 80.0
	if ctx.Golden != nil && ctx.Golden.Thresholds.PreFixPassRate > 0 {
		threshold = ctx.Golden.Thresholds.PreFixPassRate
	}
	passed := ctx.PreFixReport.PassRate >= threshold
	return CheckResult{
		ID: "tg-001", Name: "Pre-fix pass rate",
		Agent: "test-generator", Priority: "P1", Passed: passed,
		Message: fmt.Sprintf("Pre-fix pass rate: %.1f%% (threshold: %.1f%%)", ctx.PreFixReport.PassRate, threshold),
	}
}

// --- Rule Validator Graders ---

func CheckRuleIntegrity(ctx *EvalContext) CheckResult {
	if ctx.RulesSnapshot == nil || len(ctx.Rules) == 0 {
		return CheckResult{
			ID: "rv-001", Name: "Rule integrity after fix loop",
			Agent: "validator", Priority: "P0", Passed: false,
			Message: "Missing rules snapshot or final rules",
		}
	}
	snapshotConditions := make(map[string][]byte)
	for _, r := range ctx.RulesSnapshot {
		data, _ := yaml.Marshal(r.When)
		snapshotConditions[r.RuleID] = data
	}
	var changed []string
	for _, r := range ctx.Rules {
		data, _ := yaml.Marshal(r.When)
		if snap, ok := snapshotConditions[r.RuleID]; ok {
			if !bytes.Equal(snap, data) {
				changed = append(changed, r.RuleID)
			}
		}
	}
	if len(changed) > 0 {
		return CheckResult{
			ID: "rv-001", Name: "Rule integrity after fix loop",
			Agent: "validator", Priority: "P0", Passed: false,
			Message:  fmt.Sprintf("%d rules had conditions modified by fix loop", len(changed)),
			Details:  changed,
		}
	}
	return CheckResult{
		ID: "rv-001", Name: "Rule integrity after fix loop",
		Agent: "validator", Priority: "P0", Passed: true,
		Message: fmt.Sprintf("All %d rules unchanged after fix loop", len(ctx.Rules)),
	}
}

func CheckFixEffectiveness(ctx *EvalContext) CheckResult {
	if ctx.PreFixReport == nil || ctx.Report == nil {
		return CheckResult{
			ID: "rv-002", Name: "Fix loop effectiveness",
			Agent: "validator", Priority: "P1", Passed: false,
			Message: "Missing pre-fix or post-fix report",
		}
	}
	if ctx.PreFixReport.TestsFailed == 0 {
		return CheckResult{
			ID: "rv-002", Name: "Fix loop effectiveness",
			Agent: "validator", Priority: "P1", Passed: true,
			Message: "No failures to fix",
		}
	}
	fixed := ctx.PreFixReport.TestsFailed - ctx.Report.TestsFailed
	passed := ctx.Report.TestsFailed < ctx.PreFixReport.TestsFailed
	return CheckResult{
		ID: "rv-002", Name: "Fix loop effectiveness",
		Agent: "validator", Priority: "P1", Passed: passed,
		Message: fmt.Sprintf("Fixed %d/%d failures (pre: %d failed, post: %d failed)",
			fixed, ctx.PreFixReport.TestsFailed,
			ctx.PreFixReport.TestsFailed, ctx.Report.TestsFailed),
	}
}

func CheckNoRegressions(ctx *EvalContext) CheckResult {
	if ctx.PreFixReport == nil || ctx.Report == nil {
		return CheckResult{
			ID: "rv-003", Name: "No regressions from fix loop",
			Agent: "validator", Priority: "P0", Passed: false,
			Message: "Missing pre-fix or post-fix report",
		}
	}
	preFailSet := make(map[string]bool)
	for _, id := range ctx.PreFixReport.FailedRules {
		preFailSet[id] = true
	}
	var regressions []string
	for _, id := range ctx.Report.FailedRules {
		if !preFailSet[id] {
			regressions = append(regressions, id)
		}
	}
	if len(regressions) > 0 {
		return CheckResult{
			ID: "rv-003", Name: "No regressions from fix loop",
			Agent: "validator", Priority: "P0", Passed: false,
			Message:  fmt.Sprintf("%d rules regressed after fix loop", len(regressions)),
			Details:  regressions,
		}
	}
	return CheckResult{
		ID: "rv-003", Name: "No regressions from fix loop",
		Agent: "validator", Priority: "P0", Passed: true,
		Message: "No regressions",
	}
}

// --- Pipeline Graders ---

func CheckPassRate(ctx *EvalContext) CheckResult {
	if ctx.Report == nil {
		return CheckResult{
			ID: "e2e-001", Name: "Pass rate (post-fix)",
			Agent: "pipeline", Priority: "P1", Passed: false,
			Message: "No report.yaml loaded",
		}
	}
	threshold := 95.0
	if ctx.Golden != nil && ctx.Golden.Thresholds.PassRatePostFix > 0 {
		threshold = ctx.Golden.Thresholds.PassRatePostFix
	}
	passed := ctx.Report.PassRate >= threshold
	return CheckResult{
		ID: "e2e-001", Name: "Pass rate (post-fix)",
		Agent: "pipeline", Priority: "P1", Passed: passed,
		Message: fmt.Sprintf("Pass rate: %.1f%% (threshold: %.1f%%)", ctx.Report.PassRate, threshold),
	}
}

// --- Helpers ---

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
			if c.BuiltinXML != nil && c.BuiltinXML.XPath == gp.SourceFQN {
				return &ruleList[i]
			}
		}
		if gp.XPath != "" {
			if c.BuiltinXML != nil && c.BuiltinXML.XPath == gp.XPath {
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

// --- RunAll ---

func addChecks(agents map[string]*AgentEval, checks ...CheckResult) {
	for _, c := range checks {
		ae, ok := agents[c.Agent]
		if !ok {
			ae = &AgentEval{}
			agents[c.Agent] = ae
		}
		ae.Checks = append(ae.Checks, c)
	}
}

func RunAll(ctx *EvalContext) *EvalReport {
	agents := make(map[string]*AgentEval)

	if ctx.Patterns != nil || len(ctx.Rules) > 0 {
		addChecks(agents, CheckRuleValidity(ctx))
		addChecks(agents, CheckRequiredFields(ctx))
		addChecks(agents, CheckDeduplication(ctx))
	}
	if ctx.Golden != nil && ctx.Patterns != nil {
		addChecks(agents, CheckKnownPatterns(ctx))
	}
	if ctx.Golden != nil && len(ctx.Rules) > 0 {
		addChecks(agents, CheckConditionTypes(ctx))
	}

	if ctx.PreFixReport != nil {
		addChecks(agents, CheckPreFixPassRate(ctx))
	}

	if ctx.RulesSnapshot != nil && len(ctx.Rules) > 0 {
		addChecks(agents, CheckRuleIntegrity(ctx))
	}
	if ctx.PreFixReport != nil && ctx.Report != nil {
		addChecks(agents, CheckFixEffectiveness(ctx))
		addChecks(agents, CheckNoRegressions(ctx))
	}

	if ctx.Report != nil {
		addChecks(agents, CheckPassRate(ctx))
	}

	totalPassed, totalCount := 0, 0
	for _, ae := range agents {
		p := 0
		for _, c := range ae.Checks {
			if c.Passed {
				p++
			}
		}
		total := len(ae.Checks)
		var rate float64
		if total > 0 {
			rate = float64(p) / float64(total) * 100
		}
		ae.Summary = EvalSummary{Total: total, Passed: p, Failed: total - p, PassRate: rate}
		totalPassed += p
		totalCount += total
	}

	var totalRate float64
	if totalCount > 0 {
		totalRate = float64(totalPassed) / float64(totalCount) * 100
	}

	goldenName := ""
	if ctx.Golden != nil {
		goldenName = fmt.Sprintf("%s-to-%s", ctx.Golden.Source, ctx.Golden.Target)
	}

	return &EvalReport{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		GoldenSet: goldenName,
		Agents:    agents,
		Summary: EvalSummary{
			Total:    totalCount,
			Passed:   totalPassed,
			Failed:   totalCount - totalPassed,
			PassRate: totalRate,
		},
	}
}
