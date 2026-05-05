package main

import (
	"flag"

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
		cli.Fail("invalid_arguments", "provide one or more patterns files as arguments", "merge-patterns", "pass input patterns files after flags", nil)
	}

	var parts []*rules.ExtractOutput
	totalPatterns := 0
	for _, f := range files {
		p, err := rules.ReadPatternsFile(f)
		if err != nil {
			cli.Fail("read_patterns_failed", err.Error(), "merge-patterns", "verify each input patterns file path and JSON format", map[string]string{"file": f})
		}
		totalPatterns += len(p.Patterns)
		parts = append(parts, p)
	}

	merged := rules.MergePatterns(parts)
	if err := rules.WritePatternsFile(*outFile, merged); err != nil {
		cli.Fail("write_patterns_failed", err.Error(), "merge-patterns", "verify output path permissions", map[string]string{"output": *outFile})
	}

	cli.WriteJSON(output{
		InputFiles:    len(files),
		TotalPatterns: totalPatterns,
		Merged:        len(merged.Patterns),
		Duplicates:    totalPatterns - len(merged.Patterns),
	})
}
