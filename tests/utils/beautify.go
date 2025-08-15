package utils


// need to fix ops timer orignal logging verus jest log time is off

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	ColorGreen   = "\033[0;32m"
	ColorRed     = "\033[0;31m"
	ColorGray    = "\033[0;90m"
	ColorCyan    = "\033[0;36m"
	ColorBold    = "\033[1m"
	ColorReset   = "\033[0m"
	ColorBgGreen = "\033[42;30m"
	ColorBgRed   = "\033[41;37m"
)

type LogEntry map[string]any

type state struct {
	currentSuite string
	failures     []string
	passes       int
	tests        map[string]time.Time
	startTime    time.Time
	lastStepTime time.Time
}

func PrintJestReport(logData string) bool {
	s := &state{
		tests: make(map[string]time.Time),
	}

	isTestOutput := false
	isAppOutput := false

	fmt.Println()

	logLines := strings.Split(strings.TrimSpace(logData), "\n")
	for i, line := range logLines {
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		if s.startTime.IsZero() {
			s.startTime = extractTimestampAsTime(entry)
			s.lastStepTime = s.startTime
		}

		if i == 0 {
			if _, ok := entry["Test"]; ok {
				isTestOutput = true
				fmt.Printf("%s RUNS %s\n", ColorBold+ColorCyan, "Go Test Suite"+ColorReset)
			}
			if _, ok := entry["msg"]; ok {
				isAppOutput = true
				fmt.Printf("%s RUNS %s\n", ColorBold+ColorCyan, "Polykey Dev Client"+ColorReset)
			}
		}

		if isTestOutput {
			processGoTestEntry(entry, s)
		} else if isAppOutput {
			processAppLogEntry(entry, s)
		}
	}

	printSummary(s)
	return len(s.failures) > 0
}

func processAppLogEntry(entry LogEntry, s *state) {
	msg, _ := entry["msg"].(string)
	timestamp := extractTimestamp(entry, s.lastStepTime)
	s.lastStepTime = extractTimestampAsTime(entry)

	errorVal, hasError := entry["error"]

	switch msg {
	case "Configuration loaded":
		printSuiteHeader(&s.currentSuite, "SETUP")
		details := fmt.Sprintf("server=%v", entry["server"])
		printStepWithTime("PASS", "Configuration", details, timestamp)
		s.passes++
	case "gRPC connection established successfully":
		printSuiteHeader(&s.currentSuite, "CONNECTION")
		printStepWithTime("PASS", "gRPC Connection", "", timestamp)
		s.passes++
	case "Authentication successful":
		printSuiteHeader(&s.currentSuite, "AUTHENTICATION")
		details := fmt.Sprintf("expires_in=%v", entry["expires_in"])
		printStepWithTime("PASS", "Client Authentication", details, timestamp)
		s.passes++
	case "HealthCheck successful":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("status=%v, version=%v", entry["status"], entry["version"])
		printStepWithTime("PASS", "Health Check", details, timestamp)
		s.passes++
	case "CreateKey successful":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("keyId=%v", entry["keyId"])
		printStepWithTime("PASS", "CreateKey", details, timestamp)
		s.passes++
	case "GetKey successful", "GetKey successful (pre-rotation)", "GetKey successful (post-rotation)":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("keyId=%v, version=%v", entry["keyId"], entry["version"])
		printStepWithTime("PASS", msg, details, timestamp)
		s.passes++
	case "RotateKey successful":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("keyId=%v, newVersion=%v", entry["keyId"], entry["newVersion"])
		printStepWithTime("PASS", "RotateKey", details, timestamp)
		s.passes++
	case "ListKeys successful":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("count=%v", entry["count"])
		printStepWithTime("PASS", "ListKeys", details, timestamp)
		s.passes++
	case "Starting key rotation validation":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		printStepWithTime("PASS", "Key Rotation Validation Started", "", timestamp)
		s.passes++
	case "Key ID preserved after rotation":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("keyId=%v", entry["keyId"])
		printStepWithTime("PASS", "Key ID Preserved", details, timestamp)
		s.passes++
	case "Key version incremented correctly":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("originalVersion=%v, rotatedVersion=%v", entry["originalVersion"], entry["rotatedVersion"])
		printStepWithTime("PASS", "Key Version Incremented", details, timestamp)
		s.passes++
	case "Key material successfully rotated":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		printStepWithTime("PASS", "Key Material Rotated", "", timestamp)
		s.passes++
	case "Key type preserved":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		details := fmt.Sprintf("keyType=%v", entry["keyType"])
		printStepWithTime("PASS", "Key Type Preserved", details, timestamp)
		s.passes++
	case "Key rotation validation completed":
		printSuiteHeader(&s.currentSuite, "HAPPY PATH")
		printStepWithTime("PASS", "Key Rotation Validation Completed", "", timestamp)
		s.passes++
	case "Unauthenticated access test passed":
		printSuiteHeader(&s.currentSuite, "ERROR CONDITIONS")
		details := fmt.Sprintf("gRPC_code=%v", entry["code"])
		printStepWithTime("PASS", "Rejects request with no token", details, timestamp)
		s.passes++
	case "Invalid token test passed":
		printSuiteHeader(&s.currentSuite, "ERROR CONDITIONS")
		details := fmt.Sprintf("gRPC_code=%v", entry["code"])
		printStepWithTime("PASS", "Rejects request with invalid token", details, timestamp)
		s.passes++
	default:
		if hasError && errorVal != nil {
			printSuiteHeader(&s.currentSuite, "ERROR")
			details := fmt.Sprintf("%v", errorVal)
			printStepWithTime("FAIL", msg, details, timestamp)
			s.failures = append(s.failures, fmt.Sprintf("%s: %s", msg, details))
		} else {
			printSuiteHeader(&s.currentSuite, "HAPPY PATH")
			details := ""
			if v, ok := entry["details"]; ok {
				details = fmt.Sprintf("%v", v)
			}
			printStepWithTime("PASS", msg, details, timestamp)
			s.passes++
		}
	}
}


func processGoTestEntry(entry LogEntry, s *state) {
	action, _ := entry["Action"].(string)
	testName, _ := entry["Test"].(string)
	packageName, _ := entry["Package"].(string)
	timestamp := extractTimestamp(entry, s.lastStepTime)
	s.lastStepTime = extractTimestampAsTime(entry)

	if testName == "" {
		return
	}

	switch action {
	case "run":
		printSuiteHeader(&s.currentSuite, packageName)
		s.tests[testName] = time.Now()
		if timestamp != "" {
			fmt.Printf("  %s %s%s%s %s[%s]%s\n", "○", ColorGray, testName, ColorReset, ColorGreen, timestamp, ColorReset)
		} else {
			fmt.Printf("  %s %s%s%s\n", "○", ColorGray, testName, ColorReset)
		}
	case "pass":
		duration := time.Since(s.tests[testName]).Round(time.Millisecond)
		details := fmt.Sprintf("%v", duration)
		printStepWithTime("PASS", testName, details, timestamp)
		s.passes++
	case "fail":
		duration := time.Since(s.tests[testName]).Round(time.Millisecond)
		details := fmt.Sprintf("%v", duration)
		printStepWithTime("FAIL", testName, details, timestamp)
		s.failures = append(s.failures, testName)
	}
}

func extractTimestamp(entry LogEntry, lastTime time.Time) string {
	entryTime := extractTimestampAsTime(entry)
	if entryTime.IsZero() {
		return ""
	}

	if lastTime.IsZero() {
		lastTime = entryTime
	}

	duration := entryTime.Sub(lastTime)
	return fmt.Sprintf("%.2fms", float64(duration.Nanoseconds())/1e6)
}

func extractTimestampAsTime(entry LogEntry) time.Time {
	if ts, ok := entry["time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			return t
		}
	}
	if ts, ok := entry["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			return t
		}
	}
	if ts, ok := entry["ts"].(string); ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			return t
		}
	}
	return time.Time{}
}

func printSuiteHeader(currentSuite *string, newSuite string) {
	if *currentSuite != newSuite {
		separator := strings.Repeat("─", 10)
		fmt.Printf("\n%s%s %s%s%s %s%s\n", ColorGray, separator, ColorBold, newSuite, ColorReset, separator, ColorReset)
		*currentSuite = newSuite
	}
}

func printStepWithTime(status, message, details, timestamp string) {
	var color, symbol string
	if status == "PASS" {
		color, symbol = ColorGreen, "✓"
	} else {
		color, symbol = ColorRed, "✗"
	}

	timeStr := ""
	if timestamp != "" {
		timeStr = fmt.Sprintf(" %s[%s]%s", ColorGreen, timestamp, ColorReset)
	}

	if details != "" {
		fmt.Printf("  %s%s%s %s %s(%s)%s%s\n", color, symbol, ColorReset, message, ColorGray, details, ColorReset, timeStr)
	} else {
		fmt.Printf("  %s%s%s %s%s\n", color, symbol, ColorReset, message, timeStr)
	}
}

func printSummary(s *state) {
	fmt.Printf("\n%s\n%s%s\n", ColorGray, strings.Repeat("=", 40), ColorReset)
	if len(s.failures) > 0 {
		fmt.Printf(" %s FAIL %s %d failed, %d passed\n", ColorBgRed, ColorReset, len(s.failures), s.passes)
	} else {
		fmt.Printf(" %s PASS %s All %d checks passed\n", ColorBgGreen, ColorReset, s.passes)
	}
}
