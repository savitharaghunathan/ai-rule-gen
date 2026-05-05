package main

import (
	"flag"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/sanitize"
)

func main() {
	dir := flag.String("dir", "", "Directory containing XML files to sanitize (required)")
	flag.Parse()

	if *dir == "" {
		cli.Fail("invalid_arguments", "--dir is required", "sanitize", "set --dir to the test data directory containing XML files", nil)
	}

	if err := sanitize.Dir(*dir); err != nil {
		cli.Fail("sanitize_failed", err.Error(), "sanitize", "check XML content and directory permissions", map[string]string{"dir": *dir})
	}

	cli.WriteJSON(map[string]interface{}{
		"status":    "ok",
		"directory": *dir,
	})
}
