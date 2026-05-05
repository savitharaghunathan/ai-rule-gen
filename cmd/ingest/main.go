package main

import (
	"flag"
	"fmt"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/ingestion"
)

func main() {
	input := flag.String("input", "", "URL, file path, or text content (required)")
	output := flag.String("output", "", "Output markdown file path (omit to print to stdout)")
	chunkSize := flag.Int("chunk-size", ingestion.DefaultMaxChunkSize, "Maximum characters per chunk")
	flag.Parse()

	if *input == "" {
		cli.Fail("invalid_arguments", "--input is required", "ingest", "set --input to a URL, file path, or raw text", nil)
	}

	result, err := ingestion.Ingest(*input, *chunkSize)
	if err != nil {
		cli.Fail("ingest_failed", err.Error(), "ingest", "verify input source and network/file access", map[string]interface{}{"input": *input, "chunk_size": *chunkSize})
	}

	if *output != "" {
		if err := ingestion.WriteMarkdown(*output, result.Content); err != nil {
			cli.Fail("write_output_failed", err.Error(), "ingest", "check output path and write permissions", map[string]string{"output": *output})
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
