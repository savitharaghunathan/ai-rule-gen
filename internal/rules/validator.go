package rules

import (
	"fmt"
	"regexp"
	"strings"
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

		// Label format
		for _, label := range r.Labels {
			validateLabel(label, prefix, &result)
		}

		// Condition-specific validation
		validateCondition(r.When, prefix, &result)
	}

	return result
}

func validateLabel(label, prefix string, result *ValidationResult) {
	if strings.HasPrefix(label, "konveyor.io/") {
		parts := strings.SplitN(label, "=", 2)
		key := parts[0]
		switch key {
		case "konveyor.io/source", "konveyor.io/target":
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
	if c.PythonReferenced != nil && c.PythonReferenced.Pattern == "" {
		result.addError("%s: python.referenced missing required field 'pattern'", prefix)
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

func isEmptyCondition(c Condition) bool {
	return c.JavaReferenced == nil &&
		c.JavaDependency == nil &&
		c.GoReferenced == nil &&
		c.GoDependency == nil &&
		c.NodejsReferenced == nil &&
		c.CSharpReferenced == nil &&
		c.PythonReferenced == nil &&
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
