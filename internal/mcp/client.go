package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Client connects to an external MCP server over stdio (JSON-RPC 2.0).
// It spawns the server as a child process and communicates via stdin/stdout.
type Client struct {
	name      string
	cmd       *exec.Cmd
	transport *Transport
	tools     []Tool
	nextID    atomic.Int64
	mu        sync.Mutex
	closed    bool
}

// NewClient creates an MCP client that will spawn the given command.
// The process is not started until Connect() is called.
func NewClient(name, command string, args []string, env map[string]string) *Client {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr // let server stderr pass through for debugging

	// Merge env vars into the process environment.
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	return &Client{
		name: name,
		cmd:  cmd,
	}
}

// newClientFromPipes creates a client using pre-connected reader/writer pipes.
// Used for testing without spawning a real process.
func newClientFromPipes(name string, r io.Reader, w io.Writer) *Client {
	return &Client{
		name:      name,
		transport: NewTransport(r, w),
	}
}

// Connect starts the server process and performs the MCP initialization handshake.
func (c *Client) Connect(ctx context.Context) error {
	// If transport is already set (test mode), skip process start.
	if c.transport == nil {
		stdin, err := c.cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("creating stdin pipe: %w", err)
		}
		stdout, err := c.cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("creating stdout pipe: %w", err)
		}

		if err := c.cmd.Start(); err != nil {
			return fmt.Errorf("starting MCP server %q: %w", c.name, err)
		}

		c.transport = NewTransport(stdout, stdin)
	}

	// Send initialize request.
	initResult, err := c.call(ctx, "initialize", InitializeParams{
		ProtocolVersion: protocolVersion,
		Capabilities:    map[string]any{},
		ClientInfo:      ClientInfo{Name: "human-repl", Version: serverVersion},
	})
	if err != nil {
		c.Close()
		return fmt.Errorf("initialize handshake failed: %w", err)
	}

	// Parse server info for logging.
	var result InitializeResult
	if data, ok := initResult.(json.RawMessage); ok {
		json.Unmarshal(data, &result)
	}

	// Send initialized notification (no response expected).
	c.sendNotification("notifications/initialized", nil)

	// Discover tools.
	toolsResult, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		// Non-fatal: server may not support tools/list.
		c.tools = nil
	} else {
		if data, ok := toolsResult.(json.RawMessage); ok {
			var tr ToolsListResult
			if json.Unmarshal(data, &tr) == nil {
				c.tools = tr.Tools
			}
		}
	}

	return nil
}

// Name returns the display name of this MCP server.
func (c *Client) Name() string { return c.name }

// Tools returns the tools discovered during initialization.
func (c *Client) Tools() []Tool { return c.tools }

// CallTool invokes a tool on the connected MCP server.
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]any) (*CallToolResult, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.mu.Unlock()

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("marshaling tool args: %w", err)
	}

	raw, err := c.call(ctx, "tools/call", CallToolParams{
		Name:      toolName,
		Arguments: argsJSON,
	})
	if err != nil {
		return nil, err
	}

	data, ok := raw.(json.RawMessage)
	if !ok {
		return nil, fmt.Errorf("unexpected result type from tools/call")
	}

	var result CallToolResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing tool result: %w", err)
	}

	return &result, nil
}

// Close shuts down the MCP server process.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	if c.cmd != nil && c.cmd.Process != nil {
		// Close stdin to signal the server to exit.
		if stdin, ok := c.cmd.Stdin.(io.Closer); ok {
			stdin.Close()
		}
		// Wait briefly, then kill if still running.
		done := make(chan error, 1)
		go func() { done <- c.cmd.Wait() }()
		select {
		case <-done:
		default:
			c.cmd.Process.Kill()
			<-done
		}
	}

	return nil
}

// Alive returns true if the client is connected and not closed.
func (c *Client) Alive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}
	if c.cmd != nil && c.cmd.ProcessState != nil {
		return false // process has exited
	}
	return c.transport != nil
}

// ── Internal JSON-RPC helpers ──

// call sends a JSON-RPC request and reads the response.
func (c *Client) call(ctx context.Context, method string, params any) (any, error) {
	id := c.nextID.Add(1)

	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("marshaling params: %w", err)
		}
		paramsJSON = data
	}

	idJSON, _ := json.Marshal(id)

	req := &Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  method,
		Params:  paramsJSON,
	}

	// Check context before writing.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Write the request as JSON + newline in a goroutine to avoid blocking
	// on a full pipe when context is cancelled.
	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	type writeResult struct{ err error }
	writeCh := make(chan writeResult, 1)
	go func() {
		c.transport.mu.Lock()
		_, err := fmt.Fprintf(c.transport.writer, "%s\n", reqData)
		c.transport.mu.Unlock()
		writeCh <- writeResult{err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case wr := <-writeCh:
		if wr.err != nil {
			return nil, fmt.Errorf("writing request: %w", wr.err)
		}
	}

	// Read response with context cancellation support.
	type readResult struct {
		resp *Response
		err  error
	}
	readCh := make(chan readResult, 1)
	go func() {
		resp, err := c.readResponse()
		readCh <- readResult{resp, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-readCh:
		if r.err != nil {
			return nil, r.err
		}
		if r.resp.Error != nil {
			return nil, fmt.Errorf("RPC error %d: %s", r.resp.Error.Code, r.resp.Error.Message)
		}
		// Return result as raw JSON for caller to unmarshal.
		data, err := json.Marshal(r.resp.Result)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(data), nil
	}
}

// sendNotification sends a JSON-RPC notification (no ID, no response expected).
func (c *Client) sendNotification(method string, params any) {
	var paramsJSON json.RawMessage
	if params != nil {
		data, _ := json.Marshal(params)
		paramsJSON = data
	}

	req := struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	data, _ := json.Marshal(req)
	c.transport.mu.Lock()
	fmt.Fprintf(c.transport.writer, "%s\n", data)
	c.transport.mu.Unlock()
}

// readResponse reads one JSON-RPC response from the transport.
func (c *Client) readResponse() (*Response, error) {
	if !c.transport.scanner.Scan() {
		if err := c.transport.scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}
		return nil, io.EOF
	}

	line := c.transport.scanner.Bytes()
	if len(line) == 0 {
		return c.readResponse() // skip blank lines
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &resp, nil
}
