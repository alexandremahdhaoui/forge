# Creating Test Runners

**Execute tests and generate structured reports for forge.**

> "I needed to integrate our custom test framework. I defined the schema, implemented the Run function, and forge now tracks all test results with coverage metrics."

## What problem does this solve?

Test runners execute test frameworks and produce structured `TestReport` results. When generic-test-runner is too limited (e.g., you need to parse custom output formats or coverage data), create a custom runner.

## What is a test runner?

A test runner is an MCP server that:
- Executes test frameworks with appropriate flags
- Captures test output and parses results
- Returns structured `TestReport` with stats and coverage

Test runners do NOT manage test environments - that's the testenv's job.

## What MCP tools must a test runner provide?

| Tool | Required | Description |
|------|----------|-------------|
| `run` | Yes | Execute tests and return TestReport |
| `config-validate` | Yes | Validate forge.yaml configuration |

**RunInput:**
```go
type RunInput struct {
    Stage string `json:"stage"` // Test stage (unit, integration, e2e)
    Name  string `json:"name"`  // Test run identifier
}
```

## What is the TestReport type?

```go
type TestReport struct {
    ID           string    `json:"id"`           // Unique report ID
    Stage        string    `json:"stage"`        // Test stage name
    Status       string    `json:"status"`       // "passed" or "failed"
    StartTime    time.Time `json:"startTime"`    // When tests started
    Duration     float64   `json:"duration"`     // Duration in seconds
    TestStats    TestStats `json:"testStats"`    // Pass/fail counts
    Coverage     Coverage  `json:"coverage"`     // Coverage percentage
    ErrorMessage string    `json:"errorMessage"` // Error details if failed
}

type TestStats struct {
    Total, Passed, Failed, Skipped int
}

type Coverage struct {
    Percentage float64
    FilePath   string
}
```

## How do I implement a test runner?

Use forge-dev. Create `forge-dev.yaml` and `spec.openapi.yaml` (see [forge-dev.md](./forge-dev.md)), then implement:

```go
func Run(ctx context.Context, input mcptypes.RunInput, spec *Spec) (*forge.TestReport, error) {
    startTime := time.Now()
    stats, coverage, err := runTests(spec.TestDir, spec.Timeout)

    status := "passed"
    if err != nil {
        status = "failed"
    }

    return &forge.TestReport{
        ID:        generateID(input.Stage),
        Stage:     input.Stage,
        Status:    status,
        StartTime: startTime,
        Duration:  time.Since(startTime).Seconds(),
        TestStats: stats,
        Coverage:  coverage,
    }, nil
}
```

**Add to forge.yaml:**
```yaml
test:
  - name: unit
    runner: go://my-runner
    spec:
      testDir: ./tests
      timeout: 300
```

## When to use generic-test-runner instead?

Use `generic-test-runner` when exit code determines pass/fail and no custom output parsing is needed.

Use a custom runner when you need to parse framework-specific output, coverage data, or custom test statistics.
