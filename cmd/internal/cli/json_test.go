package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	WriteJSON(map[string]string{"key": "value"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	got := buf.String()

	var parsed map[string]string
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["key"] != "value" {
		t.Errorf("got key=%q, want %q", parsed["key"], "value")
	}
}

func TestWriteError(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	WriteError("test_code", "something broke", "build", "try again", map[string]string{"file": "main.go"})

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	got := buf.String()

	var parsed ErrorResponse
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed.Status != "error" {
		t.Errorf("status=%q, want %q", parsed.Status, "error")
	}
	if parsed.Code != "test_code" {
		t.Errorf("code=%q, want %q", parsed.Code, "test_code")
	}
	if parsed.Message != "something broke" {
		t.Errorf("message=%q, want %q", parsed.Message, "something broke")
	}
	if parsed.Step != "build" {
		t.Errorf("step=%q, want %q", parsed.Step, "build")
	}
	if parsed.Hint != "try again" {
		t.Errorf("hint=%q, want %q", parsed.Hint, "try again")
	}
}

func TestWriteErrorOmitsEmpty(t *testing.T) {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	WriteError("err", "msg", "", "", nil)

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	got := buf.String()

	var raw map[string]any
	if err := json.Unmarshal([]byte(got), &raw); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if _, ok := raw["step"]; ok {
		t.Error("empty step should be omitted")
	}
	if _, ok := raw["hint"]; ok {
		t.Error("empty hint should be omitted")
	}
	if _, ok := raw["details"]; ok {
		t.Error("nil details should be omitted")
	}
}
