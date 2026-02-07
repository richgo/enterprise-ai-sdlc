package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()

	ws, err := Init(tmpDir, "my-feature", "claude")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check .flo directory created
	easDir := filepath.Join(tmpDir, ".flo")
	if _, err := os.Stat(easDir); os.IsNotExist(err) {
		t.Error(".flo directory not created")
	}

	// Check config.yaml created
	configPath := filepath.Join(easDir, "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.yaml not created")
	}

	// Check SPEC.md created
	specPath := filepath.Join(easDir, "SPEC.md")
	if _, err := os.Stat(specPath); os.IsNotExist(err) {
		t.Error("SPEC.md not created")
	}

	// Check tasks directory created
	tasksDir := filepath.Join(easDir, "tasks")
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		t.Error("tasks directory not created")
	}

	// Verify workspace properties
	if ws.Feature != "my-feature" {
		t.Errorf("expected feature 'my-feature', got '%s'", ws.Feature)
	}
	if ws.Backend != "claude" {
		t.Errorf("expected backend 'claude', got '%s'", ws.Backend)
	}
}

func TestInitAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize once
	Init(tmpDir, "first", "claude")

	// Try to initialize again
	_, err := Init(tmpDir, "second", "claude")
	if err == nil {
		t.Error("expected error for already initialized workspace")
	}
}

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize first
	Init(tmpDir, "test-feature", "copilot")

	// Load it
	ws, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if ws.Feature != "test-feature" {
		t.Errorf("expected feature 'test-feature', got '%s'", ws.Feature)
	}
	if ws.Backend != "copilot" {
		t.Errorf("expected backend 'copilot', got '%s'", ws.Backend)
	}
}

func TestLoadNotInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Load(tmpDir)
	if err == nil {
		t.Error("expected error for non-initialized workspace")
	}
}

func TestWorkspaceTaskOperations(t *testing.T) {
	tmpDir := t.TempDir()
	ws, _ := Init(tmpDir, "test", "claude")

	// Create a task
	task, err := ws.CreateTask("Implement OAuth", "android", nil, 0)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.Title != "Implement OAuth" {
		t.Errorf("expected title 'Implement OAuth', got '%s'", task.Title)
	}
	if task.Repo != "android" {
		t.Errorf("expected repo 'android', got '%s'", task.Repo)
	}

	// List tasks
	tasks := ws.ListTasks("", "")
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Get task
	got, err := ws.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if got.ID != task.ID {
		t.Errorf("ID mismatch")
	}
}

func TestWorkspaceTaskWithDeps(t *testing.T) {
	tmpDir := t.TempDir()
	ws, _ := Init(tmpDir, "test", "claude")

	// Create first task
	task1, _ := ws.CreateTask("First", "", nil, 0)

	// Create second task with dependency
	task2, err := ws.CreateTask("Second", "", []string{task1.ID}, 0)
	if err != nil {
		t.Fatalf("CreateTask with deps failed: %v", err)
	}

	if len(task2.Deps) != 1 || task2.Deps[0] != task1.ID {
		t.Error("deps not set correctly")
	}
}

func TestWorkspaceTaskInvalidDeps(t *testing.T) {
	tmpDir := t.TempDir()
	ws, _ := Init(tmpDir, "test", "claude")

	// Try to create task with non-existent dep
	_, err := ws.CreateTask("Bad deps", "", []string{"nonexistent"}, 0)
	if err == nil {
		t.Error("expected error for invalid deps")
	}
}

func TestWorkspacePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and add tasks
	ws1, _ := Init(tmpDir, "test", "claude")
	ws1.CreateTask("Task 1", "", nil, 0)
	ws1.CreateTask("Task 2", "", nil, 0)
	ws1.Save()

	// Load and verify
	ws2, _ := Load(tmpDir)
	tasks := ws2.ListTasks("", "")
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks after reload, got %d", len(tasks))
	}
}

func TestWorkspaceGetReadyTasks(t *testing.T) {
	tmpDir := t.TempDir()
	ws, _ := Init(tmpDir, "test", "claude")

	// Create tasks with deps
	task1, _ := ws.CreateTask("No deps", "", nil, 0)
	ws.CreateTask("Has deps", "", []string{task1.ID}, 0)

	ready := ws.GetReadyTasks()
	if len(ready) != 1 {
		t.Errorf("expected 1 ready task, got %d", len(ready))
	}
	if ready[0].ID != task1.ID {
		t.Errorf("wrong task ready: %s", ready[0].ID)
	}
}

func TestWorkspaceStatus(t *testing.T) {
	tmpDir := t.TempDir()
	ws, _ := Init(tmpDir, "test", "claude")

	ws.CreateTask("Task 1", "", nil, 0)
	ws.CreateTask("Task 2", "", nil, 0)

	status := ws.Status()

	if status.Feature != "test" {
		t.Errorf("expected feature 'test', got '%s'", status.Feature)
	}
	if status.TotalTasks != 2 {
		t.Errorf("expected 2 total tasks, got %d", status.TotalTasks)
	}
	if status.PendingTasks != 2 {
		t.Errorf("expected 2 pending tasks, got %d", status.PendingTasks)
	}
	if status.ReadyTasks != 2 {
		t.Errorf("expected 2 ready tasks, got %d", status.ReadyTasks)
	}
}

func TestWorkspaceTaskMDGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := Init(tmpDir, "test-feature", "claude")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create a task with type
	task, err := ws.CreateTaskWithType("Build API", "build", "", nil, 1)
	if err != nil {
		t.Fatalf("CreateTaskWithType failed: %v", err)
	}

	// Check task.md file was created
	taskPath := filepath.Join(tmpDir, ".flo", "tasks", "TASK-"+task.ID+".md")
	if _, err := os.Stat(taskPath); os.IsNotExist(err) {
		t.Fatalf("task.md file not created at %s", taskPath)
	}

	// Read and verify content
	content, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatalf("failed to read task file: %v", err)
	}

	contentStr := string(content)
	
	// Check for YAML frontmatter
	if !contains(contentStr, "---") {
		t.Error("task.md missing frontmatter markers")
	}
	if !contains(contentStr, "id: "+task.ID) {
		t.Error("task.md missing task ID")
	}
	if !contains(contentStr, "status: pending") {
		t.Error("task.md missing status")
	}
	if !contains(contentStr, "type: build") {
		t.Error("task.md missing type")
	}
	if !contains(contentStr, "model: claude/sonnet") {
		t.Error("task.md missing model from task type")
	}
	if !contains(contentStr, "# Build API") {
		t.Error("task.md missing title")
	}
}

func TestWorkspaceCreateTaskWithType(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := Init(tmpDir, "test-feature", "claude")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create task with design type
	task, err := ws.CreateTaskWithType("System architecture", "architecture", "", nil, 0)
	if err != nil {
		t.Fatalf("CreateTaskWithType failed: %v", err)
	}

	if task.Type != "architecture" {
		t.Errorf("expected type 'design', got %q", task.Type)
	}
	
	// Should get model from task type config
	if task.Model != "claude/opus" {
		t.Errorf("expected model 'claude/opus', got %q", task.Model)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
