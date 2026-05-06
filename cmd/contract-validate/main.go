package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
	"github.com/konveyor/ai-rule-gen/internal/contract"
)

func main() {
	contractPath := flag.String("contract", "", "Path to skill contract.json (required)")
	mode := flag.String("mode", "returns", "Validation mode: inputs or returns")
	payload := flag.String("payload", "", "Raw JSON payload string (optional, use --payload-file for files)")
	payloadFile := flag.String("payload-file", "", "Path to JSON payload file (optional)")
	strict := flag.Bool("strict", true, "Fail on unknown fields")
	flag.Parse()

	if *contractPath == "" {
		cli.Fail("invalid_arguments", "--contract is required", "contract-validate", "set --contract to agents/<skill>/contract.json", nil)
	}
	if *payload == "" && *payloadFile == "" {
		cli.Fail("invalid_arguments", "one of --payload or --payload-file is required", "contract-validate", "provide JSON payload inline or via file", nil)
	}
	if *payload != "" && *payloadFile != "" {
		cli.Fail("invalid_arguments", "provide only one of --payload or --payload-file", "contract-validate", "use a single payload source", nil)
	}
	if *mode != "inputs" && *mode != "returns" {
		cli.Fail("invalid_arguments", "--mode must be one of: inputs, returns", "contract-validate", "set --mode to inputs or returns", nil)
	}

	c, err := contract.Load(*contractPath)
	if err != nil {
		cli.Fail("load_contract_failed", err.Error(), "contract-validate", "verify contract path and JSON schema", map[string]string{"contract": *contractPath})
	}

	raw, err := readPayload(*payload, *payloadFile)
	if err != nil {
		cli.Fail("read_payload_failed", err.Error(), "contract-validate", "verify payload source", map[string]string{"payload_file": *payloadFile})
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		cli.Fail("parse_payload_failed", err.Error(), "contract-validate", "ensure payload is valid JSON object", nil)
	}

	fields := c.Returns
	if *mode == "inputs" {
		fields = c.Inputs
	}
	errs := contract.ValidatePayload(fields, obj, *strict)
	if len(errs) > 0 {
		cli.Fail("contract_validation_failed", "payload does not satisfy contract", "contract-validate", "check required fields and value types", map[string]any{
			"mode":        *mode,
			"contract":    c.Name,
			"error_count": len(errs),
			"errors":      formatErrors(errs),
		})
	}

	cli.WriteJSON(map[string]any{
		"status":   "ok",
		"mode":     *mode,
		"contract": c.Name,
	})
}

func readPayload(inline, path string) ([]byte, error) {
	if inline != "" {
		return []byte(inline), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading payload file %s: %w", path, err)
	}
	return data, nil
}

func formatErrors(errs []contract.ValidationError) []map[string]string {
	out := make([]map[string]string, 0, len(errs))
	for _, e := range errs {
		out = append(out, map[string]string{
			"field":   e.Field,
			"message": e.Message,
		})
	}
	return out
}
