package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// helper to run a sequence of JSON-RPC requests and return responses.
func runRequests(t *testing.T, spec string, examples map[string]string, requests ...string) []Response {
	t.Helper()

	input := strings.Join(requests, "\n") + "\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	transport := NewTransport(reader, &output)
	server := NewServer(transport, spec, examples)

	if err := server.Run(); err != nil {
		t.Fatalf("server.Run() error: %v", err)
	}

	// Parse responses
	var responses []Response
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if line == "" {
			continue
		}
		var resp Response
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("failed to parse response %q: %v", line, err)
		}
		responses = append(responses, resp)
	}
	return responses
}

func TestInitialize(t *testing.T) {
	responses := runRequests(t, "test spec", nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
	)

	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	// Check result has server info
	resultBytes, _ := json.Marshal(resp.Result)
	var result InitializeResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result.ServerInfo.Name != "human-compiler" {
		t.Errorf("server name = %q, want %q", result.ServerInfo.Name, "human-compiler")
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability")
	}
}

func TestToolsList(t *testing.T) {
	responses := runRequests(t, "test spec", nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	)

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	resp := responses[1]
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolsListResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(result.Tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(result.Tools))
	}

	names := make(map[string]bool)
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}

	expected := []string{"human_build", "human_validate", "human_ir", "human_examples", "human_spec", "human_read_file"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestPing(t *testing.T) {
	responses := runRequests(t, "", nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}`,
	)

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	resp := responses[1]
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestUnknownMethod(t *testing.T) {
	responses := runRequests(t, "", nil,
		`{"jsonrpc":"2.0","id":1,"method":"unknown/method","params":{}}`,
	)

	if len(responses) != 1 {
		t.Fatalf("expected 1 response, got %d", len(responses))
	}

	resp := responses[0]
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != ErrCodeMethodNot {
		t.Errorf("error code = %d, want %d", resp.Error.Code, ErrCodeMethodNot)
	}
}

func TestHumanValidate(t *testing.T) {
	source := `app Test is a web application`
	args, _ := json.Marshal(map[string]string{"source": source})

	responses := runRequests(t, "", nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"human_validate","arguments":`+string(args)+`}}`,
	)

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	resp := responses[1]
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result CallToolResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "Valid") && !strings.Contains(text, "valid") && !strings.Contains(text, "diagnostic") {
		t.Errorf("expected validation result, got: %s", text)
	}
}

func TestHumanSpec(t *testing.T) {
	specText := "# Human Language Spec\nThis is a test spec."

	responses := runRequests(t, specText, nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"human_spec","arguments":{}}}`,
	)

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	resp := responses[1]
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result CallToolResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	if result.IsError {
		t.Error("unexpected tool error")
	}
	if result.Content[0].Text != specText {
		t.Errorf("spec text = %q, want %q", result.Content[0].Text, specText)
	}
}

func TestHumanExamplesList(t *testing.T) {
	examples := map[string]string{
		"taskflow": "app TaskFlow is a web application",
		"blog":     "app Blog is a web application",
	}

	responses := runRequests(t, "", examples,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"human_examples","arguments":{}}}`,
	)

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	resp := responses[1]
	resultBytes, _ := json.Marshal(resp.Result)
	var result CallToolResult
	json.Unmarshal(resultBytes, &result)

	text := result.Content[0].Text
	if !strings.Contains(text, "taskflow") || !strings.Contains(text, "blog") {
		t.Errorf("expected example names in output, got: %s", text)
	}
}

func TestHumanExamplesGet(t *testing.T) {
	examples := map[string]string{
		"taskflow": "app TaskFlow is a web application",
	}

	args, _ := json.Marshal(map[string]string{"name": "taskflow"})

	responses := runRequests(t, "", examples,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"human_examples","arguments":`+string(args)+`}}`,
	)

	resp := responses[1]
	resultBytes, _ := json.Marshal(resp.Result)
	var result CallToolResult
	json.Unmarshal(resultBytes, &result)

	if result.IsError {
		t.Errorf("unexpected tool error: %s", result.Content[0].Text)
	}
	if !strings.Contains(result.Content[0].Text, "app TaskFlow") {
		t.Errorf("expected example source, got: %s", result.Content[0].Text)
	}
}

func TestHumanIR(t *testing.T) {
	source := `app Test is a web application

data User:
  name is text, required
  email is email, required`

	args, _ := json.Marshal(map[string]string{"source": source})

	responses := runRequests(t, "", nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"human_ir","arguments":`+string(args)+`}}`,
	)

	resp := responses[1]
	resultBytes, _ := json.Marshal(resp.Result)
	var result CallToolResult
	json.Unmarshal(resultBytes, &result)

	if result.IsError {
		t.Errorf("unexpected tool error: %s", result.Content[0].Text)
	}

	text := result.Content[0].Text
	if !strings.Contains(text, "name: Test") && !strings.Contains(text, "name: \"Test\"") {
		t.Errorf("expected IR YAML with app name, got: %s", text[:min(200, len(text))])
	}
}

func TestHumanReadFileNoBuilt(t *testing.T) {
	args, _ := json.Marshal(map[string]string{"path": "some/file.txt"})

	responses := runRequests(t, "", nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"human_read_file","arguments":`+string(args)+`}}`,
	)

	resp := responses[1]
	resultBytes, _ := json.Marshal(resp.Result)
	var result CallToolResult
	json.Unmarshal(resultBytes, &result)

	if !result.IsError {
		t.Error("expected error when no build has been done")
	}
	if !strings.Contains(result.Content[0].Text, "No build output") {
		t.Errorf("expected 'no build output' message, got: %s", result.Content[0].Text)
	}
}

func TestHumanReadFilePathTraversal(t *testing.T) {
	// Set up a server with a fake build dir
	input := ""
	reader := strings.NewReader(input)
	var output bytes.Buffer
	transport := NewTransport(reader, &output)
	server := NewServer(transport, "", nil)
	server.lastBuildDir = "/tmp/fake-build"

	result := server.handleReadFile([]byte(`{"path":"../../etc/passwd"}`))
	if !result.IsError {
		t.Error("expected error for path traversal")
	}
	if !strings.Contains(result.Content[0].Text, "Invalid path") {
		t.Errorf("expected path traversal rejection, got: %s", result.Content[0].Text)
	}
}

func TestNotificationsInitializedNoResponse(t *testing.T) {
	// notifications/initialized should produce no response
	responses := runRequests(t, "", nil,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping","params":{}}`,
	)

	// Should get 2 responses: initialize + ping (not 3)
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses (initialize + ping), got %d", len(responses))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
