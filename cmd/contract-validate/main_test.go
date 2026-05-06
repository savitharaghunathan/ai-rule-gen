package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "contract-validate")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = filepath.Join(".") // current package
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

const testContract = `{
  "name": "test-skill",
  "version": "1.0.0",
  "inputs": [
    {"name": "guide", "type": "string", "required": true},
    {"name": "count", "type": "number", "required": false}
  ],
  "returns": [
    {"name": "result", "type": "string", "required": true},
    {"name": "lang", "type": "string", "required": false, "enum": ["java", "go"]}
  ]
}`

func TestValidPayload(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	contract := writeFile(t, dir, "contract.json", testContract)
	payload := writeFile(t, dir, "payload.json", `{"guide": "path/to/guide.md"}`)

	cmd := exec.Command(bin, "--contract", contract, "--mode", "inputs", "--payload-file", payload)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success, got error: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"status": "ok"`) {
		t.Errorf("expected ok status in output: %s", out)
	}
}

func TestMissingRequiredField(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	contract := writeFile(t, dir, "contract.json", testContract)
	payload := writeFile(t, dir, "payload.json", `{"count": 5}`)

	cmd := exec.Command(bin, "--contract", contract, "--mode", "inputs", "--payload-file", payload)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected failure for missing required field")
	}
	if !strings.Contains(string(out), "required field is missing") {
		t.Errorf("expected 'required field is missing' in output: %s", out)
	}
}

func TestEnumValidation(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	contract := writeFile(t, dir, "contract.json", testContract)

	t.Run("valid enum", func(t *testing.T) {
		payload := writeFile(t, dir, "valid.json", `{"result": "ok", "lang": "java"}`)
		cmd := exec.Command(bin, "--contract", contract, "--mode", "returns", "--payload-file", payload)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("expected success: %v\n%s", err, out)
		}
	})

	t.Run("invalid enum", func(t *testing.T) {
		payload := writeFile(t, dir, "invalid.json", `{"result": "ok", "lang": "ruby"}`)
		cmd := exec.Command(bin, "--contract", contract, "--mode", "returns", "--payload-file", payload)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("expected failure for invalid enum value")
		}
		if !strings.Contains(string(out), "not in enum") {
			t.Errorf("expected 'not in enum' in output: %s", out)
		}
	})
}

func TestInlinePayload(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	contract := writeFile(t, dir, "contract.json", testContract)

	cmd := exec.Command(bin, "--contract", contract, "--mode", "inputs", "--payload", `{"guide": "test.md"}`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), `"status": "ok"`) {
		t.Errorf("expected ok status: %s", out)
	}
}

func TestMissingContract(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--payload", `{}`)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected failure for missing --contract")
	}
	if !strings.Contains(string(out), "invalid_arguments") {
		t.Errorf("expected invalid_arguments error: %s", out)
	}
}

func TestStrictUnknownFields(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	contract := writeFile(t, dir, "contract.json", testContract)
	payload := writeFile(t, dir, "payload.json", `{"guide": "x", "unknown_field": true}`)

	cmd := exec.Command(bin, "--contract", contract, "--mode", "inputs", "--payload-file", payload, "--strict=true")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected failure for unknown field in strict mode")
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Errorf("expected 'unknown field' in output: %s", out)
	}
}
