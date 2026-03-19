package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/konveyor/ai-rule-gen/internal/extraction"
	"github.com/konveyor/ai-rule-gen/internal/generation"
	"github.com/konveyor/ai-rule-gen/internal/ingestion"
	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
	"github.com/konveyor/ai-rule-gen/internal/workspace"
	"github.com/konveyor/ai-rule-gen/templates"
	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	Concerns          []string `json:"concerns"`
	PatternsExtracted int      `json:"patterns_extracted"`
}

// GenerateRulesHandler returns an MCP tool handler for generate_rules.
// If completer is nil, it creates one from RULEGEN_LLM_PROVIDER env var.
func GenerateRulesHandler(completer llm.Completer) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input GenerateInput
		if err := json.Unmarshal(req.Params.Arguments, &input); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		if input.Input == "" || input.Source == "" || input.Target == "" {
			return errorResult("input, source, and target are required"), nil
		}
		if input.OutputPath == "" {
			input.OutputPath = "output"
		}

		c := completer
		if c == nil {
			var err error
			c, err = llm.NewCompleterFromEnv()
			if err != nil {
				return errorResult(fmt.Sprintf("LLM configuration error: %v", err)), nil
			}
			if c == nil {
				return errorResult("LLM provider required: set RULEGEN_LLM_PROVIDER (anthropic, openai, gemini, ollama) and the corresponding API key env var"), nil
			}
		}

		result, err := RunGeneratePipeline(ctx, c, input)
		if err != nil {
			return errorResult(fmt.Sprintf("generate_rules failed: %v", err)), nil
		}

		data, err := json.Marshal(result)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to marshal result: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(data)},
			},
		}, nil
	}
}

// RunGeneratePipeline executes the full rule generation pipeline.
func RunGeneratePipeline(ctx context.Context, completer llm.Completer, input GenerateInput) (*GenerateOutput, error) {
	// 1. Ingest
	ingested, err := ingestion.Ingest(input.Input, ingestion.DefaultMaxChunkSize)
	if err != nil {
		return nil, fmt.Errorf("ingestion: %w", err)
	}

	// 1b. Auto-detect source/target/language if not provided
	if input.Source == "" || input.Target == "" {
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
		fmt.Printf("Auto-detected: source=%s, target=%s, language=%s\n", input.Source, input.Target, input.Language)
	}

	// 2. Extract patterns (LLM)
	fmt.Printf("Extracting patterns from %d chunk(s)...\n", len(ingested.Chunks))
	extractTmpl, err := templates.Load("extraction/extract_patterns.tmpl")
	if err != nil {
		return nil, fmt.Errorf("loading extraction template: %w", err)
	}
	extractor := extraction.New(completer, extractTmpl)
	patterns, err := extractor.Extract(ctx, ingested.Chunks, input.Source, input.Target, input.Language)
	if err != nil {
		return nil, fmt.Errorf("extraction: %w", err)
	}
	fmt.Printf("Extracted %d patterns\n", len(patterns))

	// 3. Generate rules (deterministic + LLM for messages)
	fmt.Println("Generating rules...")
	messageTmpl, err := templates.Load("generation/generate_message.tmpl")
	if err != nil {
		return nil, fmt.Errorf("loading message template: %w", err)
	}
	generator := generation.New(completer, messageTmpl)
	grouped, ruleset, err := generator.Generate(ctx, patterns, generation.GenerateInput{
		Source:   input.Source,
		Target:   input.Target,
		Language: input.Language,
	})
	if err != nil {
		return nil, fmt.Errorf("generation: %w", err)
	}

	// 4. Validate
	var allRules []rules.Rule
	for _, rr := range grouped {
		allRules = append(allRules, rr...)
	}
	fmt.Printf("Generated %d rules, validating...\n", len(allRules))
	result := rules.Validate(allRules)
	if !result.Valid {
		return nil, fmt.Errorf("generated rules failed validation: %v", result.Errors)
	}

	// 5. Save to workspace
	ws, err := workspace.New(input.OutputPath, input.Source, input.Target)
	if err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	if err := rules.WriteRuleset(ws.RulesetPath(), ruleset); err != nil {
		return nil, fmt.Errorf("writing ruleset: %w", err)
	}
	if err := rules.WriteRulesGrouped(ws.RulesDir(), grouped); err != nil {
		return nil, fmt.Errorf("writing rules: %w", err)
	}

	// Build output
	var filesWritten []string
	var concerns []string
	filesWritten = append(filesWritten, "ruleset.yaml")
	for concern := range grouped {
		name := concern
		if name == "" {
			name = "general"
		}
		filesWritten = append(filesWritten, name+".yaml")
		concerns = append(concerns, name)
	}

	return &GenerateOutput{
		OutputPath:        ws.Root,
		FilesWritten:      filesWritten,
		RuleCount:         len(allRules),
		Concerns:          concerns,
		PatternsExtracted: len(patterns),
	}, nil
}
