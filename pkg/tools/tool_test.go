package tools

import (
	"strings"
	"testing"
)

func TestNewTool(t *testing.T) {
	tool := New("test_tool", "A test tool", nil, func(args Args) (string, error) {
		return "ok", nil
	})

	if tool.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got '%s'", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got '%s'", tool.Description)
	}
}

func TestToolExecute(t *testing.T) {
	tool := New("greet", "Greets a person", nil, func(args Args) (string, error) {
		name, _ := args["name"].(string)
		return "Hello, " + name, nil
	})

	result, err := tool.Execute(Args{"name": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello, World" {
		t.Errorf("expected 'Hello, World', got '%s'", result)
	}
}

func TestToolExecuteWithSchema(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
		"required": []any{"name"},
	}

	tool := New("greet", "Greets a person", schema, func(args Args) (string, error) {
		name, _ := args["name"].(string)
		return "Hello, " + name, nil
	})

	// Valid args
	result, err := tool.Execute(Args{"name": "World"})
	if err != nil {
		t.Fatalf("unexpected error with valid args: %v", err)
	}
	if result != "Hello, World" {
		t.Errorf("expected 'Hello, World', got '%s'", result)
	}

	// Missing required arg
	_, err = tool.Execute(Args{})
	if err == nil {
		t.Error("expected error for missing required arg")
	}
}

func TestToolRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	tool := New("my_tool", "My tool", nil, func(args Args) (string, error) {
		return "result", nil
	})

	reg.Register(tool)

	got, err := reg.Get("my_tool")
	if err != nil {
		t.Fatalf("failed to get tool: %v", err)
	}
	if got.Name != "my_tool" {
		t.Errorf("expected name 'my_tool', got '%s'", got.Name)
	}
}

func TestToolRegistryGetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestToolRegistryList(t *testing.T) {
	reg := NewRegistry()

	reg.Register(New("tool1", "First", nil, nil))
	reg.Register(New("tool2", "Second", nil, nil))
	reg.Register(New("tool3", "Third", nil, nil))

	tools := reg.List()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestToolRegistryExecute(t *testing.T) {
	reg := NewRegistry()

	reg.Register(New("echo", "Echoes input", nil, func(args Args) (string, error) {
		msg, _ := args["message"].(string)
		return msg, nil
	}))

	result, err := reg.Execute("echo", Args{"message": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got '%s'", result)
	}
}

func TestToolRegistryExecuteNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Execute("nonexistent", Args{})
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestToolSchemaValidation(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{
				"type": "integer",
			},
		},
		"required": []any{"count"},
	}

	tool := New("counter", "Counts", schema, func(args Args) (string, error) {
		return "counted", nil
	})

	// Wrong type (string instead of integer) - should fail validation
	_, err := tool.Execute(Args{"count": "not a number"})
	if err == nil {
		t.Error("expected error for wrong type")
	}
}

func TestToolHandlerError(t *testing.T) {
	tool := New("failing", "Always fails", nil, func(args Args) (string, error) {
		return "", &ToolError{Message: "intentional failure"}
	})

	_, err := tool.Execute(Args{})
	if err == nil {
		t.Error("expected error from handler")
	}
	if !strings.Contains(err.Error(), "intentional failure") {
		t.Errorf("expected error message 'intentional failure', got '%s'", err.Error())
	}
}
