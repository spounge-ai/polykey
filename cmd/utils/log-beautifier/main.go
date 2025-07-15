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
	ColorGreen = "\033[0;32m"
	ColorRed   = "\033[0;31m"
	ColorGray  = "\033[0;90m"
	ColorCyan  = "\033[0;36m"
	ColorBold  = "\033[1m"
	ColorReset = "\033[0m"
)

type LogEntry map[string]interface{}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var currentSuite string
	pendingRPCs := make(map[string]time.Time)

	for scanner.Scan() {
		line := scanner.Text()

		jsonStart := strings.Index(line, "{")
		if jsonStart == -1 {
			fmt.Println(line)
			continue
		}
		jsonStr := line[jsonStart:]

		var entry LogEntry
		if err := json.Unmarshal([]byte(jsonStr), &entry); err != nil {
			fmt.Println(line)
			continue
		}
		processServerLogEntry(entry, &currentSuite, pendingRPCs)
	}
}

func processServerLogEntry(entry LogEntry, currentSuite *string, pendingRPCs map[string]time.Time) {
	msg, _ := entry["msg"].(string)
	method, _ := entry["method"].(string)

	switch msg {
	case "server starting":
		printSuiteHeader(currentSuite, "SETUP")
		details := fmt.Sprintf("addr=%v", entry["address"])
		printStep("PASS", "Server Listening", details)

	case "gRPC call received":
		printSuiteHeader(currentSuite, "CONNECTION")
		printStep("PASS", "gRPC Connection", method)
		printSuiteHeader(currentSuite, "EXECUTION")
		pendingRPCs[method] = time.Now()
		fmt.Printf("  %s %s%s\n", "○", ColorGray, method)

	case "gRPC call finished":
		startTime, ok := pendingRPCs[method]
		if !ok {
			return
		}
		delete(pendingRPCs, method)

		duration := time.Since(startTime).Round(time.Millisecond)
		code, _ := entry["code"].(string)
		status := "PASS"
		if code != "OK" {
			status = "FAIL"
		}
		printStep(status, method, duration.String())

	case "server shutting down", "server stopped":
		printSuiteHeader(currentSuite, "SHUTDOWN")
		printStep("PASS", msg, "")
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

	if status == "PASS" || status == "FAIL" {
		if _, err := os.Stdout.WriteString("\033[1A\033[K"); err != nil {
			// Ignoring error: terminal may not support cursor movement
			_ = err
		}
	}


	if details != "" {
		fmt.Printf("  %s%s%s %s %s(%s)%s\n", color, symbol, ColorReset, message, ColorGray, details, ColorReset)
	} else {
		fmt.Printf("  %s%s%s %s\n", color, symbol, ColorReset, message)
	}
}
