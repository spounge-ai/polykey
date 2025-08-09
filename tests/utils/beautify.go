package utils

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

type LogEntry map[string]interface{}

type state struct {
	currentSuite  string
	failures      []string
	passes        int
	tests         map[string]time.Time
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

	switch msg {
	case "Configuration loaded":
		printSuiteHeader(&s.currentSuite, "SETUP")
		details := fmt.Sprintf("server=%v", entry["server"])
		printStep("PASS", "Configuration", details)
		s.passes++
	case "gRPC connection established successfully":
		printSuiteHeader(&s.currentSuite, "CONNECTION")
		printStep("PASS", "gRPC Connection", "")
		s.passes++
	case "HealthCheck successful":
		printSuiteHeader(&s.currentSuite, "HEALTH CHECK")
		details := fmt.Sprintf("status=%v, version=%v", entry["status"], entry["version"])
		printStep("PASS", "Health Check", details)
		s.passes++
	case "CreateKey successful":
		printSuiteHeader(&s.currentSuite, "EXECUTION")
		details := fmt.Sprintf("keyId=%v", entry["keyId"])
		printStep("PASS", "CreateKey", details)
		s.passes++
		
	case "GetKey successful":
		printSuiteHeader(&s.currentSuite, "EXECUTION")
		details := fmt.Sprintf("keyId=%v", entry["keyId"])
		printStep("PASS", "GetKey", details)
		s.passes++
		
	case "RotateKey successful":
		printSuiteHeader(&s.currentSuite, "EXECUTION")
		details := fmt.Sprintf("keyId=%v, newVersion=%v", entry["keyId"], entry["newVersion"])
		printStep("PASS", "RotateKey", details)
		s.passes++
	case "ListKeys successful":
		printSuiteHeader(&s.currentSuite, "EXECUTION")
		details := fmt.Sprintf("count=%v", entry["count"])
		printStep("PASS", "ListKeys", details)
		s.passes++

	case "CreateKey failed", "GetKey failed", "HealthCheck failed", "gRPC connection failed", "RotateKey failed", "ListKeys failed", "failed to load TLS credentials":
		printSuiteHeader(&s.currentSuite, "ERROR")
		details := fmt.Sprintf("%v", entry["error"])
		printStep("FAIL", msg, details)
		s.failures = append(s.failures, fmt.Sprintf("%s: %s", msg, details))
	}
}



func processGoTestEntry(entry LogEntry, s *state) {
	action, _ := entry["Action"].(string)
	testName, _ := entry["Test"].(string)
	packageName, _ := entry["Package"].(string)

	if testName == "" {
		return
	}

	switch action {
	case "run":
		printSuiteHeader(&s.currentSuite, packageName)
		s.tests[testName] = time.Now()
		fmt.Printf("  %s %s%s%s\n", "○", ColorGray, testName, ColorReset)
	case "pass":
		duration := time.Since(s.tests[testName]).Round(time.Millisecond)
		details := fmt.Sprintf("%v", duration)
		printStep("PASS", testName, details)
		s.passes++
	case "fail":
		duration := time.Since(s.tests[testName]).Round(time.Millisecond)
		details := fmt.Sprintf("%v", duration)
		printStep("FAIL", testName, details)
		s.failures = append(s.failures, testName)
	}
}

func printSuiteHeader(currentSuite *string, newSuite string) {
	if *currentSuite != newSuite {
		separator := strings.Repeat("─", 10)
		fmt.Printf("\n%s%s %s%s%s %s%s\n", ColorGray, separator, ColorBold, newSuite, ColorReset, separator, ColorReset)
	}
}

func printStep(status, message, details string) {
	var color, symbol string
	if status == "PASS" {
		color, symbol = ColorGreen, "✓"
	} else {
		color, symbol = ColorRed, "✗"
	}
	if details != "" {
		fmt.Printf("  %s%s%s %s %s(%s)%s\n", color, symbol, ColorReset, message, ColorGray, details, ColorReset)
	} else {
		fmt.Printf("  %s%s%s %s\n", color, symbol, ColorReset, message)
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
