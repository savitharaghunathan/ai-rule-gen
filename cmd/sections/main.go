package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/coverage"
)

type output struct {
	Total           int                `json:"total"`
	ContentSections int                `json:"content_sections"`
	HeaderOnly      int                `json:"header_only"`
	Sections        []coverage.Section `json:"sections"`
}

func main() {
	guide := flag.String("guide", "", "Path to migration guide markdown (required)")
	flag.Parse()

	if *guide == "" {
		fmt.Fprintln(os.Stderr, "error: --guide is required")
		os.Exit(1)
	}

	data, err := os.ReadFile(*guide)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading guide: %v\n", err)
		os.Exit(1)
	}

	sections := coverage.ParseSections(string(data))
	coverage.ClassifySections(sections)

	contentCount := 0
	headerCount := 0
	for _, s := range sections {
		if s.Type == "content" {
			contentCount++
		} else {
			headerCount++
		}
	}

	cli.WriteJSON(output{
		Total:           len(sections),
		ContentSections: contentCount,
		HeaderOnly:      headerCount,
		Sections:        sections,
	})
}
