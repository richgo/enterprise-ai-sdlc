# TaskRegistry Component Specification

## Overview

TaskRegistry manages a collection of tasks and provides query/mutation operations. It validates dependencies and provides the DAG for orchestration.

## Interface

```go
type Registry interface {
    // CRUD
    Add(task *Task) error
    Get(id string) (*Task, error)
    Update(task *Task) error
    Delete(id string) error
    
    // Queries
    List() []*Task
    ListByStatus(status Status) []*Task
    ListByRepo(repo string) []*Task
    GetReady() []*Task  // Tasks with all deps complete, status pending
    
    // Dependencies
    GetDeps(id string) ([]*Task, error)
    GetDependents(id string) ([]*Task, error)
    ValidateDeps(task *Task) error
    
    // Persistence
    Save(path string) error
    Load(path string) error
}
```

## Acceptance Criteria

### Add Task
- [ ] Can add a task to the registry
- [ ] Returns error if task with same ID already exists
- [ ] Returns error if task fails validation
- [ ] Returns error if deps reference non-existent tasks

### Get Task
- [ ] Returns task by ID
- [ ] Returns error if task not found

### Update Task
- [ ] Can update existing task
- [ ] Returns error if task not found
- [ ] Returns error if update fails validation
- [ ] Returns error if new deps reference non-existent tasks

### Delete Task
- [ ] Can delete task by ID
- [ ] Returns error if task not found
- [ ] Returns error if other tasks depend on this task

### List Operations
- [ ] List() returns all tasks
- [ ] ListByStatus() filters by status
- [ ] ListByRepo() filters by repository
- [ ] GetReady() returns tasks that can be started (pending + all deps complete)

### Dependency Operations
- [ ] GetDeps() returns tasks this task depends on
- [ ] GetDependents() returns tasks that depend on this task
- [ ] Detects circular dependencies on Add/Update

### Persistence
- [ ] Save() writes tasks to JSON file
- [ ] Load() reads tasks from JSON file
- [ ] Load() validates all tasks and deps after loading
