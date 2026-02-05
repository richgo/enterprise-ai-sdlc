package task

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryAdd(t *testing.T) {
	reg := NewRegistry()

	task := New("ua-001", "Implement OAuth")
	err := reg.Add(task)
	if err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// Verify it's in the registry
	got, err := reg.Get("ua-001")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if got.ID != "ua-001" {
		t.Errorf("expected ID 'ua-001', got '%s'", got.ID)
	}
}

func TestRegistryAddDuplicate(t *testing.T) {
	reg := NewRegistry()

	task1 := New("ua-001", "First")
	reg.Add(task1)

	task2 := New("ua-001", "Duplicate")
	err := reg.Add(task2)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestRegistryAddInvalidTask(t *testing.T) {
	reg := NewRegistry()

	task := &Task{ID: "", Title: "No ID"}
	err := reg.Add(task)
	if err == nil {
		t.Error("expected error for invalid task")
	}
}

func TestRegistryAddWithInvalidDeps(t *testing.T) {
	reg := NewRegistry()

	task := New("ua-002", "Depends on non-existent")
	task.Deps = []string{"ua-001"} // Doesn't exist

	err := reg.Add(task)
	if err == nil {
		t.Error("expected error for invalid dependency")
	}
}

func TestRegistryAddWithValidDeps(t *testing.T) {
	reg := NewRegistry()

	// Add dependency first
	dep := New("ua-001", "Dependency")
	reg.Add(dep)

	// Now add task that depends on it
	task := New("ua-002", "Depends on ua-001")
	task.Deps = []string{"ua-001"}

	err := reg.Add(task)
	if err != nil {
		t.Fatalf("failed to add task with valid deps: %v", err)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestRegistryUpdate(t *testing.T) {
	reg := NewRegistry()

	task := New("ua-001", "Original")
	reg.Add(task)

	task.Title = "Updated"
	err := reg.Update(task)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	got, _ := reg.Get("ua-001")
	if got.Title != "Updated" {
		t.Errorf("expected title 'Updated', got '%s'", got.Title)
	}
}

func TestRegistryUpdateNotFound(t *testing.T) {
	reg := NewRegistry()

	task := New("ua-001", "Does not exist")
	err := reg.Update(task)
	if err == nil {
		t.Error("expected error for updating nonexistent task")
	}
}

func TestRegistryDelete(t *testing.T) {
	reg := NewRegistry()

	task := New("ua-001", "To delete")
	reg.Add(task)

	err := reg.Delete("ua-001")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = reg.Get("ua-001")
	if err == nil {
		t.Error("expected task to be deleted")
	}
}

func TestRegistryDeleteWithDependents(t *testing.T) {
	reg := NewRegistry()

	// Add tasks with dependency
	dep := New("ua-001", "Dependency")
	reg.Add(dep)

	task := New("ua-002", "Depends on ua-001")
	task.Deps = []string{"ua-001"}
	reg.Add(task)

	// Try to delete the dependency
	err := reg.Delete("ua-001")
	if err == nil {
		t.Error("expected error when deleting task with dependents")
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()

	reg.Add(New("ua-001", "First"))
	reg.Add(New("ua-002", "Second"))
	reg.Add(New("ua-003", "Third"))

	tasks := reg.List()
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestRegistryListByStatus(t *testing.T) {
	reg := NewRegistry()

	t1 := New("ua-001", "Pending")
	reg.Add(t1)

	t2 := New("ua-002", "In Progress")
	reg.Add(t2)
	t2.SetStatus(StatusInProgress)
	reg.Update(t2)

	t3 := New("ua-003", "Also Pending")
	reg.Add(t3)

	pending := reg.ListByStatus(StatusPending)
	if len(pending) != 2 {
		t.Errorf("expected 2 pending tasks, got %d", len(pending))
	}

	inProgress := reg.ListByStatus(StatusInProgress)
	if len(inProgress) != 1 {
		t.Errorf("expected 1 in_progress task, got %d", len(inProgress))
	}
}

func TestRegistryListByRepo(t *testing.T) {
	reg := NewRegistry()

	t1 := New("ua-001", "Android task")
	t1.Repo = "android"
	reg.Add(t1)

	t2 := New("ua-002", "iOS task")
	t2.Repo = "ios"
	reg.Add(t2)

	t3 := New("ua-003", "Another Android task")
	t3.Repo = "android"
	reg.Add(t3)

	android := reg.ListByRepo("android")
	if len(android) != 2 {
		t.Errorf("expected 2 android tasks, got %d", len(android))
	}
}

func TestRegistryGetReady(t *testing.T) {
	reg := NewRegistry()

	// Task with no deps - should be ready
	t1 := New("ua-001", "No deps")
	reg.Add(t1)

	// Task with incomplete dep - not ready
	t2 := New("ua-002", "Has dep")
	t2.Deps = []string{"ua-001"}
	reg.Add(t2)

	ready := reg.GetReady()
	if len(ready) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(ready))
	}
	if ready[0].ID != "ua-001" {
		t.Errorf("expected ua-001 to be ready, got %s", ready[0].ID)
	}

	// Complete the dependency
	t1.SetStatus(StatusInProgress)
	reg.Update(t1)
	t1.SetStatus(StatusComplete)
	reg.Update(t1)

	// Now t2 should be ready
	ready = reg.GetReady()
	if len(ready) != 1 {
		t.Errorf("expected 1 ready task after dep complete, got %d", len(ready))
	}
	if ready[0].ID != "ua-002" {
		t.Errorf("expected ua-002 to be ready, got %s", ready[0].ID)
	}
}

func TestRegistryGetDeps(t *testing.T) {
	reg := NewRegistry()

	t1 := New("ua-001", "Dep 1")
	t2 := New("ua-002", "Dep 2")
	reg.Add(t1)
	reg.Add(t2)

	t3 := New("ua-003", "Has deps")
	t3.Deps = []string{"ua-001", "ua-002"}
	reg.Add(t3)

	deps, err := reg.GetDeps("ua-003")
	if err != nil {
		t.Fatalf("failed to get deps: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}
}

func TestRegistryGetDependents(t *testing.T) {
	reg := NewRegistry()

	t1 := New("ua-001", "Base")
	reg.Add(t1)

	t2 := New("ua-002", "Depends on base")
	t2.Deps = []string{"ua-001"}
	reg.Add(t2)

	t3 := New("ua-003", "Also depends on base")
	t3.Deps = []string{"ua-001"}
	reg.Add(t3)

	dependents, err := reg.GetDependents("ua-001")
	if err != nil {
		t.Fatalf("failed to get dependents: %v", err)
	}
	if len(dependents) != 2 {
		t.Errorf("expected 2 dependents, got %d", len(dependents))
	}
}

func TestRegistryCircularDependency(t *testing.T) {
	reg := NewRegistry()

	// Create circular: A -> B -> C -> A
	tA := New("ua-A", "A")
	reg.Add(tA)

	tB := New("ua-B", "B")
	tB.Deps = []string{"ua-A"}
	reg.Add(tB)

	tC := New("ua-C", "C")
	tC.Deps = []string{"ua-B"}
	reg.Add(tC)

	// Try to make A depend on C (creates cycle)
	tA.Deps = []string{"ua-C"}
	err := reg.Update(tA)
	if err == nil {
		t.Error("expected error for circular dependency")
	}
}

func TestRegistrySaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "tasks.json")

	// Create and save registry
	reg := NewRegistry()
	reg.Add(New("ua-001", "First"))

	t2 := New("ua-002", "Second")
	t2.Deps = []string{"ua-001"}
	reg.Add(t2)

	err := reg.Save(filePath)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("save file not created")
	}

	// Load into new registry
	reg2 := NewRegistry()
	err = reg2.Load(filePath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Verify contents
	tasks := reg2.List()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}

	task2, _ := reg2.Get("ua-002")
	if len(task2.Deps) != 1 || task2.Deps[0] != "ua-001" {
		t.Error("deps not preserved after load")
	}
}
