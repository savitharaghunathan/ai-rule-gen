package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

type output struct {
	InputFiles    int `json:"input_files"`
	TotalPatterns int `json:"total_patterns"`
	Merged        int `json:"merged"`
	Duplicates    int `json:"duplicates"`
}

func main() {
	outFile := flag.String("output", "patterns.json", "Output path for merged patterns")
	flag.Parse()

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "error: provide one or more patterns files as arguments")
		os.Exit(1)
	}

	var parts []*rules.ExtractOutput
	totalPatterns := 0
	for _, f := range files {
		p, err := rules.ReadPatternsFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		totalPatterns += len(p.Patterns)
		parts = append(parts, p)
	}

	merged := rules.MergePatterns(parts)
	if err := rules.WritePatternsFile(*outFile, merged); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cli.WriteJSON(output{
		InputFiles:    len(files),
		TotalPatterns: totalPatterns,
		Merged:        len(merged.Patterns),
		Duplicates:    totalPatterns - len(merged.Patterns),
	})
}
