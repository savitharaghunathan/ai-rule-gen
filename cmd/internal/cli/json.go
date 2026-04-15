package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// WriteJSON marshals v as indented JSON and prints it to stdout.
func WriteJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
