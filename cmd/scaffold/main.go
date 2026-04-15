package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/scaffold"
)

func main() {
	rulesDir := flag.String("rules", "", "Path to rules directory (required)")
	output := flag.String("output", "", "Output directory (required)")
	language := flag.String("language", "", "Programming language (auto-detected if omitted)")
	flag.Parse()

	if *rulesDir == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "error: --rules and --output are required")
		os.Exit(1)
	}

	result, err := scaffold.Run(*rulesDir, *output, *language)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cli.WriteJSON(result)
}
