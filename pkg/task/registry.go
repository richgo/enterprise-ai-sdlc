package task

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Registry manages a collection of tasks with dependency tracking.
type Registry struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewRegistry creates an empty task registry.
func NewRegistry() *Registry {
	return &Registry{
		tasks: make(map[string]*Task),
	}
}

// Add adds a task to the registry.
// Returns error if task ID exists, validation fails, or deps are invalid.
func (r *Registry) Add(task *Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID '%s' already exists", task.ID)
	}

	if err := r.validateDepsLocked(task); err != nil {
		return err
	}

	r.tasks[task.ID] = task
	return nil
}

// Get returns a task by ID.
func (r *Registry) Get(id string) (*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task '%s' not found", id)
	}
	return task, nil
}

// Update updates an existing task.
func (r *Registry) Update(task *Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return fmt.Errorf("task '%s' not found", task.ID)
	}

	if err := r.validateDepsLocked(task); err != nil {
		return err
	}

	// Check for circular dependencies
	if err := r.checkCircularLocked(task.ID, task.Deps, make(map[string]bool)); err != nil {
		return err
	}

	r.tasks[task.ID] = task
	return nil
}

// Delete removes a task by ID.
// Returns error if task has dependents.
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return fmt.Errorf("task '%s' not found", id)
	}

	// Check for dependents
	for _, task := range r.tasks {
		for _, dep := range task.Deps {
			if dep == id {
				return fmt.Errorf("cannot delete task '%s': task '%s' depends on it", id, task.ID)
			}
		}
	}

	delete(r.tasks, id)
	return nil
}

// List returns all tasks.
func (r *Registry) List() []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// ListByStatus returns tasks with the given status.
func (r *Registry) ListByStatus(status Status) []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*Task
	for _, task := range r.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// ListByRepo returns tasks for the given repository.
func (r *Registry) ListByRepo(repo string) []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*Task
	for _, task := range r.tasks {
		if task.Repo == repo {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetReady returns tasks that are ready to start.
// A task is ready if it's pending and all its dependencies are complete.
func (r *Registry) GetReady() []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ready []*Task
	for _, task := range r.tasks {
		if task.Status != StatusPending {
			continue
		}
		if r.allDepsCompleteLocked(task) {
			ready = append(ready, task)
		}
	}
	return ready
}

// GetDeps returns the tasks that the given task depends on.
func (r *Registry) GetDeps(id string) ([]*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task '%s' not found", id)
	}

	deps := make([]*Task, 0, len(task.Deps))
	for _, depID := range task.Deps {
		if dep, exists := r.tasks[depID]; exists {
			deps = append(deps, dep)
		}
	}
	return deps, nil
}

// GetDependents returns tasks that depend on the given task.
func (r *Registry) GetDependents(id string) ([]*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.tasks[id]; !exists {
		return nil, fmt.Errorf("task '%s' not found", id)
	}

	var dependents []*Task
	for _, task := range r.tasks {
		for _, dep := range task.Deps {
			if dep == id {
				dependents = append(dependents, task)
				break
			}
		}
	}
	return dependents, nil
}

// ValidateDeps checks if all dependencies exist.
func (r *Registry) ValidateDeps(task *Task) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.validateDepsLocked(task)
}

// validateDepsLocked checks deps without acquiring lock.
func (r *Registry) validateDepsLocked(task *Task) error {
	for _, depID := range task.Deps {
		if _, exists := r.tasks[depID]; !exists {
			return fmt.Errorf("dependency '%s' not found", depID)
		}
	}
	return nil
}

// allDepsCompleteLocked checks if all deps are complete without acquiring lock.
func (r *Registry) allDepsCompleteLocked(task *Task) bool {
	for _, depID := range task.Deps {
		dep, exists := r.tasks[depID]
		if !exists || dep.Status != StatusComplete {
			return false
		}
	}
	return true
}

// checkCircularLocked detects circular dependencies via DFS.
func (r *Registry) checkCircularLocked(startID string, deps []string, visited map[string]bool) error {
	for _, depID := range deps {
		if depID == startID {
			return fmt.Errorf("circular dependency detected: %s", startID)
		}
		if visited[depID] {
			continue
		}
		visited[depID] = true

		dep, exists := r.tasks[depID]
		if !exists {
			continue
		}
		if err := r.checkCircularLocked(startID, dep.Deps, visited); err != nil {
			return err
		}
	}
	return nil
}

// registryData is the JSON structure for persistence.
type registryData struct {
	Tasks []*Task `json:"tasks"`
}

// Save writes the registry to a JSON file.
func (r *Registry) Save(path string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data := registryData{
		Tasks: make([]*Task, 0, len(r.tasks)),
	}
	for _, task := range r.tasks {
		data.Tasks = append(data.Tasks, task)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}

	return nil
}

// Load reads the registry from a JSON file.
func (r *Registry) Load(path string) error {
	jsonData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	var data registryData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing and add all tasks
	r.tasks = make(map[string]*Task)
	
	// First pass: add all tasks without dep validation
	for _, task := range data.Tasks {
		if err := task.Validate(); err != nil {
			return fmt.Errorf("invalid task '%s': %w", task.ID, err)
		}
		r.tasks[task.ID] = task
	}

	// Second pass: validate all deps
	for _, task := range r.tasks {
		if err := r.validateDepsLocked(task); err != nil {
			return fmt.Errorf("task '%s': %w", task.ID, err)
		}
	}

	return nil
}
