package feedback

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/verify"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
)

var timestampSuffix = regexp.MustCompile(`-(\d{8}-\d{6})$`)

type verifyFile struct {
	Results []verify.Result `json:"results"`
}

// DiscoverRuns finds output directories matching the *-YYYYMMDD-HHMMSS
// naming convention. If filter is non-empty, only directories whose name
// contains filter (case-insensitive) are included.
func DiscoverRuns(baseDir, filter string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("reading output directory %s: %w", baseDir, err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !timestampSuffix.MatchString(name) {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
			continue
		}
		dirs = append(dirs, filepath.Join(baseDir, name))
	}
	sort.Strings(dirs)
	return dirs, nil
}

// LoadRun reads structured artifacts from a single run directory and
// joins them into a RunSummary. Returns nil if the directory lacks
// report.yaml (minimum requirement).
func LoadRun(dir string) (*RunSummary, error) {
	reportPath := filepath.Join(dir, "report.yaml")
	report, err := workspace.ReadReport(reportPath)
	if err != nil {
		return nil, nil // skip runs without a report
	}

	ts := parseTimestamp(filepath.Base(dir))

	run := &RunSummary{
		Dir:              dir,
		Timestamp:        ts,
		Sources:          report.Sources,
		Targets:          report.Targets,
		RulesTotal:       report.RulesTotal,
		TestsPassed:      report.TestsPassed,
		TestsFailed:      report.TestsFailed,
		KantraLimitation: report.KantraLimitation,
		PassRate:         report.PassRate,
	}

	patternsPath := filepath.Join(dir, "patterns.json")
	extract, err := rules.ReadPatternsFile(patternsPath)
	if err != nil {
		return run, nil // report-only run is still useful
	}
	run.Language = extract.Language

	verifyResults := loadVerifyResults(filepath.Join(dir, "verify-results.json"))
	verifyByIndex := indexVerifyResults(verifyResults)

	reportByRuleID := make(map[string]string, len(report.Rules))
	for _, rs := range report.Rules {
		reportByRuleID[rs.RuleID] = rs.TestStatus
	}

	verifiedByRuleID := make(map[string]string, len(report.Rules))
	for _, rs := range report.Rules {
		verifiedByRuleID[rs.RuleID] = rs.SourceVerified
	}

	prefixGens := make(map[string]*rules.IDGenerator)

	for i, p := range extract.Patterns {
		concern := p.Concern
		if concern == "" {
			concern = "general"
		}
		changeType := rules.ChangeType(p.LocationType, p.ProviderType, p.DependencyName, p.XPath)
		prefix := rules.RuleIDPrefix(concern, changeType)
		if _, ok := prefixGens[prefix]; !ok {
			prefixGens[prefix] = rules.NewIDGenerator()
		}
		ruleID := prefixGens[prefix].Next(prefix)
		outcome := PatternOutcome{
			SourceFQN:    p.SourceFQN,
			LocationType: p.LocationType,
			ProviderType: p.ProviderType,
			Category:     p.Category,
			Complexity:   p.Complexity,
			Concern:      p.Concern,
			HasArtifact:  p.SourceArtifact != nil,
			IsDependency: p.DependencyName != "",
			IsXPath:      p.XPath != "",
		}

		if vr, ok := verifyByIndex[i]; ok {
			outcome.VerifyStatus = string(vr.Status)
			outcome.VerifyReason = vr.Reason
		}

		if ts, ok := reportByRuleID[ruleID]; ok {
			outcome.TestStatus = ts
		}

		if sv, ok := verifiedByRuleID[ruleID]; ok && outcome.VerifyStatus == "" {
			outcome.VerifyStatus = sv
		}

		run.Patterns = append(run.Patterns, outcome)
	}

	return run, nil
}

// LoadRuns loads all discovered run directories, skipping any that
// lack the minimum artifacts.
func LoadRuns(dirs []string) []*RunSummary {
	var runs []*RunSummary
	for _, dir := range dirs {
		run, _ := LoadRun(dir)
		if run != nil {
			runs = append(runs, run)
		}
	}
	return runs
}

func parseTimestamp(name string) time.Time {
	m := timestampSuffix.FindStringSubmatch(name)
	if len(m) < 2 {
		return time.Time{}
	}
	t, _ := time.Parse("20060102-150405", m[1])
	return t
}

func loadVerifyResults(path string) []verify.Result {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var vf verifyFile
	if err := json.Unmarshal(data, &vf); err != nil {
		return nil
	}
	return vf.Results
}

func indexVerifyResults(results []verify.Result) map[int]verify.Result {
	m := make(map[int]verify.Result, len(results))
	for _, r := range results {
		m[r.PatternIndex] = r
	}
	return m
}

