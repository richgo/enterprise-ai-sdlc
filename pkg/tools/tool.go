// Package tools provides backend-agnostic tool definitions for EAS.
package tools

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Args represents the arguments passed to a tool handler.
type Args map[string]any

// Handler is the function signature for tool handlers.
type Handler func(args Args) (string, error)

// Tool represents an operation that agents can invoke.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema,omitempty"`
	Handler     Handler        `json:"-"`
}

// ToolError represents an error from tool execution.
type ToolError struct {
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

// New creates a new Tool with the given parameters.
func New(name, description string, schema map[string]any, handler Handler) *Tool {
	return &Tool{
		Name:        name,
		Description: description,
		Schema:      schema,
		Handler:     handler,
	}
}

// Execute runs the tool with the given arguments.
// It validates arguments against the schema (if present) before calling the handler.
func (t *Tool) Execute(args Args) (string, error) {
	if t.Schema != nil {
		if err := t.validateArgs(args); err != nil {
			return "", fmt.Errorf("argument validation failed: %w", err)
		}
	}

	if t.Handler == nil {
		return "", fmt.Errorf("tool '%s' has no handler", t.Name)
	}

	return t.Handler(args)
}

// validateArgs validates arguments against the JSON schema.
func (t *Tool) validateArgs(args Args) error {
	schema := t.Schema
	
	// Check if it's an object schema
	schemaType, _ := schema["type"].(string)
	if schemaType != "object" {
		return nil // Only validate object schemas
	}

	// Check required fields
	required, ok := schema["required"].([]any)
	if ok {
		for _, reqField := range required {
			fieldName, _ := reqField.(string)
			if _, exists := args[fieldName]; !exists {
				return fmt.Errorf("missing required field: %s", fieldName)
			}
		}
	}

	// Check field types
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return nil
	}

	for fieldName, value := range args {
		propSchema, ok := properties[fieldName].(map[string]any)
		if !ok {
			continue // Unknown field, skip
		}

		expectedType, _ := propSchema["type"].(string)
		if err := validateType(fieldName, value, expectedType); err != nil {
			return err
		}
	}

	return nil
}

// validateType checks if a value matches the expected JSON Schema type.
func validateType(fieldName string, value any, expectedType string) error {
	if value == nil {
		return nil // null is valid for any type in JSON Schema by default
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string", fieldName)
		}
	case "integer":
		switch v := value.(type) {
		case int, int64, float64:
			// float64 is acceptable if it's a whole number
			if f, ok := v.(float64); ok && f != float64(int64(f)) {
				return fmt.Errorf("field '%s' must be an integer", fieldName)
			}
		default:
			return fmt.Errorf("field '%s' must be an integer", fieldName)
		}
	case "number":
		switch value.(type) {
		case int, int64, float64:
			// All numeric types are valid
		default:
			return fmt.Errorf("field '%s' must be a number", fieldName)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean", fieldName)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return fmt.Errorf("field '%s' must be an array", fieldName)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("field '%s' must be an object", fieldName)
		}
	}

	return nil
}

// ToJSON returns the tool definition as JSON (for MCP/API responses).
func (t *Tool) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// Registry manages a collection of tools.
type Registry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool *Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (*Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool, nil
}

// List returns all registered tools.
func (r *Registry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Execute runs a tool by name with the given arguments.
func (r *Registry) Execute(name string, args Args) (string, error) {
	tool, err := r.Get(name)
	if err != nil {
		return "", err
	}
	return tool.Execute(args)
}
