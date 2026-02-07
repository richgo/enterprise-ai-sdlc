# MCP Server Documentation

## Overview

Flo exposes tools to AI agents via the Model Context Protocol (MCP). This enables Claude Code, Copilot, and other MCP-compatible agents to interact with the Flo workspace.

## Starting the MCP Server

```bash
flo mcp serve
```

The server runs on stdio, suitable for integration with AI agent frameworks.

## Available Tools

| Tool | Description |
|------|-------------|
| `flo_task_get` | Get task details by ID |
| `flo_task_list` | List all tasks with optional filters |
| `flo_run_tests` | Run tests for the current workspace |
| `flo_task_complete` | Mark a task as complete |
| `flo_spec_read` | Read the feature specification |

## Agent Discovery

Agents discover Flo via the MCP configuration file (`.flo/mcp.json`), which is auto-generated during workspace initialization:

```json
{
  "mcpServers": {
    "flo": {
      "command": "flo",
      "args": ["mcp", "serve"],
      "cwd": "/path/to/workspace"
    }
  }
}
```

## Concurrent Sessions

The MCP server is stateless - each agent session gets isolated access. Workspace state is protected by file locking (see pkg/task/registry.go).

## Protocol

Flo implements MCP 1.0. See https://modelcontextprotocol.io for the full specification.
