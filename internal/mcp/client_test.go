package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

// mockMCPServer simulates an MCP server over pipes for testing.
// It runs in a goroutine reading requests and writing responses.
func mockMCPServer(serverIn io.Reader, serverOut io.Writer) {
	transport := NewTransport(serverIn, serverOut)

	for {
		req, err := transport.ReadRequest()
		if err != nil {
			return // EOF or error, stop
		}

		var resp *Response

		switch req.Method {
		case "initialize":
			resp = &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: InitializeResult{
					ProtocolVersion: protocolVersion,
					Capabilities: ServerCapabilities{
						Tools: &ToolsCapability{},
					},
					ServerInfo: ServerInfo{
						Name:    "mock-server",
						Version: "1.0.0",
					},
				},
			}

		case "notifications/initialized":
			// No response for notifications.
			continue

		case "tools/list":
			resp = &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: ToolsListResult{
					Tools: []Tool{
						{Name: "test_tool", Description: "A test tool"},
						{Name: "echo_tool", Description: "Echoes input"},
					},
				},
			}

		case "tools/call":
			var params CallToolParams
			json.Unmarshal(req.Params, &params)

			if params.Name == "echo_tool" {
				var args map[string]string
				json.Unmarshal(params.Arguments, &args)
				resp = &Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: CallToolResult{
						Content: []ContentItem{
							{Type: "text", Text: fmt.Sprintf("echo: %s", args["message"])},
						},
					},
				}
			} else if params.Name == "error_tool" {
				resp = &Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Result: CallToolResult{
						Content: []ContentItem{
							{Type: "text", Text: "tool error"},
						},
						IsError: true,
					},
				}
			} else {
				resp = &Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &RPCError{Code: ErrCodeInternal, Message: "unknown tool: " + params.Name},
				}
			}

		default:
			resp = &Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: ErrCodeMethodNot, Message: "unknown method"},
			}
		}

		if resp != nil {
			transport.WriteResponse(resp)
		}
	}
}

// setupTestClient creates a client connected to a mock server via pipes.
func setupTestClient(t *testing.T) *Client {
	t.Helper()

	// Client writes to serverIn, reads from serverOut.
	// Server reads from serverIn, writes to serverOut.
	clientToServer, serverIn := io.Pipe()
	serverToClient, clientOut := io.Pipe()

	go mockMCPServer(clientToServer, clientOut)

	client := newClientFromPipes("test-server", serverToClient, serverIn)

	t.Cleanup(func() {
		serverIn.Close()
		clientOut.Close()
		clientToServer.Close()
		serverToClient.Close()
	})

	return client
}

func TestClientConnect(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if client.Name() != "test-server" {
		t.Errorf("Name() = %q, want %q", client.Name(), "test-server")
	}
}

func TestClientToolsDiscovery(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tools := client.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	if tools[0].Name != "test_tool" {
		t.Errorf("tool[0].Name = %q, want test_tool", tools[0].Name)
	}
	if tools[1].Name != "echo_tool" {
		t.Errorf("tool[1].Name = %q, want echo_tool", tools[1].Name)
	}
}

func TestClientCallTool(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	result, err := client.CallTool(ctx, "echo_tool", map[string]any{
		"message": "hello world",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Text != "echo: hello world" {
		t.Errorf("result text = %q, want %q", result.Content[0].Text, "echo: hello world")
	}
	if result.IsError {
		t.Error("expected IsError = false")
	}
}

func TestClientCallToolRPCError(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	_, err := client.CallTool(ctx, "nonexistent_tool", map[string]any{})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("error = %q, expected to contain 'unknown tool'", err.Error())
	}
}

func TestClientCallToolAfterClose(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	client.Close()

	_, err := client.CallTool(ctx, "echo_tool", map[string]any{"message": "hi"})
	if err == nil {
		t.Fatal("expected error after close")
	}
	if !strings.Contains(err.Error(), "closed") {
		t.Errorf("error = %q, expected 'closed'", err.Error())
	}
}

func TestClientAlive(t *testing.T) {
	client := setupTestClient(t)

	// Not connected yet — transport is set but no handshake done.
	// Alive should be true since transport exists and not closed.
	if !client.Alive() {
		t.Error("expected Alive() = true before close")
	}

	client.Close()

	if client.Alive() {
		t.Error("expected Alive() = false after close")
	}
}

func TestClientDoubleClose(t *testing.T) {
	client := setupTestClient(t)

	ctx := context.Background()
	client.Connect(ctx)

	// Double close should not panic.
	client.Close()
	client.Close()
}

func TestClientContextCancellation(t *testing.T) {
	// Create a client with no server responding — should timeout.
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	// Create a reader that never returns data.
	client := newClientFromPipes("hang-server", r, w)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}
