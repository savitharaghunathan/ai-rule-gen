package compare

import (
	"fmt"
	"os"
	"sort"

	"github.com/konveyor/ai-rule-gen/internal/eval"
)

// RunKantraDiff runs kantra-analyze on appDir with each ruleset and diffs the results.
func RunKantraDiff(rulesDirA, rulesDirB, appDir string) (*KantraDiff, error) {
	outA, err := os.MkdirTemp("", "compare-kantra-a-*")
	if err != nil {
		return nil, fmt.Errorf("temp dir A: %w", err)
	}
	defer os.RemoveAll(outA)

	outB, err := os.MkdirTemp("", "compare-kantra-b-*")
	if err != nil {
		return nil, fmt.Errorf("temp dir B: %w", err)
	}
	defer os.RemoveAll(outB)

	fmt.Fprintf(os.Stderr, "[compare] running kantra on ruleset A: %s\n", rulesDirA)
	covA, err := eval.RunKantraAnalyze(rulesDirA, appDir, outA)
	if err != nil {
		return nil, fmt.Errorf("kantra A: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[compare] running kantra on ruleset B: %s\n", rulesDirB)
	covB, err := eval.RunKantraAnalyze(rulesDirB, appDir, outB)
	if err != nil {
		return nil, fmt.Errorf("kantra B: %w", err)
	}

	return buildKantraDiff(appDir, covA, covB), nil
}

func buildKantraDiff(appDir string, a, b *eval.AppCoverage) *KantraDiff {
	filesA := filesByRule(a)
	filesB := filesByRule(b)

	allFiles := map[string]struct{}{}
	for f := range filesA {
		allFiles[f] = struct{}{}
	}
	for f := range filesB {
		allFiles[f] = struct{}{}
	}

	var aOnly, bOnly []string
	var both []FileFinding
	for f := range allFiles {
		_, inA := filesA[f]
		_, inB := filesB[f]
		switch {
		case inA && inB:
			both = append(both, FileFinding{
				File:   f,
				RulesA: sortedSliceFromMap(filesA[f]),
				RulesB: sortedSliceFromMap(filesB[f]),
			})
		case inA:
			aOnly = append(aOnly, f)
		case inB:
			bOnly = append(bOnly, f)
		}
	}
	sort.Strings(aOnly)
	sort.Strings(bOnly)
	sort.Slice(both, func(i, j int) bool { return both[i].File < both[j].File })

	return &KantraDiff{
		AppDir:          appDir,
		RulesFiredA:     a.RulesFired,
		RulesFiredB:     b.RulesFired,
		IncidentsA:      a.TotalIncidents,
		IncidentsB:      b.TotalIncidents,
		FilesAOnly:      aOnly,
		FilesBOnly:      bOnly,
		FilesBoth:       both,
		RulesFiredOnlyA: rulesNotInOther(a, b),
		RulesFiredOnlyB: rulesNotInOther(b, a),
	}
}

// filesByRule inverts violations to file → ruleIDs that flagged it.
func filesByRule(c *eval.AppCoverage) map[string]map[string]bool {
	out := make(map[string]map[string]bool)
	for ruleID, v := range c.Violations {
		for _, f := range v.Files {
			set, ok := out[f]
			if !ok {
				set = map[string]bool{}
				out[f] = set
			}
			set[ruleID] = true
		}
	}
	return out
}

func rulesNotInOther(a, b *eval.AppCoverage) []string {
	var out []string
	for ruleID := range a.Violations {
		if _, ok := b.Violations[ruleID]; !ok {
			out = append(out, ruleID)
		}
	}
	sort.Strings(out)
	return out
}

func sortedSliceFromMap(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
