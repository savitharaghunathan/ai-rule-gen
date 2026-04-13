package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationResult holds errors and warnings from rule validation.
type ValidationResult struct {
	Valid     bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	RuleCount int     `json:"rule_count"`
}

// Validate checks a set of rules for structural validity.
func Validate(rules []Rule) ValidationResult {
	result := ValidationResult{
		Valid:     true,
		RuleCount: len(rules),
	}
	seenIDs := make(map[string]bool)

	for i, r := range rules {
		prefix := fmt.Sprintf("rule[%d]", i)
		if r.RuleID != "" {
			prefix = fmt.Sprintf("rule[%d] (%s)", i, r.RuleID)
		}

		// Required: ruleID
		if r.RuleID == "" {
			result.addError("%s: missing required field 'ruleID'", prefix)
		} else {
			if strings.ContainsAny(r.RuleID, "\n\r;") {
				result.addError("%s: ruleID must not contain newlines or semicolons", prefix)
			}
			if seenIDs[r.RuleID] {
				result.addError("%s: duplicate ruleID", prefix)
			}
			seenIDs[r.RuleID] = true
		}

		// Required: message or tag
		if r.Message == "" && len(r.Tag) == 0 {
			result.addError("%s: either 'message' or 'tag' must be set", prefix)
		}

		// Required: when condition
		if isEmptyCondition(r.When) {
			result.addError("%s: missing required field 'when'", prefix)
		}

		// Description: warn if missing (production rules should have one)
		if r.Description == "" {
			result.addWarning("%s: missing 'description' field", prefix)
		}

		// Category validation
		if r.Category != "" {
			switch r.Category {
			case CategoryMandatory, CategoryOptional, CategoryPotential:
				// valid
			default:
				result.addError("%s: invalid category %q (must be mandatory, optional, or potential)", prefix, r.Category)
			}
		}

		// Effort range
		if r.Effort < 0 || r.Effort > 10 {
			result.addWarning("%s: effort %d is outside expected range 1-10", prefix, r.Effort)
		}
		if r.Effort == 0 {
			result.addWarning("%s: effort is 0 (likely unset)", prefix)
		}

		// Link URL validation
		for j, link := range r.Links {
			if link.URL == "" {
				result.addWarning("%s: links[%d] has empty URL", prefix, j)
			} else if !strings.HasPrefix(link.URL, "http://") && !strings.HasPrefix(link.URL, "https://") {
				result.addWarning("%s: links[%d] URL %q should start with http:// or https://", prefix, j, link.URL)
			}
		}

		// Label format
		for _, label := range r.Labels {
			validateLabel(label, prefix, &result)
		}

		// Condition-specific validation
		validateCondition(r.When, prefix, &result)

		// Multiple bare conditions check: only one condition type at the top level
		if n := countConditionTypes(r.When); n > 1 {
			result.addError("%s: when block has %d condition types set; use 'or' or 'and' combinator to combine multiple conditions", prefix, n)
		}
	}

	return result
}

func validateLabel(label, prefix string, result *ValidationResult) {
	if strings.HasPrefix(label, "konveyor.io/") {
		parts := strings.SplitN(label, "=", 2)
		key := parts[0]
		switch key {
		case "konveyor.io/source", "konveyor.io/target",
			LabelGeneratedBy, LabelTestResult, LabelReview:
			if len(parts) != 2 || parts[1] == "" {
				result.addWarning("%s: label %q should have a value (e.g., %s=value)", prefix, label, key)
			}
		}
	}
}

func validateCondition(c Condition, prefix string, result *ValidationResult) {
	if c.JavaReferenced != nil {
		if c.JavaReferenced.Pattern == "" {
			result.addError("%s: java.referenced missing required field 'pattern'", prefix)
		}
		if c.JavaReferenced.Location != "" && !ValidJavaLocations[c.JavaReferenced.Location] {
			result.addError("%s: java.referenced invalid location %q", prefix, c.JavaReferenced.Location)
		}
	}
	if c.JavaDependency != nil {
		if c.JavaDependency.Name == "" && c.JavaDependency.NameRegex == "" {
			result.addError("%s: java.dependency requires 'name' or 'name_regex'", prefix)
		}
		validateRegex(c.JavaDependency.NameRegex, prefix+": java.dependency name_regex", result)
	}
	if c.GoReferenced != nil && c.GoReferenced.Pattern == "" {
		result.addError("%s: go.referenced missing required field 'pattern'", prefix)
	}
	if c.GoDependency != nil {
		if c.GoDependency.Name == "" && c.GoDependency.NameRegex == "" {
			result.addError("%s: go.dependency requires 'name' or 'name_regex'", prefix)
		}
	}
	if c.NodejsReferenced != nil && c.NodejsReferenced.Pattern == "" {
		result.addError("%s: nodejs.referenced missing required field 'pattern'", prefix)
	}
	if c.CSharpReferenced != nil {
		if c.CSharpReferenced.Pattern == "" {
			result.addError("%s: csharp.referenced missing required field 'pattern'", prefix)
		}
		if c.CSharpReferenced.Location != "" && !ValidCSharpLocations[c.CSharpReferenced.Location] {
			result.addError("%s: csharp.referenced invalid location %q", prefix, c.CSharpReferenced.Location)
		}
	}
	if c.BuiltinFilecontent != nil {
		if c.BuiltinFilecontent.Pattern == "" {
			result.addError("%s: builtin.filecontent missing required field 'pattern'", prefix)
		}
		validateRegex(c.BuiltinFilecontent.Pattern, prefix+": builtin.filecontent pattern", result)
		validateRegex(c.BuiltinFilecontent.FilePattern, prefix+": builtin.filecontent filePattern", result)
	}
	if c.BuiltinFile != nil && c.BuiltinFile.Pattern == "" {
		result.addError("%s: builtin.file missing required field 'pattern'", prefix)
	}
	if c.BuiltinXML != nil && c.BuiltinXML.XPath == "" {
		result.addError("%s: builtin.xml missing required field 'xpath'", prefix)
	}
	if c.BuiltinJSON != nil && c.BuiltinJSON.XPath == "" {
		result.addError("%s: builtin.json missing required field 'xpath'", prefix)
	}
	if c.BuiltinXMLPublicID != nil {
		if c.BuiltinXMLPublicID.Regex == "" {
			result.addError("%s: builtin.xmlPublicID missing required field 'regex'", prefix)
		}
		validateRegex(c.BuiltinXMLPublicID.Regex, prefix+": builtin.xmlPublicID regex", result)
	}

	for i, entry := range c.Or {
		validateCondition(entry.Condition, fmt.Sprintf("%s: or[%d]", prefix, i), result)
	}
	for i, entry := range c.And {
		validateCondition(entry.Condition, fmt.Sprintf("%s: and[%d]", prefix, i), result)
	}
}

func validateRegex(pattern, context string, result *ValidationResult) {
	if pattern == "" {
		return
	}
	if _, err := regexp.Compile(pattern); err != nil {
		result.addError("%s: invalid regex %q: %v", context, pattern, err)
	}
}

// countConditionTypes returns how many provider condition types are set on a Condition.
// Combinators (or/and) and chaining fields (from/as/ignore/not) are not counted.
// A valid top-level when block should have exactly 0 (combinator-only) or 1.
func countConditionTypes(c Condition) int {
	n := 0
	if c.JavaReferenced != nil {
		n++
	}
	if c.JavaDependency != nil {
		n++
	}
	if c.GoReferenced != nil {
		n++
	}
	if c.GoDependency != nil {
		n++
	}
	if c.NodejsReferenced != nil {
		n++
	}
	if c.CSharpReferenced != nil {
		n++
	}
	if c.BuiltinFilecontent != nil {
		n++
	}
	if c.BuiltinFile != nil {
		n++
	}
	if c.BuiltinXML != nil {
		n++
	}
	if c.BuiltinJSON != nil {
		n++
	}
	if len(c.BuiltinHasTags) > 0 {
		n++
	}
	if c.BuiltinXMLPublicID != nil {
		n++
	}
	return n
}

func isEmptyCondition(c Condition) bool {
	return c.JavaReferenced == nil &&
		c.JavaDependency == nil &&
		c.GoReferenced == nil &&
		c.GoDependency == nil &&
		c.NodejsReferenced == nil &&
		c.CSharpReferenced == nil &&
		c.BuiltinFilecontent == nil &&
		c.BuiltinFile == nil &&
		c.BuiltinXML == nil &&
		c.BuiltinJSON == nil &&
		c.BuiltinHasTags == nil &&
		c.BuiltinXMLPublicID == nil &&
		c.Or == nil &&
		c.And == nil
}

func (r *ValidationResult) addError(format string, args ...any) {
	r.Valid = false
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

func (r *ValidationResult) addWarning(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// ConsistencyResult holds errors from bidirectional rule ↔ test validation.
type ConsistencyResult struct {
	Valid            bool     `json:"valid"`
	Errors           []string `json:"errors,omitempty"`
	RulesWithoutTests []string `json:"rules_without_tests,omitempty"`
	TestsWithoutRules []string `json:"tests_without_rules,omitempty"`
}

// ValidateConsistency checks bidirectional consistency between rules and test files.
// Every rule should have at least one test case, and every test case should reference a real rule.
func ValidateConsistency(rulesDir, testsDir string) (*ConsistencyResult, error) {
	// Collect rule IDs
	allRules, err := ReadRulesDir(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("reading rules: %w", err)
	}
	ruleIDs := make(map[string]bool, len(allRules))
	for _, r := range allRules {
		if r.RuleID != "" {
			ruleIDs[r.RuleID] = true
		}
	}

	// Collect test rule IDs
	testRuleIDs, err := readTestRuleIDs(testsDir)
	if err != nil {
		return nil, fmt.Errorf("reading tests: %w", err)
	}

	result := &ConsistencyResult{Valid: true}

	// Check: every rule has a test
	for id := range ruleIDs {
		if !testRuleIDs[id] {
			result.RulesWithoutTests = append(result.RulesWithoutTests, id)
		}
	}

	// Check: every test references a real rule
	for id := range testRuleIDs {
		if !ruleIDs[id] {
			result.TestsWithoutRules = append(result.TestsWithoutRules, id)
		}
	}

	// Sort for deterministic output
	sort.Strings(result.RulesWithoutTests)
	sort.Strings(result.TestsWithoutRules)

	if len(result.RulesWithoutTests) > 0 {
		result.Valid = false
		for _, id := range result.RulesWithoutTests {
			result.Errors = append(result.Errors, fmt.Sprintf("rule %q has no test case", id))
		}
	}
	if len(result.TestsWithoutRules) > 0 {
		result.Valid = false
		for _, id := range result.TestsWithoutRules {
			result.Errors = append(result.Errors, fmt.Sprintf("test references non-existent rule %q", id))
		}
	}

	return result, nil
}

// testFileEntry is a minimal struct to parse test YAML for rule IDs only.
type testFileEntry struct {
	Tests []struct {
		RuleID string `yaml:"ruleID"`
	} `yaml:"tests"`
}

// readTestRuleIDs reads all .test.yaml/.test.yml files in a directory and
// returns the set of rule IDs referenced.
func readTestRuleIDs(testsDir string) (map[string]bool, error) {
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		return nil, fmt.Errorf("reading tests directory %s: %w", testsDir, err)
	}

	ids := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".test.yaml") && !strings.HasSuffix(name, ".test.yml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(testsDir, name))
		if err != nil {
			return nil, fmt.Errorf("reading test file %s: %w", name, err)
		}
		var tf testFileEntry
		if err := yaml.Unmarshal(data, &tf); err != nil {
			return nil, fmt.Errorf("parsing test file %s: %w", name, err)
		}
		for _, t := range tf.Tests {
			if t.RuleID != "" {
				ids[t.RuleID] = true
			}
		}
	}

	return ids, nil
}
