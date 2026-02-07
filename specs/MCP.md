# MCP Server Specification

## Overview

The MCP (Model Context Protocol) server exposes EAS tools to Claude Code. It implements the MCP JSON-RPC protocol over stdio.

## Protocol

MCP uses JSON-RPC 2.0 over stdio. Messages are newline-delimited JSON.

### Methods

- `initialize` - Handshake with capabilities
- `tools/list` - List available tools
- `tools/call` - Execute a tool

## Commands

```bash
`flo mcp serve          # Start MCP server on stdio
```

## Tool Mapping

EAS tools are exposed as MCP tools:
- `eas_task_list` → lists tasks
- `eas_task_get` → get task details  
- `eas_task_claim` → claim a task
- `eas_task_complete` → complete a task
- `eas_run_tests` → run tests
- `eas_spec_read` → read SPEC.md

## Acceptance Criteria

### initialize
- [ ] Returns server info and capabilities
- [ ] Supports tools capability

### tools/list
- [ ] Returns all EAS tools
- [ ] Includes name, description, inputSchema

### tools/call
- [ ] Executes requested tool
- [ ] Returns result or error
- [ ] Validates arguments against schema
