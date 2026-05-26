package eval

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/rules"
	"gopkg.in/yaml.v3"
)

// kantraRuleset matches the top-level structure of kantra analyze output.yaml.
type kantraRuleset struct {
	Name       string                      `yaml:"name"`
	Violations map[string]kantraViolation  `yaml:"violations"`
}

type kantraViolation struct {
	Incidents []kantraIncident `yaml:"incidents"`
}

type kantraIncident struct {
	URI string `yaml:"uri"`
}

// RunKantraAnalyze runs kantra analyze against an app with the given rules.
func RunKantraAnalyze(rulesDir, appDir, outputDir string) (*AppCoverage, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kantra", "analyze",
		"-i", appDir,
		"--rules", rulesDir,
		"--enable-default-rulesets=false",
		"--run-local",
		"-o", outputDir,
		"--overwrite",
		"--no-progress",
	)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("kantra analyze timed out after 5 minutes")
		}
		return nil, fmt.Errorf("kantra analyze: %w", err)
	}

	return parseAnalyzeOutput(filepath.Join(outputDir, "output.yaml"))
}

func parseAnalyzeOutput(path string) (*AppCoverage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading kantra output %s: %w", path, err)
	}

	var rulesets []kantraRuleset
	if err := yaml.Unmarshal(data, &rulesets); err != nil {
		return nil, fmt.Errorf("parsing kantra output %s: %w", path, err)
	}

	violations := make(map[string]Violation)
	totalIncidents := 0

	for _, rs := range rulesets {
		for ruleID, v := range rs.Violations {
			files := make(map[string]bool)
			for _, inc := range v.Incidents {
				fname := filepath.Base(inc.URI)
				files[fname] = true
			}

			var fileList []string
			for f := range files {
				fileList = append(fileList, f)
			}
			sort.Strings(fileList)

			violations[ruleID] = Violation{
				Incidents: len(v.Incidents),
				Files:     fileList,
			}
			totalIncidents += len(v.Incidents)
		}
	}

	return &AppCoverage{
		RulesFired:     len(violations),
		TotalIncidents: totalIncidents,
		Violations:     violations,
	}, nil
}

// FillNotFired sets the NotFired list by comparing loaded rules against violations.
func FillNotFired(cov *AppCoverage, ruleList []rules.Rule) {
	cov.TotalRules = len(ruleList)
	for _, r := range ruleList {
		if _, ok := cov.Violations[r.RuleID]; !ok {
			cov.NotFired = append(cov.NotFired, r.RuleID)
		}
	}
	sort.Strings(cov.NotFired)
}
