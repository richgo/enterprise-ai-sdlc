// Package mcp implements an MCP (Model Context Protocol) server for EAS tools.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/richgo/flo/pkg/tools"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "eas-mcp-server"
	serverVersion   = "0.1.0"
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"` // Can be number, string, or null
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      any        `json:"id,omitempty"`
	Result  any        `json:"result,omitempty"`
	Error   *ErrorResp `json:"error,omitempty"`
}

// ErrorResp represents a JSON-RPC 2.0 error.
type ErrorResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Server is an MCP server that exposes tools.
type Server struct {
	tools *tools.Registry
}

// NewServer creates a new MCP server with the given tools.
func NewServer(toolReg *tools.Registry) *Server {
	return &Server{
		tools: toolReg,
	}
}

// HandleRequest processes a single MCP request and returns a response.
// Returns nil response for notifications (requests without ID).
func (s *Server) HandleRequest(req Request) (*Response, error) {
	// Notifications don't get responses
	if req.ID == nil {
		// Handle known notifications silently
		return nil, nil
	}

	resp := &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		resp.Result = s.handleInitialize(req.Params)
	case "tools/list":
		resp.Result = s.handleToolsList()
	case "tools/call":
		result, err := s.handleToolsCall(req.Params)
		if err != nil {
			resp.Error = &ErrorResp{
				Code:    -32000,
				Message: err.Error(),
			}
		} else {
			resp.Result = result
		}
	default:
		resp.Error = &ErrorResp{
			Code:    -32601,
			Message: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	return resp, nil
}

func (s *Server) handleInitialize(params map[string]any) map[string]any {
	return map[string]any{
		"protocolVersion": protocolVersion,
		"serverInfo": map[string]any{
			"name":    serverName,
			"version": serverVersion,
		},
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
	}
}

func (s *Server) handleToolsList() map[string]any {
	toolsList := s.tools.List()
	result := make([]map[string]any, 0, len(toolsList))

	for _, tool := range toolsList {
		toolInfo := map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
		}
		if tool.Schema != nil {
			toolInfo["inputSchema"] = tool.Schema
		} else {
			toolInfo["inputSchema"] = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		result = append(result, toolInfo)
	}

	return map[string]any{
		"tools": result,
	}
}

func (s *Server) handleToolsCall(params map[string]any) (map[string]any, error) {
	name, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tool name")
	}

	args, _ := params["arguments"].(map[string]any)
	if args == nil {
		args = make(map[string]any)
	}

	result, err := s.tools.Execute(name, tools.Args(args))
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": result,
			},
		},
	}, nil
}

// ProcessRequest reads a single request from input and writes response to output.
func (s *Server) ProcessRequest(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	line := scanner.Bytes()
	if len(line) == 0 {
		return nil // Empty line, skip
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		// Send parse error
		resp := Response{
			JSONRPC: "2.0",
			Error: &ErrorResp{
				Code:    -32700,
				Message: "Parse error: " + err.Error(),
			},
		}
		return s.writeResponse(output, &resp)
	}

	resp, err := s.HandleRequest(req)
	if err != nil {
		return err
	}

	if resp != nil {
		return s.writeResponse(output, resp)
	}

	return nil
}

func (s *Server) writeResponse(output io.Writer, resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = output.Write(append(data, '\n'))
	return err
}

// Serve runs the MCP server on stdio until EOF.
func (s *Server) Serve(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := Response{
				JSONRPC: "2.0",
				Error: &ErrorResp{
					Code:    -32700,
					Message: "Parse error: " + err.Error(),
				},
			}
			s.writeResponse(output, &resp)
			continue
		}

		resp, err := s.HandleRequest(req)
		if err != nil {
			continue
		}

		if resp != nil {
			s.writeResponse(output, resp)
		}
	}

	return scanner.Err()
}
