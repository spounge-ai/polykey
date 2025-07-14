package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ANSI color codes for pretty printing
const (
	ColorGreen = "\033[0;32m"
	ColorRed   = "\033[0;31m"
	ColorGray  = "\033[0;90m"
	ColorReset = "\033[0m"
)

// LogEntry matches the structure of slog's JSON output.
type LogEntry map[string]interface{}

// PrintJestReport takes a slice of JSON log lines, parses them,
// and prints a formatted, Jest-style report.
func PrintJestReport(logLines []string) {
	successCount := 0
	failureCount := 0
	var finalError error

	// Print header
	fmt.Println("\n▶ REPORT for", ColorGray, "polykey-dev-client")

	for _, line := range logLines {
		if line == "" {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip non-JSON lines
		}

		ok, isFailure := formatLogEntry(entry)
		if ok {
			successCount++
		}
		if isFailure {
			failureCount++
			// Capture the error message for the summary
			if err, exists := entry["error"].(string); exists {
				finalError = fmt.Errorf("%s", err)
			}
		}
	}

	// Print the final summary
	fmt.Println("\n---")
	if failureCount > 0 {
		fmt.Printf("%sFAIL%s: %d failed, %d passed\n", ColorRed, ColorReset, failureCount, successCount)
		if finalError != nil {
			fmt.Printf("  └─ %sError: %v%s\n", ColorRed, finalError, ColorReset)
		}
	} else {
		fmt.Printf("%sPASS%s: %d total checks passed\n", ColorGreen, ColorReset, successCount)
	}
}

// formatLogEntry prints a single formatted line and returns if it was a "success" event
// and if it was a "failure" event.
func formatLogEntry(entry LogEntry) (isSuccess bool, isFailure bool) {
	msg, ok := entry["msg"].(string)
	if !ok {
		return false, false // Not a parsable event
	}

	switch {
	case strings.HasSuffix(msg, "passed"):
		fmt.Printf("  %sPASS%s %s\n", ColorGreen, ColorReset, msg)
		return true, false
	case strings.HasSuffix(msg, "completed successfully"):
		// This is a final summary message, ignore for individual line printing
		return false, false
	case msg == "Application failed":
		errMsg := entry["error"]
		fmt.Printf("  %sFAIL%s %s: %v\n", ColorRed, ColorReset, msg, errMsg)
		return false, true // This is a failure event
	case strings.Contains(msg, "Executing tool"):
		toolName := entry["tool_name"]
		fmt.Printf("  %sPASS%s %s: %v\n", ColorGreen, ColorReset, msg, toolName)
		return true, false
	default:
		// Ignore other messages for the report view (e.g., "Starting client...")
		return false, false
	}
}