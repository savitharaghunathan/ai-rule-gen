package main

import (
	"flag"

	"github.com/konveyor/ai-rule-gen/cmd/internal/cli"
)

func main() {
	logPath := flag.String("log", "", "Append structured output to this log file (overrides RULE_GEN_LOG)")
	agentFlag := flag.String("agent", "", "Name of the invoking agent (for log attribution)")
	modelFlag := flag.String("model", "", "LLM model powering the invoking agent (for log attribution)")
	message := flag.String("message", "", "Message to log (required)")
	flag.Parse()

	if *message == "" {
		cli.Fail("invalid_arguments", "--message is required", "log", "pass --message with the text to log", nil)
	}

	cli.InitLog(*logPath, *agentFlag, *modelFlag)
	defer cli.CloseLog()

	cli.Log("%s", *message)
	cli.WriteJSON(map[string]any{
		"status":  "ok",
		"message": *message,
	})
}
