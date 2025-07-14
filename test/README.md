# Developer Guide: Running & Testing

This guide explains how to use the project's Makefile and custom logging framework for a streamlined development and testing workflow.

Our goal is to provide a single, unified, and human-readable output format for both running the application and executing the test suite. This is handled by a custom utility that replaces generic tools like tparse.

---

## ▶️ Running the Application

The primary way to run the `dev_client` is with a `make` command that automatically formats the output into a clean, Jest-style report.

### Commands

#### `make run-client`

This is the main command for local development. It runs the client and pipes its structured logs through our custom beautifier to produce a readable report.

**Example Output:**

```bash
▶ RUNS Polykey Dev Client

────────── SETUP ──────────
  ✓ Configuration (server=localhost:50051)

────────── CONNECTION ──────────
  ✓ Network Connectivity
    Initial connection state ... (state=IDLE)
    Connection state changed ... (state=CONNECTING)
    Connection state changed ... (state=READY)
  ✓ gRPC Connection

────────── EXECUTION ──────────
  ✓ Tool Execution (tool=example_tool)
    └─ Status: 'Tool 'example_tool' executed successfully'
    └─ Received Output (fields=3)

========================================
 PASS  All 4 checks passed
```

---

#### `make run-client-json`

Use this command if you need to see the raw, machine-readable JSON logs. This is useful for debugging the log output itself.

---

## 🧪 Running Tests

The test commands use the exact same beautifier to provide a consistent look and feel. This replaces the need for external tools like `tparse`.

### Commands

- `make test`: Runs all unit tests.  
- `make test-race`: Runs all unit tests with the race detector enabled.  
- `make test-integration`: Runs the integration test suite against a live Docker environment.

**Example Test Output:**

```bash
▶ RUNS Go Test Suite

────────── github.com/SpoungeAI/polykey-service/internal/config ──────────
  ✓ TestDetectRuntime (0.1ms)
  ✓ TestLoadConfig (0.2ms)

────────── github.com/SpoungeAI/polykey-service/cmd/dev_client ──────────
  ✓ TestSomethingInClient (0.1ms)

========================================
 PASS  All 3 tests passed
```

---

## ⚙️ Beautifier Internals: How It Works

The custom log formatter is a simple Go program located at `test/utils/beautify.go`. It is designed to read a stream of JSON data from standard input, parse it line-by-line, and print a formatted, human-readable report to standard output.

---

### Input Data Format

The beautifier is designed to consume newline-delimited JSON. It intelligently detects one of two formats:

**Application Logs (slog)** – Standard structured logs from our application. The key field it looks for is `"msg"`.

```json
{"time":"...","level":"INFO","msg":"Configuration loaded","server":"localhost:50051"}
```

**Test Logs (go test -json)** – The standard output from Go's test runner. The key fields it looks for are `"Test"` and `"Action"`.

```json
{"Time":"...","Action":"pass","Package":"...","Test":"TestLoadConfig","Elapsed":0.00}
```

---

### How to Call It

The beautifier is invoked by piping the JSON output of another command into it. This pattern is configured in the Makefile.

**Generic Pattern:**

```bash
<some-command> | go run ./test/utils/beautify.go
```

**Makefile Example (make test):**

```makefile
test:
	@$(GO) test -v -json ./... | $(GO) run ./test/utils/beautify.go
```

This command runs the tests, outputs the results as JSON, and immediately sends that output to the beautifier for formatting.

---

## 🔧 How to Extend the Beautifier

Adding a new formatted step to the report is a simple, two-step process.

---

### The Concept

- The application (`cmd/dev_client/main.go`) logs important events as structured JSON to an in-memory buffer.  
- The beautifier utility (`test/utils/beautify.go`) reads these JSON logs and looks for specific `msg` fields to decide what to print.

---

### Step 1: Add a Log Message in Your Application

In your application code (e.g., in `cmd/dev_client/main.go`), add a new, specific log message for the event you want to report.

**Example: Let's say you want to add a step for "Validating Parameters".**

```go
// In a function within main.go
func executeTestRequest(...) error {
    // ... existing code ...

    logger.Info("Parameters validated successfully") // <-- ADD THIS LINE

    resp, err := client.ExecuteTool(requestCtx, req)
    // ...
}
```

---

### Step 2: Add a Case in the Beautifier

Now, open `test/utils/beautify.go` and add a case to the switch statement inside the `processAppLogEntry` function to handle your new message.

```go
// In test/utils/beautify.go

func processAppLogEntry(entry LogEntry, s *state) {
    // ... existing code ...
	switch {
    // ... existing cases ...

    // ADD THIS NEW CASE
    case msg == "Parameters validated successfully":
        printSuiteHeader(&s.currentSuite, "EXECUTION") // Or a new suite like "VALIDATION"
        printStep("PASS", "Parameter Validation", "")
        s.passes++

    // ... existing cases ...
    }
}
```

Now when you run:

```bash
make run-client
```

You’ll see your new **"Parameter Validation"** step in the report — whether it passes or fails.
````
