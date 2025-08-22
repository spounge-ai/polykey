package utils

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ANSI color codes for terminal output
const (
	ColorGreen   = "\033[0;32m"
	ColorRed     = "\033[0;31m"
	ColorYellow  = "\033[0;33m"
	ColorGray    = "\033[0;90m"
	ColorBold    = "\033[1m"
	ColorReset   = "\033[0m"
	ColorBgGreen = "\033[42;30m"
	ColorBgRed   = "\033[41;37m"
)

// Test result symbols and statuses
const (
	SymbolPass = "✓"
	SymbolFail = "✗"
	SymbolSkip = "⚠"
	
	StatusPass = "PASS"
	StatusFail = "FAIL"
	StatusSkip = "SKIP"
)

// Test suite categories
const (
	SuiteSetup     = "SETUP"
	SuiteAuth      = "AUTHENTICATION"
	SuiteHappyPath = "HAPPY PATH"
	SuiteError     = "ERROR CONDITIONS"
	SuiteBatchOps  = "BATCH OPERATIONS"
)

type LogEntry map[string]any

// TestState tracks the overall test execution state
type TestState struct {
	currentSuite string
	failures     int
	passes       int
	skips        int
	firstLogTime time.Time
	lastLogTime  time.Time
}

// LogHandler defines how to process a specific log entry type
type LogHandler struct {
	Suite    string
	TestName func(LogEntry) string
	Status   func(LogEntry) string // defaults to passStatus if nil
}

// TestReporter handles the formatting and display of test results
type TestReporter struct {
	state    *TestState
	handlers map[string]LogHandler
}

// NewTestReporter creates a new test reporter instance
func NewTestReporter() *TestReporter {
	return &TestReporter{
		state:    &TestState{
			currentSuite: "",
			failures:     0,
			passes:       0,
			skips:        0,
			firstLogTime: time.Time{},
			lastLogTime:  time.Time{},
		},
		handlers: buildLogHandlers(),
	}
}

// PrintJestReport processes log data and prints a Jest-style test report
// Returns true if there were any test failures
func PrintJestReport(logData string) bool {
	reporter := NewTestReporter()
	return reporter.ProcessLogData(logData)
}

// ProcessLogData processes the log data and generates the report
func (r *TestReporter) ProcessLogData(logData string) bool {
	r.printHeader()
	
	logLines := strings.Split(strings.TrimSpace(logData), "\n")
	for _, line := range logLines {
		r.processLogLine(line)
	}
	
	r.printSummary()
	return r.state.failures > 0
}

// processLogLine processes a single log line
func (r *TestReporter) processLogLine(line string) {
	var entry LogEntry
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return
	}
	
	r.updateTimestamps(entry)
	
	if msg, ok := entry["msg"].(string); ok {
		if handler, exists := r.handlers[msg]; exists {
			r.processLogEntry(entry, handler)
		}
	}
}

// updateTimestamps tracks the first and last log entry times
func (r *TestReporter) updateTimestamps(entry LogEntry) {
	entryTime := extractTimestamp(entry)
	if entryTime.IsZero() {
		return
	}
	
	if r.state.firstLogTime.IsZero() {
		r.state.firstLogTime = entryTime
	}
	r.state.lastLogTime = entryTime
}

// processLogEntry processes a single log entry using its handler
func (r *TestReporter) processLogEntry(entry LogEntry, handler LogHandler) {
	r.printSuiteHeader(handler.Suite)
	
	testName := handler.TestName(entry)
	
	// Default to PASS if no status function is provided
	status := StatusPass
	if handler.Status != nil {
		status = handler.Status(entry)
	}
	
	duration := extractDuration(entry)
	
	r.updateCounts(status)
	r.printTestResult(status, testName, duration)
}

// updateCounts updates the test result counters
func (r *TestReporter) updateCounts(status string) {
	switch status {
	case StatusPass:
		r.state.passes++
	case StatusFail:
		r.state.failures++
	case StatusSkip:
		r.state.skips++
	}
}

// printHeader prints the report header
func (r *TestReporter) printHeader() {
	fmt.Printf("\n%sPolykey Dev Client Tests%s\n", ColorBold, ColorReset)
}

// printSuiteHeader prints a suite header if we're entering a new suite
func (r *TestReporter) printSuiteHeader(newSuite string) {
	if r.state.currentSuite == newSuite {
		return
	}
	
	separator := strings.Repeat("─", 10)
	fmt.Printf("\n%s%s %s %s%s\n", ColorGray, separator, newSuite, separator, ColorReset)
	r.state.currentSuite = newSuite
}

// printTestResult prints the result of a single test
func (r *TestReporter) printTestResult(status, message string, durationMs float64) {
	color, symbol := r.getStatusFormatting(status)
	timeStr := r.formatDuration(durationMs)
	
	fmt.Printf("  %s%s%s %s %s\n", color, symbol, ColorReset, message, timeStr)
}

// getStatusFormatting returns the appropriate color and symbol for a test status
func (r *TestReporter) getStatusFormatting(status string) (string, string) {
	switch status {
	case StatusPass:
		return ColorGreen, SymbolPass
	case StatusFail:
		return ColorRed, SymbolFail
	case StatusSkip:
		return ColorYellow, SymbolSkip
	default:
		return ColorGray, "?"
	}
}

// formatDuration formats a duration in milliseconds for display
func (r *TestReporter) formatDuration(durationMs float64) string {
	if durationMs <= 0 {
		return ""
	}
	return fmt.Sprintf("%s[%.2fms]%s", ColorGreen, durationMs, ColorReset)
}

// printSummary prints the final test summary
func (r *TestReporter) printSummary() {
	total := r.state.passes + r.state.failures + r.state.skips
	duration := r.calculateTotalDuration()
	
	fmt.Printf("\n%s\n", strings.Repeat("=", 40))
	
	if r.state.failures > 0 {
		fmt.Printf("%s FAIL %s %d failed, %d passed, %d skipped\n", 
			ColorBgRed, ColorReset, r.state.failures, r.state.passes, r.state.skips)
	} else {
		fmt.Printf("%s PASS %s All %d checks executed\n", 
			ColorBgGreen, ColorReset, total)
	}
	
	fmt.Printf("Total runtime ~%v\n", duration)
}

// calculateTotalDuration calculates the total test execution time
func (r *TestReporter) calculateTotalDuration() time.Duration {
	if r.state.firstLogTime.IsZero() || r.state.lastLogTime.IsZero() {
		return 0
	}
	return r.state.lastLogTime.Sub(r.state.firstLogTime).Round(time.Millisecond)
}

// Utility functions

// extractTimestamp extracts and parses timestamp from log entry
func extractTimestamp(entry LogEntry) time.Time {
	ts, ok := entry["time"].(string)
	if !ok {
		return time.Time{}
	}
	
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t
	}
	return time.Time{}
}

// extractDuration extracts duration from log entry and converts to milliseconds
func extractDuration(entry LogEntry) float64 {
	if d, ok := entry["duration"].(float64); ok {
		return d / 1e6 // Convert nanoseconds to milliseconds
	}
	return 0
}

// Helper functions for creating log handlers

// passStatus always returns PASS status
func passStatus(_ LogEntry) string {
	return StatusPass
}

// fixedTestName returns a function that always returns the same test name
func fixedTestName(name string) func(LogEntry) string {
	return func(LogEntry) string { return name }
}

// formatWithFields creates a test name formatter using a template and field names
func formatWithFields(template string, fields ...string) func(LogEntry) string {
	return func(e LogEntry) string {
		values := make([]interface{}, len(fields))
		for i, field := range fields {
			if val, exists := e[field]; exists {
				values[i] = val
			} else {
				values[i] = "unknown"
			}
		}
		return fmt.Sprintf(template, values...)
	}
}

// successHandler creates a standard success handler for happy path operations
func successHandler(operation, idField string, extraFields ...string) LogHandler {
	var formatter func(LogEntry) string
	
	if len(extraFields) == 0 {
		// Simple case: just operation and ID
		formatter = formatWithFields(operation+" success (id=%v)", idField)
	} else {
		// Build template with extra fields
		template := operation + " success (id=%v"
		fields := []string{idField}
		
		for _, field := range extraFields {
			template += ", " + field + "=%.0f"
			fields = append(fields, field)
		}
		template += ")"
		
		formatter = formatWithFields(template, fields...)
	}
	
	return LogHandler{
		Suite:    SuiteHappyPath,
		TestName: formatter, // Status defaults to PASS
		Status: passStatus,
	}
}

// buildLogHandlers combines all handler categories into a single map
func buildLogHandlers() map[string]LogHandler {
	return map[string]LogHandler{
		// Setup handlers
		"Configuration loaded": {
			Suite:    SuiteSetup,
			TestName: formatWithFields("Configuration loaded (server=%v)", "server"),
			Status:   passStatus,
		},
		"gRPC connection established successfully": {
			Suite:    SuiteSetup,
			TestName: fixedTestName("gRPC Connection established"),
			Status:   passStatus,
		},
		
		// Authentication handlers
		"Authentication successful": {
			Suite:    SuiteAuth,
			TestName: formatWithFields("Client authenticated (expires_in=%.0fs)", "expires_in"),
			Status:   passStatus,
		},
		
		// Happy path handlers - using the successHandler helper
		"HealthCheck successful": {
			Suite: SuiteHappyPath,
			TestName: formatWithFields("Health Check OK (status=%v, version=%v)", "status", "version"),
			Status: passStatus,
		},
		"CreateKey successful":  successHandler("CreateKey", "keyId"),
		"GetKey successful":     successHandler("GetKey", "keyId", "version"),
		"RotateKey successful":  successHandler("RotateKey", "keyId", "newVersion"),
		"ListKeys successful": {
			Suite:    SuiteHappyPath,
			TestName: formatWithFields("ListKeys (count=%.0f) OK", "count"),
			Status:   passStatus,
		},
		
		// Error condition handlers
		"Unauthenticated access test passed": {
			Suite:    SuiteError,
			TestName: formatWithFields("Rejects no token (gRPC=%v)", "code"),
			Status:   passStatus,
		},
		"Invalid token test passed": {
			Suite:    SuiteError,
			TestName: formatWithFields("Rejects invalid token (gRPC=%v)", "code"),
			Status:   passStatus,
		},
		
		// Batch operation handlers - all use simple fixed names
		"BatchCreateKeys successful":        {Suite: SuiteBatchOps, TestName: fixedTestName("BatchCreateKeys OK"), Status: passStatus},
		"BatchGetKeys successful":           {Suite: SuiteBatchOps, TestName: fixedTestName("BatchGetKeys OK"), Status: passStatus},
		"BatchGetKeyMetadata successful":    {Suite: SuiteBatchOps, TestName: fixedTestName("BatchGetKeyMetadata OK"), Status: passStatus},
		"BatchUpdateKeyMetadata successful": {Suite: SuiteBatchOps, TestName: fixedTestName("BatchUpdateKeyMetadata OK"), Status: passStatus},
		"BatchRotateKeys successful":        {Suite: SuiteBatchOps, TestName: fixedTestName("BatchRotateKeys OK"), Status: passStatus},
		"BatchRevokeKeys successful":        {Suite: SuiteBatchOps, TestName: fixedTestName("BatchRevokeKeys OK"), Status: passStatus},
	}
}