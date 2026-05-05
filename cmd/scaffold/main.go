package main

import (
	"flag"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/scaffold"
)

func main() {
	rulesDir := flag.String("rules", "", "Path to rules directory (required)")
	output := flag.String("output", "", "Path to tests output directory (required)")
	language := flag.String("language", "", "Programming language (auto-detected if omitted)")
	languagesDir := flag.String("languages-dir", "languages", "Path to languages/ config directory")
	flag.Parse()

	if *rulesDir == "" || *output == "" {
		cli.Fail("invalid_arguments", "--rules and --output are required", "scaffold", "provide rules directory and tests output directory", nil)
	}

	result, err := scaffold.Run(*rulesDir, *output, *language, *languagesDir)
	if err != nil {
		cli.Fail("scaffold_failed", err.Error(), "scaffold", "verify rules directory and language config path", map[string]string{"rules": *rulesDir, "output": *output, "languages_dir": *languagesDir})
	}

	cli.WriteJSON(result)
}
