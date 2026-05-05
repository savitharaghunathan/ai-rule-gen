package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// WriteJSON marshals v as indented JSON and prints it to stdout.
func WriteJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// ErrorResponse is a standard machine-readable command error payload.
type ErrorResponse struct {
	Status  string      `json:"status"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Step    string      `json:"step,omitempty"`
	Hint    string      `json:"hint,omitempty"`
	Details any `json:"details,omitempty"`
}

// WriteError prints a standardized error payload to stderr.
func WriteError(code, message, step, hint string, details any) {
	payload := ErrorResponse{
		Status:  "error",
		Code:    code,
		Message: message,
		Step:    step,
		Hint:    hint,
		Details: details,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, string(data))
}

// Fail writes a standardized error payload and exits.
func Fail(code, message, step, hint string, details any) {
	WriteError(code, message, step, hint, details)
	os.Exit(1)
}
