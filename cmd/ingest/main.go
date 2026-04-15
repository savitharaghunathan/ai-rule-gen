package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/ingestion"
)

func main() {
	input := flag.String("input", "", "URL, file path, or text content (required)")
	output := flag.String("output", "", "Output markdown file path (omit to print to stdout)")
	chunkSize := flag.Int("chunk-size", ingestion.DefaultMaxChunkSize, "Maximum characters per chunk")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		os.Exit(1)
	}

	result, err := ingestion.Ingest(*input, *chunkSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *output != "" {
		if err := ingestion.WriteMarkdown(*output, result.Content); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		cli.WriteJSON(map[string]interface{}{
			"output":     *output,
			"length":     len(result.Content),
			"chunks":     len(result.Chunks),
			"chunk_size": result.ChunkSize,
		})
		return
	}

	fmt.Print(result.Content)
}
