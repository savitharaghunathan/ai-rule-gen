package main

import (
	"flag"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/coverage"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	guide := flag.String("guide", "", "Path to migration guide markdown (required)")
	patterns := flag.String("patterns", "", "Path to patterns.json (required)")
	language := flag.String("language", "", "Language for artifact detection (java, go). Auto-detected from patterns.json if omitted")
	flag.Parse()

	if *guide == "" || *patterns == "" {
		cli.Fail("invalid_arguments", "--guide and --patterns are required", "coverage", "provide both guide markdown and patterns.json paths", nil)
	}

	guideData, err := os.ReadFile(*guide)
	if err != nil {
		cli.Fail("read_guide_failed", err.Error(), "coverage", "verify guide path and read permissions", map[string]string{"guide": *guide})
	}

	extractOutput, err := rules.ReadPatternsFile(*patterns)
	if err != nil {
		cli.Fail("read_patterns_failed", err.Error(), "coverage", "verify patterns path and JSON format", map[string]string{"patterns": *patterns})
	}

	lang := *language
	if lang == "" {
		lang = extractOutput.Language
	}

	sections := coverage.ParseSections(string(guideData))
	scanner := coverage.NewScanner(lang)
	result := coverage.CheckCoverage(sections, scanner, extractOutput)

	cli.WriteJSON(result)
}
