package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Flags controls output format.
var (
	JSONMode  bool
	AgentMode bool
)

// JSON prints data as JSON.
func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// Error prints an error with optional hint.
func Error(msg, hint string) {
	if JSONMode {
		JSON(map[string]any{"success": false, "error": msg, "hint": hint})
		return
	}
	fmt.Fprintf(os.Stderr, "❌ %s\n", msg)
	if hint != "" {
		fmt.Fprintf(os.Stderr, "  💡 %s\n", hint)
	}
}

// Success prints a success message.
func Success(msg string) {
	if JSONMode {
		JSON(map[string]any{"success": true, "message": msg})
		return
	}
	fmt.Printf("✅ %s\n", msg)
}

// Info prints an info line (suppressed in JSON/agent mode).
func Info(msg string) {
	if JSONMode || AgentMode {
		return
	}
	fmt.Println(msg)
}

// Warning prints a warning message.
func Warning(msg string) {
	if JSONMode {
		return
	}
	fmt.Fprintf(os.Stderr, "⚠️  %s\n", msg)
}

// StatusLine prints an overwritable status line (suppressed in JSON/agent mode).
func StatusLine(msg string) {
	if JSONMode || AgentMode {
		return
	}
	fmt.Printf("\r\033[K%s", msg)
}
