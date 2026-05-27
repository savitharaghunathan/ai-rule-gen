package eval

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
// appDir is used to compute relative file paths from kantra incident URIs.
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
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("kantra analyze timed out after 5 minutes")
		}
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("kantra analyze: %w\n%s", err, errMsg)
		}
		return nil, fmt.Errorf("kantra analyze: %w", err)
	}

	absApp, _ := filepath.Abs(appDir)
	return parseAnalyzeOutput(filepath.Join(outputDir, "output.yaml"), absApp)
}

func parseAnalyzeOutput(path, appDir string) (*AppCoverage, error) {
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
				fname := relativeFromURI(inc.URI, appDir)
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

// relativeFromURI extracts a relative file path from a kantra incident URI.
// Kantra URIs are typically file:///absolute/path/to/File.java. If appDir is
// provided, the result is relative to it; otherwise the full path is returned.
func relativeFromURI(uri, appDir string) string {
	p := uri
	if u, err := url.Parse(uri); err == nil && u.Scheme == "file" {
		p = u.Path
	}
	if appDir != "" {
		if rel, err := filepath.Rel(appDir, p); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return p
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
