# Tools Component Specification

## Overview

Tools are operations that agents can invoke. Tool definitions are backend-agnostic and can be used with both Claude Code (MCP) and Copilot SDK.

## Data Model

```go
type Tool struct {
    Name        string                 // Tool identifier
    Description string                 // Human-readable description for LLM
    Schema      map[string]any         // JSON Schema for parameters
    Handler     func(Args) (string, error)
}

type Args map[string]any
```

## EAS Tools

### Task Management
- `eas_task_list` - List tasks with optional filters
- `eas_task_get` - Get task details by ID
- `eas_task_claim` - Mark task as in_progress
- `eas_task_complete` - Mark task complete (runs tests first)

### TDD Enforcement
- `eas_run_tests` - Run tests for current task
- `eas_test_status` - Check if tests pass

### Context
- `eas_spec_read` - Read feature SPEC.md
- `eas_context_get` - Get full context for a task

## Acceptance Criteria

### Tool Definition
- [ ] Can create tool with name, description, schema, handler
- [ ] Schema is valid JSON Schema
- [ ] Handler receives parsed arguments
- [ ] Handler returns string result or error

### Tool Execution
- [ ] Execute() validates args against schema
- [ ] Execute() calls handler with parsed args
- [ ] Execute() returns result or error

### Tool Registry
- [ ] Can register multiple tools
- [ ] Can get tool by name
- [ ] Can list all tools
- [ ] Returns error for unknown tool
