package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

const (
	protocolVersion = "2025-03-26"
	serverName      = "human-compiler"
	serverVersion   = "0.4.0"
)

// Server is an MCP server that exposes the Human compiler as tools.
type Server struct {
	transport    *Transport
	spec         string
	examples     map[string]string
	lastBuildDir string
	mu           sync.Mutex
	logger       *log.Logger
}

// NewServer creates a new MCP server.
func NewServer(transport *Transport, spec string, examples map[string]string) *Server {
	return &Server{
		transport: transport,
		spec:      spec,
		examples:  examples,
		logger:    log.New(os.Stderr, "[human-mcp] ", log.LstdFlags),
	}
}

// Run starts the main dispatch loop. It reads JSON-RPC requests from the
// transport and dispatches them to the appropriate handlers.
func (s *Server) Run() error {
	defer s.cleanup()

	for {
		req, err := s.transport.ReadRequest()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			s.logger.Printf("read error: %v", err)
			return err
		}

		resp := s.dispatch(req)
		if resp != nil {
			if err := s.transport.WriteResponse(resp); err != nil {
				s.logger.Printf("write error: %v", err)
				return err
			}
		}
	}
}

// dispatch routes a request to the appropriate handler.
func (s *Server) dispatch(req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		// Client acknowledgment — no response needed
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "ping":
		return s.handlePing(req)
	default:
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: ErrCodeMethodNot, Message: fmt.Sprintf("unknown method: %s", req.Method)},
		}
	}
}

// handleInitialize responds to the MCP initialize handshake.
func (s *Server) handleInitialize(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: protocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: ServerInfo{
				Name:    serverName,
				Version: serverVersion,
			},
		},
	}
}

// handleToolsList returns all available tools.
func (s *Server) handleToolsList(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ToolsListResult{Tools: AllTools()},
	}
}

// handleToolsCall dispatches to the appropriate tool handler with panic recovery.
func (s *Server) handleToolsCall(req *Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: ErrCodeInvalidReq, Message: "invalid tools/call params: " + err.Error()},
		}
	}

	// Dispatch with panic recovery
	result := s.callToolSafe(params.Name, params.Arguments)

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// callToolSafe calls a tool handler with panic recovery.
func (s *Server) callToolSafe(name string, args json.RawMessage) (result *CallToolResult) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("panic in tool %s: %v", name, r)
			result = toolError(fmt.Sprintf("Internal error in %s: %v", name, r))
		}
	}()

	switch name {
	case "human_build":
		return s.handleBuild(args)
	case "human_validate":
		return s.handleValidate(args)
	case "human_ir":
		return s.handleIR(args)
	case "human_examples":
		return s.handleExamples(args)
	case "human_spec":
		return s.handleSpec(args)
	case "human_read_file":
		return s.handleReadFile(args)
	default:
		return toolError(fmt.Sprintf("Unknown tool: %s", name))
	}
}

// handlePing responds to a ping request.
func (s *Server) handlePing(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]any{},
	}
}

// cleanup removes any temporary build directories.
func (s *Server) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastBuildDir != "" {
		os.RemoveAll(s.lastBuildDir)
		s.lastBuildDir = ""
	}
}
