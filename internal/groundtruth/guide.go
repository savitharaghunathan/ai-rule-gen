package groundtruth

import (
	"os"
	"regexp"
	"sort"
	"time"
)

var (
	fqnRe       = regexp.MustCompile(`\b([a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*){2,}\.[A-Z][A-Za-z0-9]*)\b`)
	importRe    = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([a-zA-Z0-9_.]+)\s*;`)
	packageDotC = regexp.MustCompile(`\b([a-z][a-z0-9]*(?:\.[a-z][a-z0-9]*)+)\.[A-Z]`)
)

// ExtractFromGuide reads an ingested migration guide markdown file and extracts
// Java FQNs as ground truth entries. It finds:
// - Full FQNs like org.apache.http.client.methods.HttpGet
// - Import statements in code blocks
func ExtractFromGuide(guidePath string) ([]Entry, error) {
	data, err := os.ReadFile(guidePath)
	if err != nil {
		return nil, err
	}

	text := string(data)
	seen := make(map[string]bool)
	var fqns []string

	for _, m := range fqnRe.FindAllStringSubmatch(text, -1) {
		fqn := m[1]
		if !seen[fqn] {
			seen[fqn] = true
			fqns = append(fqns, fqn)
		}
	}

	for _, m := range importRe.FindAllStringSubmatch(text, -1) {
		fqn := m[1]
		if !seen[fqn] {
			seen[fqn] = true
			fqns = append(fqns, fqn)
		}
	}

	sort.Strings(fqns)

	today := time.Now().Format("2006-01-02")
	entries := make([]Entry, 0, len(fqns))
	for _, fqn := range fqns {
		entries = append(entries, Entry{
			OldAPI:       fqn,
			ActionType:   "package_change",
			Severity:     "high",
			ReviewedBy:   "guide-extract",
			ReviewedDate: today,
		})
	}

	return entries, nil
}
