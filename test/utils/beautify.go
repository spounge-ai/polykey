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
	currentSuite string
	failures     []string
	passes       int
	tests        map[string]time.Time
}

func PrintJestReport(logLines []string) {
	s := &state{
		tests: make(map[string]time.Time),
	}
	isTestOutput := false
	isAppOutput := false

	fmt.Println()

	for i, line := range logLines {
		if line == "" {
			continue
		}
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
}

func processAppLogEntry(entry LogEntry, s *state) {
	msg, _ := entry["msg"].(string)
	level, _ := entry["level"].(string)

	if level == "DEBUG" {
		printSuiteHeader(&s.currentSuite, "CONNECTION")
		details := fmt.Sprintf("state=%v", entry["state"])
		fmt.Printf("    %s%s %s%s%s\n", ColorGray, msg, "...", details, ColorReset)
		return
	}

	switch {
	case msg == "Configuration loaded":
		printSuiteHeader(&s.currentSuite, "SETUP")
		details := fmt.Sprintf("server=%v", entry["server"])
		printStep("PASS", "Configuration", details)
		s.passes++
	case msg == "Network connectivity test passed":
		printSuiteHeader(&s.currentSuite, "CONNECTION")
		printStep("PASS", "Network Connectivity", "")
		s.passes++
	case msg == "gRPC connection established successfully":
		printSuiteHeader(&s.currentSuite, "CONNECTION")
		printStep("PASS", "gRPC Connection", "")
		s.passes++
	case msg == "Executing tool":
		printSuiteHeader(&s.currentSuite, "EXECUTION")
		details := fmt.Sprintf("tool=%v", entry["tool_name"])
		printStep("PASS", "Tool Execution", details)
		s.passes++
	case msg == "Tool execution completed":
		printSuiteHeader(&s.currentSuite, "EXECUTION")
		details := fmt.Sprintf("'%s'", entry["status_message"])
		fmt.Printf("    %s└─ Status: %s%s%s\n", ColorGray, ColorCyan, details, ColorReset)
	case msg == "Received struct output":
		printSuiteHeader(&s.currentSuite, "EXECUTION")
		details := fmt.Sprintf("fields=%v", entry["field_count"])
		fmt.Printf("    %s└─ Received Output %s(%s)%s\n", ColorGray, ColorGray, details, ColorReset)
	case msg == "Application failed":
		printSuiteHeader(&s.currentSuite, "ERROR")
		details := fmt.Sprintf("%v", entry["error"])
		printStep("FAIL", "Application Run", details)
		s.failures = append(s.failures, fmt.Sprintf("Application failed: %s", details))
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
		fmt.Printf("  %s %s%s\n", "○", ColorGray, testName)
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
		fmt.Printf("\n%s%s %s %s%s\n", ColorGray, separator, ColorBold+newSuite, separator, ColorReset)
		*currentSuite = newSuite
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
	fmt.Println(ColorGray + "\n" + strings.Repeat("=", 40) + ColorReset)
	if len(s.failures) > 0 {
		fmt.Printf(" %s FAIL %s %d failed, %d passed\n", ColorBgRed, ColorReset, len(s.failures), s.passes)
	} else {
		fmt.Printf(" %s PASS %s All %d checks passed\n", ColorBgGreen, ColorReset, s.passes)
	}
}