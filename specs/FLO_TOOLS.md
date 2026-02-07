# EAS Tools Specification

## Overview

EAS tools are the concrete tool implementations that agents invoke. They operate on the TaskRegistry and enforce TDD.

## Tools

### eas_task_list
Lists tasks with optional filters.

**Parameters:**
- `status` (string, optional): Filter by status
- `repo` (string, optional): Filter by repository

**Returns:** JSON array of tasks

### eas_task_get
Gets detailed info about a task.

**Parameters:**
- `task_id` (string, required): Task ID

**Returns:** JSON object with task details

### eas_task_claim
Claims a task (sets status to in_progress).

**Parameters:**
- `task_id` (string, required): Task ID

**Returns:** Success message or error

### eas_task_complete
Marks task complete. Runs tests first.

**Parameters:**
- `task_id` (string, required): Task ID

**Returns:** Success message or error (if tests fail)

### eas_run_tests
Runs tests for a task.

**Parameters:**
- `task_id` (string, required): Task ID

**Returns:** Test output and pass/fail status

### eas_spec_read
Reads the feature specification.

**Parameters:** None

**Returns:** SPEC.md contents

## Acceptance Criteria

### eas_task_list
- [ ] Returns all tasks when no filters
- [ ] Filters by status correctly
- [ ] Filters by repo correctly
- [ ] Returns empty array if no matches

### eas_task_claim
- [ ] Changes status to in_progress
- [ ] Returns error if task not found
- [ ] Returns error if task not pending
- [ ] Returns error if deps not complete

### eas_task_complete
- [ ] Returns error if tests fail
- [ ] Changes status to complete if tests pass
- [ ] Updates timestamp

### eas_run_tests
- [ ] Executes configured test command
- [ ] Returns output and success status
- [ ] Handles missing test command gracefully
