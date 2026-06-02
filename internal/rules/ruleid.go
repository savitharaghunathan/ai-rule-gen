package rules

import (
	"fmt"
	"strings"
	"unicode"
)

// IDGenerator produces sequential rule numbers, incrementing by 10
// to leave room for manual insertions.
type IDGenerator struct {
	current int
}

// NewIDGenerator creates a generator. IDs start at 00010 and increment by 10.
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{current: 0}
}

// Next returns the next rule ID with the given prefix.
func (g *IDGenerator) Next(prefix string) string {
	g.current += 10
	return fmt.Sprintf("%s-%05d", prefix, g.current)
}

// Slugify converts a string to a lowercase kebab-case slug suitable for
// use in rule IDs. Non-alphanumeric characters become hyphens; consecutive
// hyphens are collapsed; leading/trailing hyphens are trimmed.
func Slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevHyphen := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
		} else if !prevHyphen {
			b.WriteRune('-')
			prevHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// RuleIDPrefix builds the prefix portion of a rule ID from the concern
// group (filename) and the change type.
// Format: {category}-{change-type}  e.g. "security-removal", "http-client-import"
func RuleIDPrefix(concern, changeType string) string {
	c := Slugify(concern)
	if c == "" {
		c = "general"
	}
	if changeType == "" {
		changeType = "change"
	}
	return c + "-" + changeType
}

// ChangeType derives the change-type component of a rule ID from a
// pattern's detection mechanism.
func ChangeType(locationType, providerType, dependencyName, xpath string) string {
	if dependencyName != "" {
		return "dependency"
	}
	if xpath != "" {
		return "xml"
	}
	switch locationType {
	case "IMPORT", "PACKAGE":
		return "import"
	case "METHOD_CALL", "CONSTRUCTOR_CALL":
		return "method"
	case "ANNOTATION":
		return "annotation"
	case "INHERITANCE", "IMPLEMENTS_TYPE":
		return "type"
	}
	if providerType == "builtin" {
		return "pattern"
	}
	return "change"
}
