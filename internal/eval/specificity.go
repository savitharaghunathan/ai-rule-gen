package eval

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/rules"
)

var importRe = regexp.MustCompile(`^\s*import\s+(?:static\s+)?([a-zA-Z0-9_.]+)\s*;`)

type broadRule struct {
	ruleID string
	prefix string
}

// DetectSpecificityGaps finds imports in the app that are only covered by
// broad PACKAGE-level rules but have no dedicated IMPORT/TYPE/METHOD_CALL rule.
func DetectSpecificityGaps(ruleList []rules.Rule, cov *AppCoverage, appDir string) []SpecificityGap {
	if appDir == "" || cov == nil {
		return nil
	}

	broad := findBroadRules(ruleList)
	if len(broad) == 0 {
		return nil
	}

	specificPrefixes := buildSpecificCoverage(ruleList)

	appImports := scanAppImports(appDir)
	if len(appImports) == 0 {
		return nil
	}

	gapMap := make(map[string]*SpecificityGap)
	for fqn, files := range appImports {
		broadID := matchesBroadRule(fqn, broad)
		if broadID == "" {
			continue
		}
		if coveredBySpecific(fqn, specificPrefixes) {
			continue
		}
		if g, ok := gapMap[fqn]; ok {
			g.AppFiles = mergeFiles(g.AppFiles, files)
		} else {
			gapMap[fqn] = &SpecificityGap{
				BroadRuleID: broadID,
				ImportFQN:   fqn,
				AppFiles:    files,
			}
		}
	}

	var gaps []SpecificityGap
	for _, g := range gapMap {
		sort.Strings(g.AppFiles)
		gaps = append(gaps, *g)
	}
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].ImportFQN < gaps[j].ImportFQN
	})
	return gaps
}

func findBroadRules(ruleList []rules.Rule) []broadRule {
	var broad []broadRule
	for _, r := range ruleList {
		loc, pattern := extractJavaLocation(r.When)
		if loc == rules.LocationPackage && pattern != "" {
			broad = append(broad, broadRule{ruleID: r.RuleID, prefix: pattern})
		}
	}
	return broad
}

func buildSpecificCoverage(ruleList []rules.Rule) []string {
	var prefixes []string
	for _, r := range ruleList {
		pats := extractAllJavaPatterns(r.When)
		for _, lp := range pats {
			if lp.loc != rules.LocationPackage && lp.loc != "" && lp.pattern != "" {
				prefixes = append(prefixes, lp.pattern)
			}
		}
	}
	return prefixes
}

type locPattern struct {
	loc     string
	pattern string
}

func extractAllJavaPatterns(cond rules.Condition) []locPattern {
	var result []locPattern
	if cond.JavaReferenced != nil {
		result = append(result, locPattern{cond.JavaReferenced.Location, cond.JavaReferenced.Pattern})
	}
	for _, entry := range cond.Or {
		result = append(result, extractAllJavaPatterns(entry.Condition)...)
	}
	for _, entry := range cond.And {
		result = append(result, extractAllJavaPatterns(entry.Condition)...)
	}
	return result
}

func extractJavaLocation(cond rules.Condition) (string, string) {
	if cond.JavaReferenced != nil {
		return cond.JavaReferenced.Location, cond.JavaReferenced.Pattern
	}
	if len(cond.Or) > 0 {
		for _, entry := range cond.Or {
			if loc, pat := extractJavaLocation(entry.Condition); loc != "" {
				return loc, pat
			}
		}
	}
	if len(cond.And) > 0 {
		for _, entry := range cond.And {
			if loc, pat := extractJavaLocation(entry.Condition); loc != "" {
				return loc, pat
			}
		}
	}
	return "", ""
}

func matchesBroadRule(fqn string, broad []broadRule) string {
	for _, b := range broad {
		if strings.HasPrefix(fqn, b.prefix) {
			return b.ruleID
		}
	}
	return ""
}

func coveredBySpecific(fqn string, specificPrefixes []string) bool {
	for _, sp := range specificPrefixes {
		if strings.HasPrefix(fqn, sp) || strings.HasPrefix(sp, fqn) {
			return true
		}
	}
	return false
}

// scanAppImports walks .java files and extracts import FQNs.
// Returns map[importFQN][]relativeFilePaths.
func scanAppImports(appDir string) map[string][]string {
	result := make(map[string][]string)

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
		if filepath.Ext(d.Name()) != ".java" {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		rel, _ := filepath.Rel(appDir, path)
		if rel == "" {
			rel = filepath.Base(path)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if m := importRe.FindStringSubmatch(line); len(m) > 1 {
				fqn := m[1]
				result[fqn] = append(result[fqn], rel)
			}
		}
		return nil
	})

	return result
}

func mergeFiles(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for _, f := range a {
		seen[f] = true
	}
	for _, f := range b {
		seen[f] = true
	}
	var merged []string
	for f := range seen {
		merged = append(merged, f)
	}
	return merged
}
