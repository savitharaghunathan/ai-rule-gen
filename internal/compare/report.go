package compare

import (
	"fmt"
	"io"
	"strings"
)

// WriteMarkdown formats the comparison as a human-readable report.
func WriteMarkdown(w io.Writer, r *Result) error {
	bw := &errWriter{w: w}

	fmt.Fprintf(bw, "# Ruleset comparison: %s vs %s\n\n", r.NameA, r.NameB)
	fmt.Fprintf(bw, "- **A**: %s (%d rules) — `%s`\n", r.NameA, r.RuleCountA, r.RulesDirA)
	fmt.Fprintf(bw, "- **B**: %s (%d rules) — `%s`\n\n", r.NameB, r.RuleCountB, r.RulesDirB)

	writeMatrixSection(bw, r)

	if r.KantraDiff != nil {
		writeKantraSection(bw, r)
	}

	return bw.err
}

func writeMatrixSection(w io.Writer, r *Result) {
	s := r.Matrix.Summary
	fmt.Fprintf(w, "## Coverage matrix\n\n")
	fmt.Fprintf(w, "How many rules on one side are matched by a rule keyed on the same API on the other side.\n\n")
	fmt.Fprintf(w, "| Direction | Covered | Partial | Missing |\n|---|---|---|---|\n")
	fmt.Fprintf(w, "| A → B (%s rules covered by %s) | %d | %d | %d |\n", r.NameA, r.NameB, s.AInBCovered, s.AInBPartial, s.AInBMissing)
	fmt.Fprintf(w, "| B → A (%s rules covered by %s) | %d | %d | %d |\n\n", r.NameB, r.NameA, s.BInACovered, s.BInAPartial, s.BInAMissing)

	writeMissingList(w, fmt.Sprintf("Rules in %s with no equivalent in %s", r.NameA, r.NameB), r.Matrix.AInB)
	writeMissingList(w, fmt.Sprintf("Rules in %s with no equivalent in %s", r.NameB, r.NameA), r.Matrix.BInA)
}

func writeMissingList(w io.Writer, header string, cov []RuleCoverage) {
	var missing []RuleCoverage
	for _, c := range cov {
		if c.Status == "missing" {
			missing = append(missing, c)
		}
	}
	if len(missing) == 0 {
		return
	}
	fmt.Fprintf(w, "### %s (%d)\n\n", header, len(missing))
	for _, c := range missing {
		desc := c.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Fprintf(w, "- `%s` — %s\n", c.RuleID, truncate(desc, 120))
		if len(c.MatchKeys) > 0 {
			fmt.Fprintf(w, "  - keys: %s\n", strings.Join(truncList(c.MatchKeys, 3), ", "))
		}
	}
	fmt.Fprintln(w)
}

func writeKantraSection(w io.Writer, r *Result) {
	k := r.KantraDiff
	fmt.Fprintf(w, "## Kantra diff (app: %s)\n\n", k.AppDir)
	fmt.Fprintf(w, "| | %s | %s |\n|---|---|---|\n", r.NameA, r.NameB)
	fmt.Fprintf(w, "| Rules fired | %d | %d |\n", k.RulesFiredA, k.RulesFiredB)
	fmt.Fprintf(w, "| Incidents | %d | %d |\n", k.IncidentsA, k.IncidentsB)
	fmt.Fprintf(w, "| Files flagged only here | %d | %d |\n", len(k.FilesAOnly), len(k.FilesBOnly))
	fmt.Fprintf(w, "| Files flagged by both | %d | %d |\n\n", len(k.FilesBoth), len(k.FilesBoth))

	if len(k.FilesAOnly) > 0 {
		fmt.Fprintf(w, "### Files flagged only by %s (%d)\n\n", r.NameA, len(k.FilesAOnly))
		for _, f := range truncList(k.FilesAOnly, 50) {
			fmt.Fprintf(w, "- `%s`\n", f)
		}
		fmt.Fprintln(w)
	}
	if len(k.FilesBOnly) > 0 {
		fmt.Fprintf(w, "### Files flagged only by %s (%d)\n\n", r.NameB, len(k.FilesBOnly))
		for _, f := range truncList(k.FilesBOnly, 50) {
			fmt.Fprintf(w, "- `%s`\n", f)
		}
		fmt.Fprintln(w)
	}
	if len(k.FilesBoth) > 0 {
		fmt.Fprintf(w, "### Files flagged by both (%d)\n\n", len(k.FilesBoth))
		limit := 25
		for i, f := range k.FilesBoth {
			if i >= limit {
				fmt.Fprintf(w, "- _(…%d more omitted)_\n", len(k.FilesBoth)-limit)
				break
			}
			fmt.Fprintf(w, "- `%s` — A: %v · B: %v\n", f.File, truncList(f.RulesA, 3), truncList(f.RulesB, 3))
		}
		fmt.Fprintln(w)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func truncList(in []string, n int) []string {
	if len(in) <= n {
		return in
	}
	out := make([]string, n+1)
	copy(out, in[:n])
	out[n] = fmt.Sprintf("…+%d", len(in)-n)
	return out
}

type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) Write(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	n, err := e.w.Write(p)
	if err != nil {
		e.err = err
	}
	return n, err
}
