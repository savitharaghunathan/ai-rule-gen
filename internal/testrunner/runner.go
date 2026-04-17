package testrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/konveyor/ai-rule-gen/internal/kantraparser"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// Result is the JSON output of a test run.
type Result struct {
	Passed      int      `json:"passed"`
	Failed      int      `json:"failed"`
	Total       int      `json:"total"`
	PassRate    float64  `json:"pass_rate"`
	FailedRules []string `json:"failed_rules,omitempty"`
	Output      string   `json:"output,omitempty"`
}

// Config holds the inputs for a test run.
type Config struct {
	RulesDir string
	TestsDir string
	Files    []string // specific test files to run (if empty, scan TestsDir)
}

// Run executes kantra test for each .test.yaml file and returns pass/fail results.
// It does NOT stamp rules or generate reports — the caller handles those.
func Run(cfg Config) (*Result, error) {
	if err := checkKantra(); err != nil {
		return nil, err
	}

	testFiles := cfg.Files
	if len(testFiles) == 0 {
		var err error
		testFiles, err = FindTestFiles(cfg.TestsDir)
		if err != nil {
			return nil, fmt.Errorf("finding test files: %w", err)
		}
	} else {
		// Resolve bare filenames relative to TestsDir.
		for i, f := range testFiles {
			if !filepath.IsAbs(f) && !strings.Contains(f, string(filepath.Separator)) {
				testFiles[i] = filepath.Join(cfg.TestsDir, f)
			}
		}
	}
	if len(testFiles) == 0 {
		return nil, fmt.Errorf("no .test.yaml files found in %s", cfg.TestsDir)
	}

	// Run kantra test per file sequentially, collect output.
	// Track groups that error out (e.g., "unable to get build tool") so their
	// rules are correctly marked as failed rather than silently passing.
	var allOutput strings.Builder
	var erroredRuleIDs []string
	for _, tf := range testFiles {
		out, err := RunKantraTest(tf)
		allOutput.WriteString(out)
		allOutput.WriteString("\n")
		if err != nil {
			if out == "" {
				return nil, fmt.Errorf("kantra test %s: %w", filepath.Base(tf), err)
			}
			// Only mark all rules as errored if kantra failed before producing
			// any test results (e.g., "unable to get build tool"). If kantra
			// produced a Rules Summary, per-rule parsing handles pass/fail.
			if _, total := kantraparser.ParseSummary(out); total == 0 {
				if ids, parseErr := kantraparser.TestFileRuleIDs(tf); parseErr == nil {
					erroredRuleIDs = append(erroredRuleIDs, ids...)
				}
			}
		}
	}
	combinedOutput := allOutput.String()

	// Write combined kantra output.
	outputPath := filepath.Join(cfg.TestsDir, "kantra-output.txt")
	if err := os.WriteFile(outputPath, []byte(combinedOutput), 0o644); err != nil {
		return nil, fmt.Errorf("writing kantra output: %w", err)
	}

	// Determine which rule IDs to check: when running a subset of test files,
	// only check rules referenced by those files (not all rules in the directory).
	var checkIDs []string
	if len(cfg.Files) > 0 {
		seen := make(map[string]bool)
		for _, tf := range testFiles {
			ids, err := kantraparser.TestFileRuleIDs(tf)
			if err != nil {
				return nil, fmt.Errorf("parsing test file %s: %w", filepath.Base(tf), err)
			}
			for _, id := range ids {
				if !seen[id] {
					seen[id] = true
					checkIDs = append(checkIDs, id)
				}
			}
		}
	} else {
		allRules, err := rules.ReadRulesDir(cfg.RulesDir)
		if err != nil {
			return nil, fmt.Errorf("reading rules: %w", err)
		}
		for _, r := range allRules {
			checkIDs = append(checkIDs, r.RuleID)
		}
	}

	passedIDs, failedIDs := kantraparser.PassedAndFailed(combinedOutput, checkIDs, erroredRuleIDs...)

	passed := len(passedIDs)
	failed := len(failedIDs)
	total := passed + failed
	var passRate float64
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	return &Result{
		Passed:      passed,
		Failed:      failed,
		Total:       total,
		PassRate:    passRate,
		FailedRules: failedIDs,
		Output:      combinedOutput,
	}, nil
}

// FindTestFiles recursively finds .test.yaml/.test.yml files under dir.
func FindTestFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".test.yaml") || strings.HasSuffix(d.Name(), ".test.yml") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// RunKantraTest executes kantra test on a single test file and returns the combined output.
func RunKantraTest(testFile string) (string, error) {
	cmd := exec.Command("kantra", "test", testFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}
	return output, err
}

// ReadSourceTarget extracts source and target from ruleset.yaml labels.
func ReadSourceTarget(rulesDir string) (source, target string) {
	rsPath := filepath.Join(rulesDir, "ruleset.yaml")
	rs, err := rules.ReadRuleset(rsPath)
	if err != nil {
		return "", ""
	}
	for _, l := range rs.Labels {
		if v, ok := strings.CutPrefix(l, "konveyor.io/source="); ok {
			source = v
		}
		if v, ok := strings.CutPrefix(l, "konveyor.io/target="); ok {
			target = v
		}
	}
	return source, target
}

// checkKantra verifies that kantra is installed and in PATH.
func checkKantra() error {
	_, err := exec.LookPath("kantra")
	if err != nil {
		return fmt.Errorf("kantra not found in PATH: install kantra before running tests")
	}
	return nil
}
