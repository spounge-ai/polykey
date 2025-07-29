package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
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

func main() {
	s := &state{
		tests: make(map[string]time.Time),
	}
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Printf("%s RUNS %s\n", ColorBold+ColorCyan, "Go Test Suite"+ColorReset)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Ignore non-JSON lines which are typically summary lines from go test
			continue
		}
		processGoTestEntry(entry, s)
	}

	printSummary(s)
}

func processGoTestEntry(entry LogEntry, s *state) {
	action, _ := entry["Action"].(string)
	testName, _ := entry["Test"].(string)
	packageName, _ := entry["Package"].(string)

	if testName == "" {
		return
	}

	isSubTest := strings.Contains(testName, "/")

	switch action {
	case "run":
		if !isSubTest {
			printSuiteHeader(&s.currentSuite, packageName)
			s.tests[testName] = time.Now()
			fmt.Printf("  %s %s%s\n", "\u25CB", ColorGray, testName)
		}
	case "pass":
		if !isSubTest {
			if _, ok := s.tests[testName]; !ok {
				return
			}
			duration := time.Since(s.tests[testName]).Round(time.Millisecond)
			details := fmt.Sprintf("%v", duration)
			fmt.Printf("\033[1A\033[K") // Move cursor up and clear line
			printStep("PASS", testName, details)
			s.passes++
			delete(s.tests, testName)
		}
	case "fail":
		if !isSubTest {
			if _, ok := s.tests[testName]; !ok {
				return
			}
			duration := time.Since(s.tests[testName]).Round(time.Millisecond)
			details := fmt.Sprintf("%v", duration)
			fmt.Printf("\033[1A\033[K") // Move cursor up and clear line
			printStep("FAIL", testName, details)
			s.failures = append(s.failures, testName)
			delete(s.tests, testName)
		}
	case "output":
		output, _ := entry["Output"].(string)
		fmt.Printf("    %s%s%s", ColorGray, output, ColorReset)
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
	totalTests := s.passes + len(s.failures)
	if len(s.failures) > 0 {
		fmt.Printf(" %s FAIL %s %d failed, %d passed, %d total\n", ColorBgRed, ColorReset, len(s.failures), s.passes, totalTests)
	} else {
		fmt.Printf(" %s PASS %s All %d tests passed\n", ColorBgGreen, ColorReset, totalTests)
	}
}
