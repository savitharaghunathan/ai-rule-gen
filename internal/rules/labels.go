package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// StampTestResults updates rule files with pass/fail/kantra-limitation labels based on kantra results.
func StampTestResults(rulesDir string, passed, failed, kantraLimitation []string) error {
	passedSet := make(map[string]bool)
	for _, id := range passed {
		passedSet[id] = true
	}
	failedSet := make(map[string]bool)
	for _, id := range failed {
		failedSet[id] = true
	}
	kantraLimitationSet := make(map[string]bool)
	for _, id := range kantraLimitation {
		kantraLimitationSet[id] = true
	}

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return fmt.Errorf("reading rules dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if name == "ruleset.yaml" || name == "ruleset.yml" {
			continue
		}

		path := filepath.Join(rulesDir, name)
		ruleList, err := ReadRulesFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}

		modified := false
		for i := range ruleList {
			r := &ruleList[i]
			var newLabel string
			if kantraLimitationSet[r.RuleID] {
				newLabel = "konveyor.io/test-result=kantra-limitation"
			} else if passedSet[r.RuleID] {
				newLabel = "konveyor.io/test-result=passed"
			} else if failedSet[r.RuleID] {
				newLabel = "konveyor.io/test-result=failed"
			} else {
				continue
			}

			updated := false
			for j, l := range r.Labels {
				if strings.HasPrefix(l, "konveyor.io/test-result=") {
					r.Labels[j] = newLabel
					updated = true
					break
				}
			}
			if !updated {
				r.Labels = append(r.Labels, newLabel)
			}
			modified = true
		}

		if modified {
			if err := WriteRulesFile(path, ruleList); err != nil {
				return fmt.Errorf("writing %s: %w", name, err)
			}
		}
	}
	return nil
}

// StampVerificationResults updates rule files with source-verified labels.
func StampVerificationResults(rulesDir string, verified, notFound []string) error {
	verifiedSet := make(map[string]bool)
	for _, id := range verified {
		verifiedSet[id] = true
	}
	notFoundSet := make(map[string]bool)
	for _, id := range notFound {
		notFoundSet[id] = true
	}

	entries, err := os.ReadDir(rulesDir)
	if err != nil {
		return fmt.Errorf("reading rules dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if name == "ruleset.yaml" || name == "ruleset.yml" {
			continue
		}

		path := filepath.Join(rulesDir, name)
		ruleList, err := ReadRulesFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}

		modified := false
		for i := range ruleList {
			r := &ruleList[i]
			var newLabel string
			if verifiedSet[r.RuleID] {
				newLabel = "konveyor.io/source-verified=true"
			} else if notFoundSet[r.RuleID] {
				newLabel = "konveyor.io/source-verified=false"
			} else {
				continue
			}

			updated := false
			for j, l := range r.Labels {
				if strings.HasPrefix(l, "konveyor.io/source-verified=") {
					r.Labels[j] = newLabel
					updated = true
					break
				}
			}
			if !updated {
				r.Labels = append(r.Labels, newLabel)
			}
			modified = true
		}

		if modified {
			if err := WriteRulesFile(path, ruleList); err != nil {
				return fmt.Errorf("writing %s: %w", name, err)
			}
		}
	}
	return nil
}
