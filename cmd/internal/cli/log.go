package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	logFile   *os.File
	logMu     sync.Mutex
	cmdName   string
	agentName string
	modelName string
)

// InitLog opens the session log file in append mode.
// logPath overrides the RULE_GEN_LOG environment variable if non-empty.
// agent and model identify the invoking agent and LLM model in every log line.
// Call defer CloseLog() immediately after.
func InitLog(logPath, agent, model string) {
	path := logPath
	if path == "" {
		path = os.Getenv("RULE_GEN_LOG")
	}
	if path == "" {
		return
	}

	cmdName = inferCmdName()
	agentName = agent
	if agentName == "" {
		agentName = "unknown"
	}
	modelName = model

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot create log directory for %s: %v\n", path, err)
		return
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot open log file %s: %v\n", path, err)
		return
	}
	logFile = f
}

// CloseLog flushes and closes the log file.
func CloseLog() {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// Log appends a timestamped line with agent/model attribution to the session log.
// No-op if logging is not active.
func Log(format string, args ...any) {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	ts := time.Now().Format("15:04:05")
	fmt.Fprintf(logFile, "[%s] [%s] [agent=%s] %s %s\n", ts, cmdName, agentName, modelTag(), msg)
}

// LogJSON appends a timestamped JSON block with agent/model attribution to the session log.
// No-op if logging is not active.
func LogJSON(label string, v any) {
	logMu.Lock()
	defer logMu.Unlock()
	if logFile == nil {
		return
	}
	data, err := json.MarshalIndent(v, "  ", "  ")
	if err != nil {
		return
	}
	ts := time.Now().Format("15:04:05")
	fmt.Fprintf(logFile, "[%s] [%s] [agent=%s] %s %s: %s\n", ts, cmdName, agentName, modelTag(), label, string(data))
}

func modelTag() string {
	if modelName == "" {
		return "[cli]"
	}
	return "[model=" + modelName + "]"
}

func inferCmdName() string {
	if len(os.Args) == 0 {
		return "unknown"
	}
	bin := filepath.Base(os.Args[0])
	// go run ./cmd/test produces a binary like "test" or a temp path ending in the cmd name
	if strings.Contains(bin, "___") || strings.HasPrefix(bin, "exe") {
		parts := strings.Split(os.Args[0], string(filepath.Separator))
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] == "cmd" && i+1 < len(parts) {
				return parts[i+1]
			}
		}
	}
	return bin
}
