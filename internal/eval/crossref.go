package eval

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

var skipDirs = map[string]bool{
	"target":       true,
	"build":        true,
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"bin":          true,
}

func CrossRefNotFired(ruleList []rules.Rule, notFired []string, appDir string) []UnmatchedRule {
	ruleMap := make(map[string]rules.Rule, len(ruleList))
	for _, r := range ruleList {
		ruleMap[r.RuleID] = r
	}

	var results []UnmatchedRule
	for _, id := range notFired {
		r, ok := ruleMap[id]
		if !ok {
			continue
		}

		terms, pattern, exts := extractSearchInfo(r.When)
		if len(terms) == 0 {
			results = append(results, UnmatchedRule{
				RuleID:  id,
				Pattern: pattern,
				InApp:   false,
				Reason:  "no searchable terms extracted from condition",
			})
			continue
		}

		found, files := searchAppSource(appDir, terms, exts)
		um := UnmatchedRule{
			RuleID:  id,
			Pattern: pattern,
		}
		if found {
			um.InApp = true
			um.AppFiles = files
			um.Reason = "pattern found in app source but kantra did not match"
		} else {
			um.InApp = false
			um.Reason = "not in app"
		}
		results = append(results, um)
	}
	return results
}

func extractSearchInfo(cond rules.Condition) (terms []string, pattern string, extensions []string) {
	if cond.JavaReferenced != nil {
		pattern = cond.JavaReferenced.Pattern
		terms = fqnTerms(pattern)
		extensions = []string{".java"}
		return
	}
	if cond.JavaDependency != nil {
		pattern = cond.JavaDependency.Name
		if pattern != "" {
			parts := strings.SplitN(pattern, ":", 2)
			if len(parts) == 2 {
				terms = []string{parts[0], parts[1]}
			} else {
				terms = []string{pattern}
			}
		}
		extensions = []string{".xml"}
		return
	}
	if cond.GoReferenced != nil {
		pattern = cond.GoReferenced.Pattern
		terms = fqnTerms(pattern)
		extensions = []string{".go"}
		return
	}
	if cond.GoDependency != nil {
		pattern = cond.GoDependency.Name
		if pattern != "" {
			terms = []string{pattern}
		}
		extensions = []string{".go", ".mod"}
		return
	}
	if cond.NodejsReferenced != nil {
		pattern = cond.NodejsReferenced.Pattern
		terms = fqnTerms(pattern)
		extensions = []string{".js", ".ts", ".mjs", ".cjs"}
		return
	}
	if cond.CSharpReferenced != nil {
		pattern = cond.CSharpReferenced.Pattern
		terms = fqnTerms(pattern)
		extensions = []string{".cs"}
		return
	}
	if cond.PythonReferenced != nil {
		pattern = cond.PythonReferenced.Pattern
		terms = fqnTerms(pattern)
		extensions = []string{".py"}
		return
	}
	if cond.BuiltinFilecontent != nil {
		pattern = cond.BuiltinFilecontent.Pattern
		terms = regexLiterals(pattern)
		extensions = sourceExtensionsFromFilePattern(cond.BuiltinFilecontent.FilePattern)
		return
	}
	if cond.BuiltinXML != nil {
		pattern = cond.BuiltinXML.XPath
		terms = xpathTerms(pattern)
		extensions = []string{".xml"}
		return
	}

	if len(cond.Or) > 0 {
		for _, entry := range cond.Or {
			t, p, e := extractSearchInfo(entry.Condition)
			terms = append(terms, t...)
			if pattern == "" {
				pattern = p
			}
			extensions = mergeExts(extensions, e)
		}
		return
	}
	if len(cond.And) > 0 {
		for _, entry := range cond.And {
			t, p, e := extractSearchInfo(entry.Condition)
			terms = append(terms, t...)
			if pattern == "" {
				pattern = p
			}
			extensions = mergeExts(extensions, e)
		}
		return
	}

	return nil, "", nil
}

func fqnTerms(fqn string) []string {
	fqn = strings.TrimSpace(fqn)
	if fqn == "" {
		return nil
	}

	parts := strings.Split(fqn, ".")
	if len(parts) == 0 {
		return nil
	}

	var terms []string
	last := parts[len(parts)-1]
	if last != "" && last != "*" {
		terms = append(terms, last)
	}

	if len(parts) >= 2 {
		secondLast := parts[len(parts)-2]
		if secondLast != "" && isClassName(secondLast) {
			terms = append(terms, secondLast)
		}
	}
	return terms
}

func isClassName(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= 'A' && s[0] <= 'Z'
}

func regexLiterals(pattern string) []string {
	var terms []string
	var current strings.Builder
	metaChars := `\.^$*+?{}[]|()`

	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		if strings.ContainsRune(metaChars, rune(ch)) {
			if ch == '\\' && i+1 < len(pattern) {
				next := pattern[i+1]
				if !strings.ContainsRune(metaChars, rune(next)) {
					if current.Len() >= 4 {
						terms = append(terms, current.String())
					}
					current.Reset()
					i++
					continue
				}
				current.WriteByte(next)
				i++
			} else {
				if current.Len() >= 4 {
					terms = append(terms, current.String())
				}
				current.Reset()
			}
		} else {
			current.WriteByte(ch)
		}
	}
	if current.Len() >= 4 {
		terms = append(terms, current.String())
	}
	return terms
}

func xpathTerms(xpath string) []string {
	var terms []string
	parts := strings.FieldsFunc(xpath, func(r rune) bool {
		return r == '/' || r == '[' || r == ']' || r == '@' || r == '(' || r == ')' || r == '=' || r == '\'' || r == '"'
	})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) >= 3 && !strings.HasPrefix(p, "text") && p != "and" && p != "not" {
			if idx := strings.Index(p, ":"); idx >= 0 {
				p = p[idx+1:]
			}
			if len(p) >= 3 {
				terms = append(terms, p)
			}
		}
	}
	return terms
}

func sourceExtensionsFromFilePattern(fp string) []string {
	if fp == "" {
		return []string{".java", ".go", ".py", ".js", ".ts", ".cs", ".xml", ".yaml", ".yml", ".properties"}
	}
	ext := filepath.Ext(fp)
	if ext != "" {
		return []string{ext}
	}
	return []string{".java", ".go", ".py", ".js", ".ts", ".cs"}
}

func mergeExts(a, b []string) []string {
	seen := make(map[string]bool)
	for _, e := range a {
		seen[e] = true
	}
	for _, e := range b {
		seen[e] = true
	}
	var result []string
	for e := range seen {
		result = append(result, e)
	}
	sort.Strings(result)
	return result
}

func searchAppSource(appDir string, terms []string, extensions []string) (bool, []string) {
	extSet := make(map[string]bool, len(extensions))
	for _, e := range extensions {
		extSet[e] = true
	}

	lowerTerms := make([]string, len(terms))
	for i, t := range terms {
		lowerTerms[i] = strings.ToLower(t)
	}

	matched := make(map[string]bool)
	filepath.WalkDir(appDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(d.Name())
		if !extSet[ext] {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		content := strings.ToLower(string(data))
		for _, term := range lowerTerms {
			if strings.Contains(content, term) {
				rel, _ := filepath.Rel(appDir, path)
				if rel == "" {
					rel = filepath.Base(path)
				}
				matched[rel] = true
				break
			}
		}
		return nil
	})

	if len(matched) == 0 {
		return false, nil
	}
	var files []string
	for f := range matched {
		files = append(files, f)
	}
	sort.Strings(files)
	return true, files
}
