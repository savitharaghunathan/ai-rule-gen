package testgen

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/konveyor/ai-rule-gen/internal/llm"
	"github.com/konveyor/ai-rule-gen/internal/rules"
)

// FixLoopResult holds the result of generate + test + fix loop.
type FixLoopResult struct {
	GenerateOutput *GenerateOutput `json:"generate"`
	TestResult     *TestResult     `json:"test_result"`
	Iterations     int             `json:"iterations"`
}

// RunWithTests generates test data, fixes compilation errors, runs kantra, and iterates on failures.
func (g *Generator) RunWithTests(ctx context.Context, input GenerateInput, fixTmpl *template.Template, maxIterations int) (*FixLoopResult, error) {
	if maxIterations <= 0 {
		maxIterations = 3
	}

	// Step 1: Generate initial test data
	fmt.Println("  Generating test data...")
	genOutput, err := g.Generate(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("generating test data: %w", err)
	}

	testsDir := input.OutputDir + "/tests"

	// Step 2: Fix compilation errors + run kantra, iterate
	var testResult *TestResult
	for iteration := 1; iteration <= maxIterations; iteration++ {
		// Phase A: Fix compilation errors (up to 5 attempts)
		const maxCompileAttempts = 5
		for compileAttempt := 1; compileAttempt <= maxCompileAttempts; compileAttempt++ {
			compileErrors := g.checkCompilation(input, genOutput)
			if compileErrors == "" {
				break
			}

			if compileAttempt == 1 {
				fmt.Printf("  Compilation errors found (attempt %d/%d), asking LLM to fix...\n", compileAttempt, maxCompileAttempts)
			} else {
				fmt.Printf("  Still has compilation errors (attempt %d/%d), retrying...\n", compileAttempt, maxCompileAttempts)
			}

			err = g.fixCompilationErrors(ctx, input, compileErrors)
			if err != nil {
				fmt.Printf("  Warning: compilation fix failed: %v\n", err)
				break
			}
		}

		// Phase B: Run kantra tests
		fmt.Printf("  Running kantra tests (iteration %d/%d)...\n", iteration, maxIterations)
		testResult, err = RunKantraTests(ctx, testsDir, 900)
		if err != nil {
			return nil, fmt.Errorf("running kantra (iteration %d): %w", iteration, err)
		}

		fmt.Printf("  Result: %d/%d passed (%.0f%%)\n", testResult.Passed, testResult.Total, testResult.PassRate)

		// All passed — done
		if testResult.Passed == testResult.Total && testResult.Total > 0 {
			fmt.Println("  All tests passed!")
			break
		}

		// No failures parsed but tests didn't all pass
		if len(testResult.Failures) == 0 && testResult.Passed < testResult.Total {
			fmt.Println("  Warning: kantra reported failures but could not parse which rules failed")
			fmt.Println("  kantra output:")
			fmt.Println(testResult.RawOutput)
			break
		}

		// Last iteration — report and stop
		if iteration == maxIterations {
			fmt.Printf("  %d rule(s) still failing after %d iterations\n", len(testResult.Failures), maxIterations)
			break
		}

		// Phase C: Fix failing rules via code hints
		fmt.Printf("  Fixing %d failing rule(s)...\n", len(testResult.Failures))
		codeHints, err := g.generateFixHints(ctx, testResult.Failures, fixTmpl, input)
		if err != nil {
			fmt.Printf("  Warning: fix hints failed: %v\n", err)
			continue
		}

		err = g.regenerateWithHints(ctx, input, codeHints)
		if err != nil {
			fmt.Printf("  Warning: regeneration failed: %v\n", err)
			continue
		}
	}

	return &FixLoopResult{
		GenerateOutput: genOutput,
		TestResult:     testResult,
		Iterations:     maxIterations,
	}, nil
}

// checkCompilation runs the language-appropriate compile check and returns errors, or "" if clean.
func (g *Generator) checkCompilation(input GenerateInput, genOutput *GenerateOutput) string {
	language := input.Language
	if language == "" {
		language = "go"
	}

	for _, dataDir := range genOutput.DataDirs {
		fullPath := filepath.Join(input.OutputDir, "tests", dataDir)

		var cmd *exec.Cmd
		switch language {
		case "go":
			cmd = exec.Command("go", "build", "./...")
		case "java":
			cmd = exec.Command("mvn", "compile", "-q", "-B")
		case "nodejs":
			cmd = exec.Command("npx", "tsc", "--noEmit")
		case "csharp":
			cmd = exec.Command("dotnet", "build", "--no-restore")
		default:
			return ""
		}
		cmd.Dir = fullPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out)
		}
	}
	return ""
}

// fixCompilationErrors sends the source code + errors to LLM and regenerates.
func (g *Generator) fixCompilationErrors(ctx context.Context, input GenerateInput, compileErrors string) error {
	// Read current source files
	testsDir := filepath.Join(input.OutputDir, "tests")
	entries, err := os.ReadDir(filepath.Join(testsDir, "data"))
	if err != nil {
		return fmt.Errorf("reading data dir: %w", err)
	}

	language := input.Language
	if language == "" {
		language = "go"
	}
	langConfig, ok := languageConfigs[language]
	if !ok {
		return fmt.Errorf("unsupported language %q", language)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dataDir := filepath.Join(testsDir, "data", entry.Name())
		sourcePath := filepath.Join(dataDir, langConfig.SourceDir, langConfig.MainFile)
		buildPath := filepath.Join(dataDir, langConfig.BuildFile)

		sourceCode, err := os.ReadFile(sourcePath)
		if err != nil {
			continue
		}
		buildFile, err := os.ReadFile(buildPath)
		if err != nil {
			continue
		}

		// Gather API docs for packages mentioned in errors
		apiDocs := gatherAPIDocs(language, dataDir, compileErrors)

		prompt := fmt.Sprintf(`Fix the compilation errors in this %s code.

Current %s:
`+"```\n%s\n```"+`

Current %s:
`+"```\n%s\n```"+`

Compilation errors:
`+"```\n%s\n```"+`
%s
REQUIREMENTS:
1. Fix ONLY the lines mentioned in the compilation errors — do NOT change code that already compiles
2. Keep ALL the rule-triggering code — every import and usage that triggers a rule MUST remain
3. Use the API documentation above to match EXACT function signatures from the installed library version
4. If a function/type/constant doesn't exist in the installed version, find the correct alternative from the same package
5. Do NOT change the library version in %s — fix the code to match the installed version

Return EXACTLY TWO fenced code blocks:
FIRST: the fixed %s
SECOND: the fixed %s`,
			language,
			langConfig.BuildFile, string(buildFile),
			langConfig.MainFile, string(sourceCode),
			compileErrors,
			apiDocs,
			langConfig.BuildFile,
			langConfig.BuildFile, langConfig.MainFile,
		)

		response, err := g.completer.Complete(ctx, prompt)
		if err != nil {
			return fmt.Errorf("LLM fix: %w", err)
		}

		blocks := extractCodeBlocks(response)
		if len(blocks) < 2 {
			return fmt.Errorf("expected 2 code blocks in fix response, got %d", len(blocks))
		}

		// Write fixed files
		if err := os.WriteFile(buildPath, []byte(blocks[0].Content), 0o644); err != nil {
			return fmt.Errorf("writing fixed build file: %w", err)
		}
		if err := os.WriteFile(sourcePath, []byte(blocks[1].Content), 0o644); err != nil {
			return fmt.Errorf("writing fixed source file: %w", err)
		}

		// Re-run dependency resolution after fixing
		runDepResolve(language, dataDir)
	}

	return nil
}

// generateFixHints asks the LLM for code hints for each failing rule.
func (g *Generator) generateFixHints(ctx context.Context, failures []Failure, fixTmpl *template.Template, input GenerateInput) (map[string]string, error) {
	hints := make(map[string]string)

	ruleList, err := rules.ReadRulesDir(input.RulesDir)
	if err != nil {
		return nil, fmt.Errorf("reading rules: %w", err)
	}
	ruleMap := make(map[string]rules.Rule)
	for _, r := range ruleList {
		ruleMap[r.RuleID] = r
	}

	language := input.Language
	if language == "" {
		language = detectLanguage(ruleList)
		if language == "" {
			language = "go"
		}
	}

	for _, f := range failures {
		r, ok := ruleMap[f.RuleID]
		if !ok {
			continue
		}

		pattern, provider := extractRulePattern(r)
		if pattern == "" {
			continue
		}

		hint, err := g.getCodeHint(ctx, fixTmpl, f.RuleID, pattern, provider, language)
		if err != nil {
			fmt.Printf("    Warning: could not get hint for %s: %v\n", f.RuleID, err)
			continue
		}
		hints[f.RuleID] = hint
		fmt.Printf("    Fix hint for %s: %s\n", f.RuleID, hint)
	}

	return hints, nil
}

// getCodeHint calls the LLM with the fix_hint template.
func (g *Generator) getCodeHint(ctx context.Context, fixTmpl *template.Template, ruleID, pattern, provider, language string) (string, error) {
	var buf bytes.Buffer
	data := map[string]string{
		"RuleID":   ruleID,
		"Pattern":  pattern,
		"Provider": provider,
		"Language": language,
	}
	if err := fixTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering fix template: %w", err)
	}

	response, err := g.completer.Complete(ctx, buf.String())
	if err != nil {
		return "", fmt.Errorf("LLM fix hint: %w", err)
	}

	// Clean up: remove markdown fencing, trim
	hint := strings.TrimSpace(response)
	hint = strings.TrimPrefix(hint, "```")
	hint = strings.TrimSuffix(hint, "```")
	hint = strings.TrimSpace(hint)
	if idx := strings.Index(hint, "\n"); idx >= 0 && idx < 10 {
		first := hint[:idx]
		if !strings.Contains(first, " ") && !strings.Contains(first, "(") {
			hint = strings.TrimSpace(hint[idx+1:])
		}
	}

	return hint, nil
}

// regenerateWithHints regenerates test data, injecting code hints for failing rules.
func (g *Generator) regenerateWithHints(ctx context.Context, input GenerateInput, hints map[string]string) error {
	if len(hints) == 0 {
		return nil
	}

	var hintBlock strings.Builder
	hintBlock.WriteString("\n\nIMPORTANT: The following rules FAILED in kantra tests. For each one, use the EXACT code snippet provided:\n")
	for ruleID, hint := range hints {
		fmt.Fprintf(&hintBlock, "- %s: %s\n", ruleID, hint)
	}

	wrapped := &hintCompleter{
		inner: g.completer,
		extra: hintBlock.String(),
	}

	orig := g.completer
	g.completer = wrapped
	defer func() { g.completer = orig }()

	_, err := g.Generate(ctx, input)
	return err
}

// hintCompleter wraps a Completer and appends extra context to prompts.
type hintCompleter struct {
	inner llm.Completer
	extra string
}

func (h *hintCompleter) Complete(ctx context.Context, prompt string) (string, error) {
	return h.inner.Complete(ctx, prompt+h.extra)
}

// extractRulePattern extracts the primary pattern and provider from a rule's condition.
func extractRulePattern(r rules.Rule) (pattern, provider string) {
	c := r.When
	if c.GoReferenced != nil {
		return c.GoReferenced.Pattern, "go.referenced"
	}
	if c.JavaReferenced != nil {
		return c.JavaReferenced.Pattern, "java.referenced"
	}
	if c.NodejsReferenced != nil {
		return c.NodejsReferenced.Pattern, "nodejs.referenced"
	}
	if c.CSharpReferenced != nil {
		return c.CSharpReferenced.Pattern, "csharp.referenced"
	}
	if c.BuiltinFilecontent != nil {
		return c.BuiltinFilecontent.Pattern, "builtin.filecontent"
	}
	for _, entry := range c.Or {
		if p, prov := extractRulePattern(rules.Rule{When: entry.Condition}); p != "" {
			return p, prov
		}
	}
	for _, entry := range c.And {
		if p, prov := extractRulePattern(rules.Rule{When: entry.Condition}); p != "" {
			return p, prov
		}
	}
	return "", ""
}

// gatherAPIDocs extracts package names from compilation errors and looks up API docs
// to give the LLM actual function signatures instead of guessing.
// - Go: runs `go doc <package>` — full API signatures
// - Java: extracts failing symbols from javac errors (no runtime doc lookup yet)
// - Node.js: parses TypeScript error messages for property/member info
// - C#: parses dotnet build error messages for type/member info
//
// TODO: Java — integrate javap or Maven dependency:tree for richer API docs.
// TODO: Node.js — read .d.ts files from node_modules for actual type signatures.
// TODO: C# — use dotnet metadata inspection for member lookup.
func gatherAPIDocs(language, dataDir, compileErrors string) string {
	switch language {
	case "go":
		return gatherGoAPIDocs(dataDir, compileErrors)
	case "java":
		return gatherJavaAPIDocs(dataDir, compileErrors)
	case "nodejs":
		return gatherNodejsAPIDocs(dataDir, compileErrors)
	case "csharp":
		return gatherCSharpAPIDocs(dataDir, compileErrors)
	default:
		return ""
	}
}

// gatherGoAPIDocs runs `go doc` on packages referenced in compilation errors.
func gatherGoAPIDocs(dataDir, compileErrors string) string {
	// Extract package references from errors like "undefined: chacha20.New"
	// or "cannot use ... in argument to salsa20.XORKeyStream"
	re := regexp.MustCompile(`(?:undefined:\s+|argument to\s+)(\w+)\.(\w+)`)
	matches := re.FindAllStringSubmatch(compileErrors, -1)

	seen := make(map[string]bool)
	var docs strings.Builder

	for _, m := range matches {
		pkgShort := m[1]
		if seen[pkgShort] {
			continue
		}
		seen[pkgShort] = true

		fullPkg := findGoImportPath(dataDir, pkgShort)
		if fullPkg == "" {
			continue
		}

		cmd := exec.Command("go", "doc", fullPkg)
		cmd.Dir = dataDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		fmt.Fprintf(&docs, "\nAPI for %s (installed version):\n```\n%s```\n", fullPkg, string(out))
	}

	return docs.String()
}

// gatherJavaAPIDocs extracts class names from javac errors and looks up their API via javap.
func gatherJavaAPIDocs(dataDir, compileErrors string) string {
	// javac errors: "cannot find symbol ... symbol: method foo()" or "package com.example does not exist"
	re := regexp.MustCompile(`cannot find symbol.*?symbol:\s+(?:method|variable|class)\s+(\w+)`)
	matches := re.FindAllStringSubmatch(compileErrors, -1)
	if len(matches) == 0 {
		return ""
	}

	var docs strings.Builder
	seen := make(map[string]bool)
	for _, m := range matches {
		symbol := m[1]
		if seen[symbol] {
			continue
		}
		seen[symbol] = true
		fmt.Fprintf(&docs, "\nFailing symbol: %s — check the correct method name and signature in the dependency's API.\n", symbol)
	}
	return docs.String()
}

// gatherNodejsAPIDocs extracts type info from TypeScript errors and looks up .d.ts definitions.
func gatherNodejsAPIDocs(dataDir, compileErrors string) string {
	// TS errors: "Property 'foo' does not exist on type 'Bar'"
	// or "Module '"xyz"' has no exported member 'Foo'"
	reProperty := regexp.MustCompile(`Property '(\w+)' does not exist on type '(\w+)'`)
	reMember := regexp.MustCompile(`has no exported member '(\w+)'`)

	var docs strings.Builder
	seen := make(map[string]bool)

	for _, m := range reProperty.FindAllStringSubmatch(compileErrors, -1) {
		key := m[2] + "." + m[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		fmt.Fprintf(&docs, "\nFailing: %s has no property '%s' — check the correct property/method name in the package's type definitions.\n", m[2], m[1])
	}
	for _, m := range reMember.FindAllStringSubmatch(compileErrors, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		fmt.Fprintf(&docs, "\nFailing export: '%s' — check the correct exported name in the package.\n", m[1])
	}

	return docs.String()
}

// gatherCSharpAPIDocs extracts type/member info from dotnet build errors.
func gatherCSharpAPIDocs(dataDir, compileErrors string) string {
	// CS errors: "'Type' does not contain a definition for 'Member'"
	// or "The type or namespace name 'Foo' could not be found"
	re := regexp.MustCompile(`'(\w+)' does not contain a definition for '(\w+)'`)
	reNs := regexp.MustCompile(`type or namespace name '(\w+)' could not be found`)

	var docs strings.Builder
	seen := make(map[string]bool)

	for _, m := range re.FindAllStringSubmatch(compileErrors, -1) {
		key := m[1] + "." + m[2]
		if seen[key] {
			continue
		}
		seen[key] = true
		fmt.Fprintf(&docs, "\nFailing: %s has no member '%s' — check the correct member name in the installed package version.\n", m[1], m[2])
	}
	for _, m := range reNs.FindAllStringSubmatch(compileErrors, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		fmt.Fprintf(&docs, "\nFailing type/namespace: '%s' — check the correct name and using directives.\n", m[1])
	}

	return docs.String()
}

// findGoImportPath searches Go source files in dataDir for an import matching the short package name.
func findGoImportPath(dataDir, pkgShort string) string {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dataDir, e.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "\"") && strings.HasSuffix(strings.Trim(line, "\""), "/"+pkgShort) {
				start := strings.Index(line, "\"")
				end := strings.LastIndex(line, "\"")
				if start >= 0 && end > start {
					return line[start+1 : end]
				}
			}
		}
	}
	return ""
}
