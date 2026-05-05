package main

import (
	"flag"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/construct"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	patterns := flag.String("patterns", "", "Path to patterns.json (required)")
	output := flag.String("output", "", "Output directory for rule files (required)")
	flag.Parse()

	if *patterns == "" || *output == "" {
		cli.Fail(
			"invalid_arguments",
			"--patterns and --output are required",
			"construct",
			"provide both flags and retry",
			nil,
		)
	}

	extract, err := rules.ReadPatternsFile(*patterns)
	if err != nil {
		cli.Fail("read_patterns_failed", err.Error(), "construct", "verify --patterns path and JSON format", map[string]string{"patterns": *patterns})
	}

	result, err := construct.Run(extract, *output)
	if err != nil {
		cli.Fail("construct_failed", err.Error(), "construct", "inspect patterns content and output path permissions", map[string]string{"output": *output})
	}

	cli.WriteJSON(result)
}
