package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/verify"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	patternsPath := flag.String("patterns", "", "Path to patterns.json (required)")
	language := flag.String("language", "", "Override language (default: read from patterns.json)")
	outputPath := flag.String("output", "", "Write verification results to this JSON file")
	cacheDir := flag.String("cache", "", "Directory for cached JARs/artifacts (default: <patterns-dir>/verify-cache)")
	flag.Parse()

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	if *patternsPath == "" {
		cli.Fail("invalid_arguments", "--patterns is required", "verify", "set --patterns to a patterns.json file", nil)
	}

	extract, err := rules.ReadPatternsFile(*patternsPath)
	if err != nil {
		cli.Fail("read_patterns_failed", err.Error(), "verify", "check patterns.json path and format", map[string]string{"patterns": *patternsPath})
	}

	lang := *language
	if lang == "" {
		lang = extract.Language
	}
	if lang != "" {
		extract.Language = lang
	}

	cache := *cacheDir
	if cache == "" {
		dir := filepath.Dir(*patternsPath)
		cache = filepath.Join(dir, "verify-cache")
	}

	cli.Log("verifying %d patterns for language %q", len(extract.Patterns), extract.Language)

	results, err := verify.Run(extract, cache)
	if err != nil {
		cli.Fail("verify_failed", err.Error(), "verify", "check network connectivity and jar command availability", nil)
	}

	summary := verify.Summarize(results)

	cli.Log("verification complete: %d verified, %d not_found, %d skipped, %d offline",
		summary.Verified, summary.NotFound, summary.Skipped, summary.Offline)

	if *outputPath != "" {
		output := struct {
			Results []verify.Result  `json:"results"`
			Summary *verify.Summary `json:"summary"`
		}{
			Results: results,
			Summary: summary,
		}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			cli.Fail("marshal_failed", err.Error(), "verify", "unexpected marshal error", nil)
		}
		if err := os.WriteFile(*outputPath, data, 0o644); err != nil {
			cli.Fail("write_failed", err.Error(), "verify", "check output path permissions", map[string]string{"output": *outputPath})
		}
	}

	cli.WriteJSON(map[string]interface{}{
		"status":    "ok",
		"verified":  summary.Verified,
		"not_found": summary.NotFound,
		"skipped":   summary.Skipped,
		"offline":   summary.Offline,
	})
}
