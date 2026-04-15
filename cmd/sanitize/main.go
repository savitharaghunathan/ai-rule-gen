package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/sanitize"
)

func main() {
	dir := flag.String("dir", "", "Directory containing XML files to sanitize (required)")
	flag.Parse()

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "error: --dir is required")
		os.Exit(1)
	}

	if err := sanitize.Dir(*dir); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cli.WriteJSON(map[string]interface{}{
		"status":    "ok",
		"directory": *dir,
	})
}
