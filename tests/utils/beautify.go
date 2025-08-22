package utils

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// --- Constants ---
const (
	ColorGreen   = "\033[0;32m"
	ColorRed     = "\033[0;31m"
	ColorYellow  = "\033[0;33m"
	ColorGray    = "\033[0;90m"
	ColorBold    = "\033[1m"
	ColorReset   = "\033[0m"
	ColorBgGreen = "\033[42;30m"
	ColorBgRed   = "\033[41;37m"

	SymbolPass = "✓"
	SymbolFail = "✗"
	SymbolSkip = "⚠"
)

// --- Structs ---

type LogEntry map[string]any

type TestState struct {
	currentSuite  string
	failures      int
	passes        int
	skips         int
	firstLogTime  time.Time
	lastLogTime   time.Time
}

type LogHandler struct {
	Suite    string
	TestName func(e LogEntry) string
	Status   func(e LogEntry) string // PASS, FAIL, SKIP
}

// --- Main Function ---

func PrintJestReport(logData string) bool {
	state := &TestState{}

	fmt.Printf("\n%sPolykey Dev Client Tests%s\n", ColorBold, ColorReset)

	logLines := strings.Split(strings.TrimSpace(logData), "\n")
	for _, line := range logLines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Set timestamps for total duration calculation
		entryTime := extractTimestampAsTime(entry)
		if !entryTime.IsZero() {
			if state.firstLogTime.IsZero() {
				state.firstLogTime = entryTime
			}
			state.lastLogTime = entryTime
		}

		if msg, ok := entry["msg"].(string); ok {
			if handler, exists := logHandlers[msg]; exists {
				processLogEntry(state, entry, handler)
			}
		}
	}

	printSummary(state)
	return state.failures > 0
}

// --- Processing Logic ---

func processLogEntry(s *TestState, entry LogEntry, handler LogHandler) {
	printSuiteHeader(s, handler.Suite)

	testName := handler.TestName(entry)
	status := handler.Status(entry)

	var durationMs float64
	if d, ok := entry["duration"].(float64); ok {
		durationMs = d / 1e6 // slog logs duration in nanoseconds
	}

	switch status {
	case "PASS":
		s.passes++
	case "FAIL":
		s.failures++
	case "SKIP":
		s.skips++
	}

	printTestResult(status, testName, durationMs)
}

// --- Presentation Logic ---

func printSuiteHeader(s *TestState, newSuite string) {
	if s.currentSuite != newSuite {
		separator := strings.Repeat("─", 10)
		fmt.Printf("\n%s%s %s %s%s\n", ColorGray, separator, newSuite, separator, ColorReset)
		s.currentSuite = newSuite
	}
}

func printTestResult(status, message string, durationMs float64) {
	var color, symbol string
	switch status {
	case "PASS":
		color, symbol = ColorGreen, SymbolPass
	case "FAIL":
		color, symbol = ColorRed, SymbolFail
	case "SKIP":
		color, symbol = ColorYellow, SymbolSkip
	default:
		color, symbol = ColorGray, "?"
	}

	timeStr := fmt.Sprintf("%s[%.2fms]%s", ColorGreen, durationMs, ColorReset)
	fmt.Printf("  %s%s%s %s %s\n", color, symbol, ColorReset, message, timeStr)
}

func printSummary(s *TestState) {
	total := s.passes + s.failures + s.skips
	var duration time.Duration
	if !s.firstLogTime.IsZero() && !s.lastLogTime.IsZero() {
		duration = s.lastLogTime.Sub(s.firstLogTime).Round(time.Millisecond)
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 40))

	if s.failures > 0 {
		fmt.Printf("%s FAIL %s %d failed, %d passed, %d skipped\n", ColorBgRed, ColorReset, s.failures, s.passes, s.skips)
	} else {
		fmt.Printf("%s PASS %s All %d checks executed\n", ColorBgGreen, ColorReset, total)
	}
	fmt.Printf("Total runtime ~%v\n", duration)
}

func extractTimestampAsTime(entry LogEntry) time.Time {
	if ts, ok := entry["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			return t
		}
	}
	return time.Time{}
}


// --- Handlers Map ---

var logHandlers = map[string]LogHandler{
	"Configuration loaded": {
		Suite: "SETUP",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("Configuration loaded (server=%v)", e["server"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"gRPC connection established successfully": {
		Suite:    "SETUP",
		TestName: func(e LogEntry) string { return "gRPC Connection established" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"Authentication successful": {
		Suite: "AUTHENTICATION",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("Client authenticated (expires_in=%.0fs)", e["expires_in"].(float64))
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"HealthCheck successful": {
		Suite: "HAPPY PATH",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("Health Check OK (status=%v, version=%v)", e["status"], e["version"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"CreateKey successful": {
		Suite: "HAPPY PATH",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("CreateKey success (id=%v)", e["keyId"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"GetKey successful": {
		Suite: "HAPPY PATH",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("GetKey success (id=%v, version=%.0f)", e["keyId"], e["version"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"Exists check for created key passed": {
		Suite:    "HAPPY PATH",
		TestName: func(e LogEntry) string { return "Exists (created key) OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"Exists check for non-existent key passed": {
		Suite:    "HAPPY PATH",
		TestName: func(e LogEntry) string { return "Exists (non-existent key) rejected" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"RotateKey successful": {
		Suite: "HAPPY PATH",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("RotateKey success (id=%v, newVersion=%.0f)", e["keyId"], e["newVersion"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"Key material successfully rotated": {
		Suite:    "HAPPY PATH",
		TestName: func(e LogEntry) string { return "Rotation validation OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"GetKey (cached) successful": {
		Suite:    "HAPPY PATH",
		TestName: func(e LogEntry) string { return "GetKey (cached) OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"ListKeys successful": {
		Suite: "HAPPY PATH",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("ListKeys (count=%.0f) OK", e["count"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"Unauthenticated access test passed": {
		Suite: "ERROR CONDITIONS",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("Rejects no token (gRPC=%v)", e["code"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"Invalid token test passed": {
		Suite: "ERROR CONDITIONS",
		TestName: func(e LogEntry) string {
			return fmt.Sprintf("Rejects invalid token (gRPC=%v)", e["code"])
		},
		Status: func(e LogEntry) string { return "PASS" },
	},
	"BatchCreateKeys successful": {
		Suite:    "BATCH OPERATIONS",
		TestName: func(e LogEntry) string { return "BatchCreateKeys OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"BatchGetKeys successful": {
		Suite:    "BATCH OPERATIONS",
		TestName: func(e LogEntry) string { return "BatchGetKeys OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"BatchGetKeyMetadata successful": {
	Suite:    "BATCH OPERATIONS",
	TestName: func(e LogEntry) string { return "BatchGetKeyMetadata OK" },
	Status:   func(e LogEntry) string { return "PASS" },
	},
	"BatchUpdateKeyMetadata successful": {
		Suite:    "BATCH OPERATIONS",
		TestName: func(e LogEntry) string { return "BatchUpdateKeyMetadata OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"BatchRotateKeys successful": {
		Suite:    "BATCH OPERATIONS",
		TestName: func(e LogEntry) string { return "BatchRotateKeys OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
	"BatchRevokeKeys successful": {
		Suite:    "BATCH OPERATIONS",
		TestName: func(e LogEntry) string { return "BatchRevokeKeys OK" },
		Status:   func(e LogEntry) string { return "PASS" },
	},
}