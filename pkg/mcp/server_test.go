package mcp

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/richgo/flo/pkg/tools"
)

func TestMCPInitialize(t *testing.T) {
	// Create mock tools
	toolReg := tools.NewRegistry()
	toolReg.Register(tools.New("test_tool", "A test tool", nil, nil))

	server := NewServer(toolReg)

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]any{
			"protocolVersion": "2024-11-05",
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	resp, err := server.HandleRequest(req)
	if err != nil {
		t.Fatalf("initialize failed: %v", err)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocolVersion '2024-11-05', got '%v'", result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("expected serverInfo to be a map")
	}
	if serverInfo["name"] != "eas-mcp-server" {
		t.Errorf("expected server name 'eas-mcp-server', got '%v'", serverInfo["name"])
	}
}

func TestMCPToolsList(t *testing.T) {
	toolReg := tools.NewRegistry()
	toolReg.Register(tools.New("tool_a", "Tool A", map[string]any{"type": "object"}, nil))
	toolReg.Register(tools.New("tool_b", "Tool B", nil, nil))

	server := NewServer(toolReg)

	req := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	resp, err := server.HandleRequest(req)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	toolsList, ok := result["tools"].([]map[string]any)
	if !ok {
		t.Fatal("expected tools to be a list")
	}

	if len(toolsList) != 2 {
		t.Errorf("expected 2 tools, got %d", len(toolsList))
	}
}

func TestMCPToolsCall(t *testing.T) {
	toolReg := tools.NewRegistry()
	toolReg.Register(tools.New("echo", "Echo tool", nil, func(args tools.Args) (string, error) {
		msg, _ := args["message"].(string)
		return "Echo: " + msg, nil
	}))

	server := NewServer(toolReg)

	req := Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "echo",
			"arguments": map[string]any{
				"message": "hello",
			},
		},
	}

	resp, err := server.HandleRequest(req)
	if err != nil {
		t.Fatalf("tools/call failed: %v", err)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result to be a map")
	}

	content, ok := result["content"].([]map[string]any)
	if !ok {
		t.Fatal("expected content to be a list")
	}

	if len(content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(content))
	}

	if content[0]["text"] != "Echo: hello" {
		t.Errorf("expected 'Echo: hello', got '%v'", content[0]["text"])
	}
}

func TestMCPToolsCallNotFound(t *testing.T) {
	server := NewServer(tools.NewRegistry())

	req := Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params: map[string]any{
			"name":      "nonexistent",
			"arguments": map[string]any{},
		},
	}

	resp, err := server.HandleRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error response for nonexistent tool")
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	server := NewServer(tools.NewRegistry())

	req := Request{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "unknown/method",
	}

	resp, err := server.HandleRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestMCPServeStdio(t *testing.T) {
	toolReg := tools.NewRegistry()
	toolReg.Register(tools.New("test", "Test", nil, func(args tools.Args) (string, error) {
		return "ok", nil
	}))

	server := NewServer(toolReg)

	// Create request
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	reqData, _ := json.Marshal(req)

	// Simulate stdio
	input := bytes.NewBuffer(append(reqData, '\n'))
	output := &bytes.Buffer{}

	// Process one request
	err := server.ProcessRequest(input, output)
	if err != nil {
		t.Fatalf("ProcessRequest failed: %v", err)
	}

	// Parse response
	var resp Response
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
}

func TestMCPNotification(t *testing.T) {
	server := NewServer(tools.NewRegistry())

	// Notifications have no ID
	req := Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	resp, err := server.HandleRequest(req)
	if err != nil {
		t.Fatalf("notification failed: %v", err)
	}

	// Notifications should not return a response
	if resp != nil {
		t.Error("notifications should not return a response")
	}
}

func TestMCPToolsListWithSchema(t *testing.T) {
	toolReg := tools.NewRegistry()
	toolReg.Register(tools.New("greet", "Greet someone", map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Name to greet",
			},
		},
		"required": []any{"name"},
	}, nil))

	server := NewServer(toolReg)

	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp, _ := server.HandleRequest(req)
	result := resp.Result.(map[string]any)
	toolsList := result["tools"].([]map[string]any)

	tool := toolsList[0]
	inputSchema, ok := tool["inputSchema"].(map[string]any)
	if !ok {
		t.Fatal("expected inputSchema")
	}

	props, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties in schema")
	}

	nameProp, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatal("expected name property")
	}

	if nameProp["type"] != "string" {
		t.Errorf("expected type 'string', got '%v'", nameProp["type"])
	}
}
