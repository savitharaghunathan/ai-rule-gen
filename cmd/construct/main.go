package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/construct"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

func main() {
	patterns := flag.String("patterns", "", "Path to patterns.json (required)")
	output := flag.String("output", "", "Output directory for rule files (required)")
	flag.Parse()

	if *patterns == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "error: --patterns and --output are required")
		os.Exit(1)
	}

	extract, err := rules.ReadPatternsFile(*patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result, err := construct.Run(extract, *output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cli.WriteJSON(result)
}
