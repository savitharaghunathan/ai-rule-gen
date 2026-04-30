package main

import (
	"flag"
	"fmt"
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
		fmt.Fprintln(os.Stderr, "error: --guide and --patterns are required")
		os.Exit(1)
	}

	guideData, err := os.ReadFile(*guide)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading guide: %v\n", err)
		os.Exit(1)
	}

	extractOutput, err := rules.ReadPatternsFile(*patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading patterns: %v\n", err)
		os.Exit(1)
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
