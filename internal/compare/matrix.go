package compare

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MatchKeys returns sorted, deduplicated condition fingerprints for a rule.
// Each key is `<kind>:<value>` where kind is the condition provider (java,
// dep, xml, fc, ...). javax/jakarta package prefixes collapse so rules
// matching either namespace produce equal keys.
func MatchKeys(r RawRule) []string {
	if r.When == nil {
		return nil
	}
	var keys []string
	walkWhen(r.When, &keys)
	out := uniq(keys)
	sort.Strings(out)
	return out
}

func walkWhen(n *yaml.Node, out *[]string) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i].Value
			v := n.Content[i+1]
			switch k {
			case "or", "and":
				walkWhen(v, out)
			case "java.referenced":
				*out = append(*out, "java:"+normalizeJava(getField(v, "pattern")))
			case "java.dependency":
				name := getField(v, "name")
				if name == "" {
					name = getField(v, "name_regex")
				}
				if name == "" {
					name = getField(v, "nameregex")
				}
				if name != "" {
					*out = append(*out, "dep:"+strings.ToLower(name))
				}
			case "go.referenced":
				*out = append(*out, "go:"+getField(v, "pattern"))
			case "nodejs.referenced":
				*out = append(*out, "node:"+getField(v, "pattern"))
			case "csharp.referenced":
				*out = append(*out, "cs:"+getField(v, "pattern"))
			case "python.referenced":
				*out = append(*out, "py:"+getField(v, "pattern"))
			case "builtin.filecontent":
				*out = append(*out, "fc:"+getField(v, "pattern"))
			case "builtin.xml":
				if xp := getField(v, "xpath"); xp != "" {
					*out = append(*out, "xml:"+normalizeXPath(xp))
				}
			case "builtin.file":
				*out = append(*out, "file:"+getField(v, "pattern"))
			case "builtin.json":
				*out = append(*out, "json:"+getField(v, "xpath"))
			case "builtin.xmlPublicID":
				*out = append(*out, "xmlpid:"+getField(v, "regex"))
			default:
				// Recurse into chaining wrappers (from/as) and nested mappings.
				if v.Kind == yaml.MappingNode || v.Kind == yaml.SequenceNode {
					walkWhen(v, out)
				}
			}
		}
	case yaml.SequenceNode:
		for _, child := range n.Content {
			walkWhen(child, out)
		}
	}
}

func getField(n *yaml.Node, key string) string {
	if n == nil || n.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1].Value
		}
	}
	return ""
}

func normalizeJava(p string) string {
	s := strings.ToLower(p)
	if strings.HasPrefix(s, "jakarta.") {
		s = "javax." + s[len("jakarta."):]
	}
	return s
}

func normalizeXPath(x string) string {
	return strings.Join(strings.Fields(x), " ")
}

// BuildMatrix scores coverage of A against B and B against A.
func BuildMatrix(rulesA, rulesB []RawRule) Matrix {
	keysA := indexKeys(rulesA)
	keysB := indexKeys(rulesB)

	a := classify(rulesA, keysA, keysB)
	b := classify(rulesB, keysB, keysA)

	return Matrix{
		AInB:    a,
		BInA:    b,
		Summary: tallyMatrix(a, b),
	}
}

func indexKeys(rs []RawRule) map[string][]string {
	out := make(map[string][]string, len(rs))
	for _, r := range rs {
		out[r.RuleID] = MatchKeys(r)
	}
	return out
}

func classify(side []RawRule, sideKeys, otherKeys map[string][]string) []RuleCoverage {
	inverse := make(map[string][]string)
	for id, keys := range otherKeys {
		for _, k := range keys {
			inverse[k] = append(inverse[k], id)
		}
	}

	out := make([]RuleCoverage, 0, len(side))
	for _, r := range side {
		keys := sideKeys[r.RuleID]
		matched := map[string]bool{}
		partial := map[string]bool{}

		for _, k := range keys {
			for _, id := range inverse[k] {
				matched[id] = true
			}
			if !strings.HasPrefix(k, "java:") {
				continue
			}
			base := strings.TrimSuffix(strings.TrimSuffix(k, "*"), ".")
			for otherKey, ids := range inverse {
				if !strings.HasPrefix(otherKey, "java:") || otherKey == k {
					continue
				}
				otherBase := strings.TrimSuffix(strings.TrimSuffix(otherKey, "*"), ".")
				if strings.HasPrefix(base, otherBase+".") || strings.HasPrefix(otherBase, base+".") {
					for _, id := range ids {
						if !matched[id] {
							partial[id] = true
						}
					}
				}
			}
		}

		status := "missing"
		switch {
		case len(matched) > 0:
			status = "covered"
		case len(partial) > 0:
			status = "partial"
		}

		out = append(out, RuleCoverage{
			RuleID:      r.RuleID,
			Status:      status,
			MatchKeys:   keys,
			MatchedBy:   sortedKeys(matched),
			PartialBy:   sortedKeys(partial),
			Description: r.Description,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RuleID < out[j].RuleID })
	return out
}

func tallyMatrix(a, b []RuleCoverage) MatrixSummary {
	var s MatrixSummary
	for _, c := range a {
		switch c.Status {
		case "covered":
			s.AInBCovered++
		case "partial":
			s.AInBPartial++
		case "missing":
			s.AInBMissing++
		}
	}
	for _, c := range b {
		switch c.Status {
		case "covered":
			s.BInACovered++
		case "partial":
			s.BInAPartial++
		case "missing":
			s.BInAMissing++
		}
	}
	return s
}

func uniq(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
