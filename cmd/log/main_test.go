package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "log-cmd")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func TestLog_WritesMessageToFile(t *testing.T) {
	bin := buildBinary(t)
	logFile := filepath.Join(t.TempDir(), "test.log")

	cmd := exec.Command(bin, "--log", logFile, "--agent", "orchestrator", "--model", "test-model", "--message", "Pipeline start")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	line := string(data)
	if !strings.Contains(line, "Pipeline start") {
		t.Errorf("expected message in log, got %q", line)
	}
	if !strings.Contains(line, "[agent=orchestrator]") {
		t.Errorf("expected agent attribution, got %q", line)
	}
	if !strings.Contains(line, "[model=test-model]") {
		t.Errorf("expected model attribution, got %q", line)
	}
}

func TestLog_MissingMessageFails(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "--log", filepath.Join(t.TempDir(), "test.log"), "--agent", "orchestrator")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit when --message is missing")
	}

	combined := string(out)
	if !strings.Contains(combined, "invalid_arguments") {
		t.Errorf("expected invalid_arguments error code, got %q", combined)
	}
}

func TestLog_JSONOutputOnStdout(t *testing.T) {
	bin := buildBinary(t)
	logFile := filepath.Join(t.TempDir(), "test.log")

	cmd := exec.Command(bin, "--log", logFile, "--agent", "test", "--message", "hello")
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(stdout, &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, stdout)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", result["status"])
	}
	if result["message"] != "hello" {
		t.Errorf("expected message=hello, got %q", result["message"])
	}
}
