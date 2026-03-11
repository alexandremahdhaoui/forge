//go:build e2e || unit

// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testrunner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// jsonRPCRequest represents a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// jsonRPCResponse represents a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError represents a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// callToolResult holds the MCP tools/call response.
type callToolResult struct {
	Content []callToolContent `json:"content"`
}

// callToolContent holds a single content item from tools/call.
type callToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ExecuteMCPStep starts a forge MCP subprocess, sends an initialize
// request followed by a tools/call request, and returns the parsed result.
//
// Each MCP step starts a fresh subprocess (initialize + call + close).
func ExecuteMCPStep(binary string, step Step, data *TemplateData) (map[string]interface{}, error) {
	// Render input values.
	var renderedInput map[string]interface{}
	if step.Input != nil {
		var err error
		renderedInput, err = RenderMapValues(step.Input, data)
		if err != nil {
			return nil, fmt.Errorf("mcp step %q: rendering input: %w", step.Tool, err)
		}
	}

	// Start subprocess.
	cmd := exec.Command(binary, "--mcp")
	if data.CWD != "" {
		cmd.Dir = data.CWD
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp step %q: creating stdin pipe: %w", step.Tool, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp step %q: creating stdout pipe: %w", step.Tool, err)
	}
	// Capture stderr separately; the MCP server writes log lines there.
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp step %q: starting subprocess: %w", step.Tool, err)
	}

	reader := bufio.NewReader(stdout)

	// Send initialize request.
	initReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "forge-e2e-test",
				"version": "1.0.0",
			},
		},
	}

	if err := sendRequest(stdin, initReq); err != nil {
		cleanupProcess(stdin, cmd)
		return nil, fmt.Errorf("mcp step %q: sending initialize: %w", step.Tool, err)
	}

	// Read initialize response.
	if _, err := readResponse(reader); err != nil {
		cleanupProcess(stdin, cmd)
		return nil, fmt.Errorf("mcp step %q: reading initialize response: %w", step.Tool, err)
	}

	// Send tools/call request.
	callReq := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      step.Tool,
			"arguments": renderedInput,
		},
	}

	if err := sendRequest(stdin, callReq); err != nil {
		cleanupProcess(stdin, cmd)
		return nil, fmt.Errorf("mcp step %q: sending tools/call: %w", step.Tool, err)
	}

	// Read tools/call response.
	resp, err := readResponse(reader)
	if err != nil {
		cleanupProcess(stdin, cmd)
		return nil, fmt.Errorf("mcp step %q: reading tools/call response: %w", step.Tool, err)
	}

	// Close stdin and wait for exit.
	stdin.Close()
	_ = cmd.Wait()

	// Check for JSON-RPC error.
	if resp.Error != nil {
		return nil, fmt.Errorf("mcp step %q: JSON-RPC error %d: %s",
			step.Tool, resp.Error.Code, resp.Error.Message)
	}

	// Parse result.
	return extractMCPResult(resp.Result, step.Tool)
}

// sendRequest marshals and writes a JSON-RPC request followed by a newline.
func sendRequest(w io.Writer, req jsonRPCRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing request: %w", err)
	}
	return nil
}

// readResponse reads lines from the reader until it finds a valid JSON-RPC
// response. Non-JSON lines (e.g., MCP server notifications or log output
// that leaks to stdout) are skipped.
func readResponse(r *bufio.Reader) (*jsonRPCResponse, error) {
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("reading response line: %w", err)
		}

		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}

		// Skip lines that are not JSON objects.
		if !strings.HasPrefix(trimmed, "{") {
			continue
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(trimmed), &resp); err != nil {
			// Not a valid JSON-RPC response; skip.
			continue
		}

		return &resp, nil
	}
}

// extractMCPResult parses the tools/call result content and extracts
// the text content as a JSON map.
func extractMCPResult(raw json.RawMessage, tool string) (map[string]interface{}, error) {
	var tr callToolResult
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, fmt.Errorf("mcp step %q: parsing tool result: %w", tool, err)
	}

	if len(tr.Content) == 0 {
		return make(map[string]interface{}), nil
	}

	// Find the first text content item.
	for _, c := range tr.Content {
		if c.Type == "text" {
			// Try parsing as JSON first.
			var result map[string]interface{}
			if err := json.Unmarshal([]byte(c.Text), &result); err != nil {
				// Not JSON; store as raw text.
				return map[string]interface{}{"text": c.Text}, nil
			}
			return result, nil
		}
	}

	return make(map[string]interface{}), nil
}

// cleanupProcess closes stdin and waits for the process to exit.
func cleanupProcess(stdin io.WriteCloser, cmd *exec.Cmd) {
	stdin.Close()
	_ = cmd.Wait()
}
