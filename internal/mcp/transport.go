package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Transport handles reading requests from and writing responses to stdio.
type Transport struct {
	scanner *bufio.Scanner
	writer  io.Writer
	mu      sync.Mutex
}

// NewTransport creates a new Transport reading from in and writing to out.
func NewTransport(in io.Reader, out io.Writer) *Transport {
	s := bufio.NewScanner(in)
	s.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB max line
	return &Transport{
		scanner: s,
		writer:  out,
	}
}

// ReadRequest reads one newline-delimited JSON-RPC request.
// Returns io.EOF when there are no more messages.
func (t *Transport) ReadRequest() (*Request, error) {
	if !t.scanner.Scan() {
		if err := t.scanner.Err(); err != nil {
			return nil, fmt.Errorf("reading request: %w", err)
		}
		return nil, io.EOF
	}

	line := t.scanner.Bytes()
	if len(line) == 0 {
		return t.ReadRequest() // skip blank lines
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, fmt.Errorf("parsing request: %w", err)
	}
	return &req, nil
}

// WriteResponse marshals and writes a JSON-RPC response followed by a newline.
func (t *Transport) WriteResponse(resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	_, err = fmt.Fprintf(t.writer, "%s\n", data)
	return err
}
