package tools

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/konveyor/ai-rule-gen/internal/extraction"
	"github.com/konveyor/ai-rule-gen/internal/generation"
	"github.com/konveyor/ai-rule-gen/internal/ingestion"
	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
	"github.com/konveyor/ai-rule-gen/templates"
)

// GenerateInput holds the parameters for the generate pipeline.
type GenerateInput struct {
	Input      string `json:"input"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Language   string `json:"language"`
	OutputPath string `json:"output_path"`
}

// GenerateOutput holds the results of the generate pipeline.
type GenerateOutput struct {
	OutputPath        string   `json:"output_path"`
	FilesWritten      []string `json:"files_written"`
	RuleCount         int      `json:"rule_count"`
	PatternsExtracted int      `json:"patterns_extracted"`
}

// RunGeneratePipeline executes the full rule generation pipeline.
func RunGeneratePipeline(ctx context.Context, completer llm.Completer, input GenerateInput) (*GenerateOutput, error) {
	pipelineStart := time.Now()

	// 1. Ingest
	stepStart := time.Now()
	ingested, err := ingestion.Ingest(ctx, input.Input, ingestion.DefaultMaxChunkSize)
	if err != nil {
		return nil, fmt.Errorf("ingestion: %w", err)
	}
	slog.Info("ingestion complete", "chunks", len(ingested.Chunks), "duration", time.Since(stepStart).Round(time.Millisecond))

	// 1b. Auto-detect source/target/language if not provided
	if input.Source == "" || input.Target == "" {
		stepStart = time.Now()
		detectTmpl, err := templates.Load("extraction/detect_metadata.tmpl")
		if err != nil {
			return nil, fmt.Errorf("loading detect template: %w", err)
		}
		content := ingested.Chunks[0]
		meta, err := extraction.DetectMetadata(ctx, completer, detectTmpl, content)
		if err != nil {
			return nil, fmt.Errorf("auto-detection: %w", err)
		}
		if input.Source == "" {
			input.Source = meta.Source
		}
		if input.Target == "" {
			input.Target = meta.Target
		}
		if input.Language == "" {
			input.Language = meta.Language
		}
		slog.Info("auto-detection complete", "source", input.Source, "target", input.Target, "language", input.Language, "duration", time.Since(stepStart).Round(time.Millisecond))
	}

	// 2. Extract patterns (LLM)
	stepStart = time.Now()
	slog.Info("extracting patterns", "chunks", len(ingested.Chunks))
	extractTmpl, err := templates.Load("extraction/extract_patterns.tmpl")
	if err != nil {
		return nil, fmt.Errorf("loading extraction template: %w", err)
	}
	extractor := extraction.New(completer, extractTmpl)
	patterns, err := extractor.Extract(ctx, ingested.Chunks, input.Source, input.Target, input.Language)
	if err != nil {
		return nil, fmt.Errorf("extraction: %w", err)
	}
	slog.Info("extraction complete", "patterns", len(patterns), "duration", time.Since(stepStart).Round(time.Millisecond))

	// 3. Generate rules (deterministic + LLM for messages)
	stepStart = time.Now()
	slog.Info("generating rules")
	messageTmpl, err := templates.Load("generation/generate_message.tmpl")
	if err != nil {
		return nil, fmt.Errorf("loading message template: %w", err)
	}
	generator := generation.New(completer, messageTmpl)
	ruleList, ruleset, err := generator.Generate(ctx, patterns, generation.GenerateInput{
		Source:   input.Source,
		Target:   input.Target,
		Language: input.Language,
	})
	if err != nil {
		return nil, fmt.Errorf("generation: %w", err)
	}
	slog.Info("generation complete", "rules", len(ruleList), "duration", time.Since(stepStart).Round(time.Millisecond))

	// 4. Validate
	stepStart = time.Now()
	slog.Info("validating rules", "count", len(ruleList))
	result := rules.Validate(ruleList)
	if !result.Valid {
		return nil, fmt.Errorf("generated rules failed validation: %v", result.Errors)
	}
	slog.Info("validation complete", "warnings", len(result.Warnings), "duration", time.Since(stepStart).Round(time.Millisecond))

	// 5. Save to workspace
	ws, err := workspace.New(input.OutputPath, input.Source, input.Target)
	if err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	if err := rules.WriteRuleset(ws.RulesetPath(), ruleset); err != nil {
		return nil, fmt.Errorf("writing ruleset: %w", err)
	}
	rulesFilePath := fmt.Sprintf("%s/rules.yaml", ws.RulesDir())
	if err := rules.WriteRulesFile(rulesFilePath, ruleList); err != nil {
		return nil, fmt.Errorf("writing rules: %w", err)
	}

	slog.Info("pipeline complete", "total_duration", time.Since(pipelineStart).Round(time.Millisecond))

	return &GenerateOutput{
		OutputPath:        ws.Root,
		FilesWritten:      []string{"ruleset.yaml", "rules.yaml"},
		RuleCount:         len(ruleList),
		PatternsExtracted: len(patterns),
	}, nil
}

// RunExtractPipeline extracts migration patterns from input and returns them
// as ExtractOutput JSON (no rule generation, no file writing).
func RunExtractPipeline(ctx context.Context, completer llm.Completer, input GenerateInput) (*ExtractOutput, error) {
	pipelineStart := time.Now()

	// 1. Ingest
	stepStart := time.Now()
	ingested, err := ingestion.Ingest(ctx, input.Input, ingestion.DefaultMaxChunkSize)
	if err != nil {
		return nil, fmt.Errorf("ingestion: %w", err)
	}
	slog.Info("ingestion complete", "chunks", len(ingested.Chunks), "duration", time.Since(stepStart).Round(time.Millisecond))

	// 1b. Auto-detect source/target/language if not provided
	if input.Source == "" || input.Target == "" {
		stepStart = time.Now()
		detectTmpl, err := templates.Load("extraction/detect_metadata.tmpl")
		if err != nil {
			return nil, fmt.Errorf("loading detect template: %w", err)
		}
		content := ingested.Chunks[0]
		meta, err := extraction.DetectMetadata(ctx, completer, detectTmpl, content)
		if err != nil {
			return nil, fmt.Errorf("auto-detection: %w", err)
		}
		if input.Source == "" {
			input.Source = meta.Source
		}
		if input.Target == "" {
			input.Target = meta.Target
		}
		if input.Language == "" {
			input.Language = meta.Language
		}
		slog.Info("auto-detection complete", "source", input.Source, "target", input.Target, "language", input.Language, "duration", time.Since(stepStart).Round(time.Millisecond))
	}

	// 2. Extract patterns (LLM)
	stepStart = time.Now()
	slog.Info("extracting patterns", "chunks", len(ingested.Chunks))
	extractTmpl, err := templates.Load("extraction/extract_patterns.tmpl")
	if err != nil {
		return nil, fmt.Errorf("loading extraction template: %w", err)
	}
	extractor := extraction.New(completer, extractTmpl)
	patterns, err := extractor.Extract(ctx, ingested.Chunks, input.Source, input.Target, input.Language)
	if err != nil {
		return nil, fmt.Errorf("extraction: %w", err)
	}
	slog.Info("extraction complete", "patterns", len(patterns), "duration", time.Since(stepStart).Round(time.Millisecond))

	slog.Info("extract pipeline complete", "total_duration", time.Since(pipelineStart).Round(time.Millisecond))

	return &ExtractOutput{
		Source:   input.Source,
		Target:   input.Target,
		Language: input.Language,
		Patterns: patterns,
	}, nil
}

