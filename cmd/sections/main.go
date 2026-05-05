package main

import (
	"flag"
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
		cli.Fail("invalid_arguments", "--guide is required", "sections", "set --guide to a migration guide markdown path", nil)
	}

	data, err := os.ReadFile(*guide)
	if err != nil {
		cli.Fail("read_guide_failed", err.Error(), "sections", "verify guide path and read permissions", map[string]string{"guide": *guide})
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
