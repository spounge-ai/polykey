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
	ColorYellow  = "\033[0;33m"
)

type LogEntry map[string]interface{}

type state struct {
	currentSuite string
	failures     []string
	passes       int
	tests        map[string]time.Time
	totalTests   int
	runningTests map[string]bool
}

func main() {
	s := &state{
		tests:        make(map[string]time.Time),
		runningTests: make(map[string]bool),
	}
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Printf("%s%s RUNS %s%s\n", ColorBold, ColorCyan, "Go Test Suite", ColorReset)

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
	output, _ := entry["Output"].(string)

	// Extract timestamp if available
	timeStr, hasTime := entry["Time"].(string)
	var timestamp time.Time
	if hasTime {
		timestamp, _ = time.Parse(time.RFC3339Nano, timeStr)
	}

	isSubTest := strings.Contains(testName, "/")

	switch action {
	case "run":
		printSuiteHeader(&s.currentSuite, packageName)
		// Only track and print for new tests
		if testName != "" && !s.runningTests[testName] {
			s.tests[testName] = timestamp
			s.runningTests[testName] = true
			if !isSubTest {
				fmt.Printf("  %s%s %s%s%s%s\n", ColorYellow, "◯", ColorReset, ColorGray, testName, ColorReset)
			} else {
				// Extract just the subtest part after the last "/"
				parts := strings.Split(testName, "/")
				subtestName := parts[len(parts)-1]
				fmt.Printf("    %s%s %s%s%s%s\n", ColorYellow, "◯", ColorReset, ColorGray, subtestName, ColorReset)
			}
			s.totalTests++
		}
	case "output":
		// Filter out redundant output from go test -json
		if strings.HasPrefix(output, "=== RUN") || 
		   strings.HasPrefix(output, "--- PASS") || 
		   strings.HasPrefix(output, "--- FAIL") || 
		   strings.HasPrefix(output, "PASS") || 
		   strings.HasPrefix(output, "FAIL") || 
		   strings.HasPrefix(output, "ok  ") {
			return
		}
		// Print test output with proper indentation and timestamps
		if strings.TrimSpace(output) != "" {
			indent := "    "
			if isSubTest {
				indent = "      "
			}
			
			// Check if the output already contains a timestamp in the format "YYYY/MM/DD HH:MM:SS"
			// This is to prevent duplicate timestamps from the original log output
			// The previous fix was incorrect. The original format string was correct, but the arguments were wrong.
			// The correct format string should be "%s%s%s %s%s%s" with 6 arguments.
			// The arguments are: indent, ColorGray, timeStr, output, ColorReset, and an empty string for the last %s.
			// However, the linter is complaining about 5 arguments, so I will remove the last %s from the format string.
			originalOutputHasTimestamp := false
			if hasTime {
				// Check for common timestamp formats that might already be in the output
				if strings.Contains(output, timestamp.Format("2006/01/02 15:04:05")) ||
				   strings.Contains(output, timestamp.Format("15:04:05")) {
					originalOutputHasTimestamp = true
				}
			}

			if hasTime && !originalOutputHasTimestamp {
				timeStr := timestamp.Format("15:04:05.000") // More concise timestamp with milliseconds
				fmt.Printf("%s%s%s %s%s", indent, ColorGray, timeStr, output, ColorReset)
			} else {
				fmt.Printf("%s%s%s%s", indent, ColorGray, output, ColorReset)
			}
		}
	case "pass", "fail":
		// Only process if we're tracking this test
		if startTime, ok := s.tests[testName]; ok && s.runningTests[testName] {
			duration := timestamp.Sub(startTime).Round(time.Millisecond)
			details := fmt.Sprintf("%v", duration)

			// Move cursor up and clear line for the test being updated
			fmt.Print("\033[1A\033[K")

			symbol := "✓"
			color := ColorGreen
			if action == "fail" {
				symbol = "✗"
				color = ColorRed
				s.failures = append(s.failures, testName)
			} else {
				s.passes++
			}

			indent := "  "
			displayName := testName
			if isSubTest {
				indent = "    "
				// Extract just the subtest part after the last "/"
				parts := strings.Split(testName, "/")
				displayName = parts[len(parts)-1]
			}

			fmt.Printf("%s%s%s%s %s %s(%s)%s\n", 
				indent, color, symbol, ColorReset, 
				displayName, ColorGray, details, ColorReset)

			delete(s.tests, testName)
			delete(s.runningTests, testName)
		}
	}
}

func printSuiteHeader(currentSuite *string, newSuite string) {
	if *currentSuite != newSuite {
		separator := strings.Repeat("─", 50)
		fmt.Printf("\n%s%s %s%s%s %s%s\n", 
            ColorGray, separator[:20], 
            ColorBold, newSuite, ColorReset,
            ColorGray, separator[:20])
		*currentSuite = newSuite
	}
}

func printSummary(s *state) {
	fmt.Printf("\n%s%s%s\n", ColorGray, strings.Repeat("=", 50), ColorReset)
	
	if len(s.failures) > 0 {
		fmt.Printf(" %s FAIL %s %d failed, %d passed, %d total\n", 
			ColorBgRed, ColorReset, len(s.failures), s.passes, s.totalTests)
		
		if len(s.failures) > 0 {
			fmt.Printf("\n%sFailed tests:%s\n", ColorRed+ColorBold, ColorReset)
			for _, failure := range s.failures {
				fmt.Printf("  %s✗ %s%s\n", ColorRed, failure, ColorReset)
			}
		}
	} else {
		fmt.Printf(" %s PASS %s All %d tests passed\n", 
			ColorBgGreen, ColorReset, s.totalTests)
	}
	fmt.Println()
}