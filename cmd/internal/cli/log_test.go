package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestInitLog_LogPathOverridesEnvVar(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "env.log")
	flagPath := filepath.Join(t.TempDir(), "flag.log")
	t.Setenv("RULE_GEN_LOG", envPath)

	InitLog(flagPath, "test-agent", "test-model")
	Log("hello")
	CloseLog()

	// flagPath should have the entry, envPath should not exist or be empty
	data, err := os.ReadFile(flagPath)
	if err != nil {
		t.Fatalf("read flag log: %v", err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Error("expected log entry in flag log file")
	}

	if _, err := os.Stat(envPath); err == nil {
		envData, _ := os.ReadFile(envPath)
		if len(envData) > 0 {
			t.Error("expected env log file to be empty or not exist")
		}
	}
}

func TestLog_WritesTimestampedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	t.Setenv("RULE_GEN_LOG", path)

	InitLog("", "", "")
	Log("hello %s", "world")
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, "] hello world") {
		t.Errorf("expected timestamped message, got %q", line)
	}
	if !strings.HasPrefix(line, "[") {
		t.Errorf("expected line to start with timestamp bracket, got %q", line)
	}
}

func TestLog_IncludesAgentAndModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	InitLog(path, "rule-writer", "claude-sonnet-4-20250514")
	Log("extracting patterns")
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, "[agent=rule-writer]") {
		t.Errorf("expected [agent=rule-writer], got %q", line)
	}
	if !strings.Contains(line, "[model=claude-sonnet-4-20250514]") {
		t.Errorf("expected [model=claude-sonnet-4-20250514], got %q", line)
	}
}

func TestLog_DefaultsAgentAndCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	InitLog(path, "", "")
	Log("test message")
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	line := strings.TrimSpace(string(data))
	if !strings.Contains(line, "[agent=unknown]") {
		t.Errorf("expected [agent=unknown] as default, got %q", line)
	}
	if !strings.Contains(line, "[cli]") {
		t.Errorf("expected [cli] tag for no-model call, got %q", line)
	}
	if strings.Contains(line, "[model=") {
		t.Errorf("expected no [model=] tag for CLI call, got %q", line)
	}
}

func TestLog_FormatCLI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	InitLog(path, "orchestrator", "")
	Log("hello world")
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	line := strings.TrimSpace(string(data))
	// Format: [HH:MM:SS] [cmd] [agent=X] [cli] message
	re := regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\] \[\S+\] \[agent=orchestrator\] \[cli\] hello world$`)
	if !re.MatchString(line) {
		t.Errorf("log line does not match expected format, got %q", line)
	}
}

func TestLog_FormatWithModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	InitLog(path, "rule-writer", "claude-sonnet-4-20250514")
	Log("extracting")
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	line := strings.TrimSpace(string(data))
	// Format: [HH:MM:SS] [cmd] [agent=X] [model=Y] message
	re := regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\] \[\S+\] \[agent=rule-writer\] \[model=claude-sonnet-4-20250514\] extracting$`)
	if !re.MatchString(line) {
		t.Errorf("log line does not match expected format, got %q", line)
	}
}

func TestLog_NoopWhenInactive(t *testing.T) {
	t.Setenv("RULE_GEN_LOG", "")
	InitLog("", "", "")
	defer CloseLog()

	Log("this should not panic")
}

func TestLogJSON_WritesLabeledJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	InitLog(path, "orchestrator", "none")
	LogJSON("output", map[string]int{"count": 42})
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	line := string(data)
	if !strings.Contains(line, "output:") {
		t.Errorf("expected label 'output:', got %q", line)
	}
	if !strings.Contains(line, `"count"`) {
		t.Errorf("expected JSON content, got %q", line)
	}
	if !strings.Contains(line, "[agent=orchestrator]") {
		t.Errorf("expected agent attribution, got %q", line)
	}
}

func TestCloseLog_DoubleCloseIsSafe(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	InitLog(path, "", "")
	CloseLog()
	CloseLog()
}

func TestInitLog_AppendsToExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	InitLog(path, "agent-2", "model-2")
	Log("appended")
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "existing\n") {
		t.Error("existing content was overwritten")
	}
	if !strings.Contains(content, "appended") {
		t.Error("new content was not appended")
	}
	if !strings.Contains(content, "[agent=agent-2]") {
		t.Error("agent attribution missing in appended content")
	}
}

func TestMultipleAgents_SameLogFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")

	// First agent writes
	InitLog(path, "orchestrator", "claude-opus-4-6")
	Log("dispatching rule-writer")
	CloseLog()

	// Second agent writes to same file
	InitLog(path, "rule-writer", "claude-sonnet-4-20250514")
	Log("extracting patterns")
	CloseLog()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "[agent=orchestrator]") {
		t.Error("missing orchestrator entry")
	}
	if !strings.Contains(content, "[agent=rule-writer]") {
		t.Error("missing rule-writer entry")
	}
	if !strings.Contains(content, "[model=claude-opus-4-6]") {
		t.Error("missing orchestrator model")
	}
	if !strings.Contains(content, "[model=claude-sonnet-4-20250514]") {
		t.Error("missing rule-writer model")
	}

	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 log lines, got %d", len(lines))
	}
}
