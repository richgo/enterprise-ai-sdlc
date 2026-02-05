package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/richgo/enterprise-ai-sdlc/pkg/task"
)

func setupTestRegistry() *task.Registry {
	reg := task.NewRegistry()

	t1 := task.New("ua-001", "Implement OAuth")
	t1.Repo = "android"
	reg.Add(t1)

	t2 := task.New("ua-002", "Add token storage")
	t2.Repo = "android"
	t2.Deps = []string{"ua-001"}
	reg.Add(t2)

	t3 := task.New("ua-003", "iOS OAuth")
	t3.Repo = "ios"
	reg.Add(t3)

	return reg
}

func TestEASTaskList(t *testing.T) {
	taskReg := setupTestRegistry()
	tools := NewEASTools(taskReg, nil)

	// List all
	result, err := tools.Get("eas_task_list")
	if err != nil {
		t.Fatalf("tool not found: %v", err)
	}

	output, err := result.Execute(Args{})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	var tasks []map[string]any
	if err := json.Unmarshal([]byte(output), &tasks); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestEASTaskListFilterByStatus(t *testing.T) {
	taskReg := setupTestRegistry()

	// Set one task to in_progress
	task1, _ := taskReg.Get("ua-001")
	task1.SetStatus(task.StatusInProgress)
	taskReg.Update(task1)

	tools := NewEASTools(taskReg, nil)
	tool, _ := tools.Get("eas_task_list")

	output, _ := tool.Execute(Args{"status": "pending"})

	var tasks []map[string]any
	json.Unmarshal([]byte(output), &tasks)

	if len(tasks) != 2 {
		t.Errorf("expected 2 pending tasks, got %d", len(tasks))
	}
}

func TestEASTaskListFilterByRepo(t *testing.T) {
	taskReg := setupTestRegistry()
	tools := NewEASTools(taskReg, nil)
	tool, _ := tools.Get("eas_task_list")

	output, _ := tool.Execute(Args{"repo": "android"})

	var tasks []map[string]any
	json.Unmarshal([]byte(output), &tasks)

	if len(tasks) != 2 {
		t.Errorf("expected 2 android tasks, got %d", len(tasks))
	}
}

func TestEASTaskGet(t *testing.T) {
	taskReg := setupTestRegistry()
	tools := NewEASTools(taskReg, nil)
	tool, _ := tools.Get("eas_task_get")

	output, err := tool.Execute(Args{"task_id": "ua-001"})
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	var taskData map[string]any
	json.Unmarshal([]byte(output), &taskData)

	if taskData["id"] != "ua-001" {
		t.Errorf("expected id 'ua-001', got '%v'", taskData["id"])
	}
	if taskData["title"] != "Implement OAuth" {
		t.Errorf("expected title 'Implement OAuth', got '%v'", taskData["title"])
	}
}

func TestEASTaskGetNotFound(t *testing.T) {
	taskReg := setupTestRegistry()
	tools := NewEASTools(taskReg, nil)
	tool, _ := tools.Get("eas_task_get")

	_, err := tool.Execute(Args{"task_id": "nonexistent"})
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestEASTaskClaim(t *testing.T) {
	taskReg := setupTestRegistry()
	tools := NewEASTools(taskReg, nil)
	tool, _ := tools.Get("eas_task_claim")

	output, err := tool.Execute(Args{"task_id": "ua-001"})
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}

	if !strings.Contains(output, "claimed") {
		t.Errorf("expected success message, got '%s'", output)
	}

	// Verify status changed
	claimed, _ := taskReg.Get("ua-001")
	if claimed.Status != task.StatusInProgress {
		t.Errorf("expected status 'in_progress', got '%s'", claimed.Status)
	}
}

func TestEASTaskClaimNotPending(t *testing.T) {
	taskReg := setupTestRegistry()

	// Set task to in_progress first
	task1, _ := taskReg.Get("ua-001")
	task1.SetStatus(task.StatusInProgress)
	taskReg.Update(task1)

	tools := NewEASTools(taskReg, nil)
	tool, _ := tools.Get("eas_task_claim")

	_, err := tool.Execute(Args{"task_id": "ua-001"})
	if err == nil {
		t.Error("expected error for non-pending task")
	}
}

func TestEASTaskClaimDepsIncomplete(t *testing.T) {
	taskReg := setupTestRegistry()
	tools := NewEASTools(taskReg, nil)
	tool, _ := tools.Get("eas_task_claim")

	// ua-002 depends on ua-001 which is not complete
	_, err := tool.Execute(Args{"task_id": "ua-002"})
	if err == nil {
		t.Error("expected error for incomplete dependencies")
	}
}

func TestEASTaskComplete(t *testing.T) {
	taskReg := setupTestRegistry()

	// Create a mock test runner that always passes
	testRunner := &MockTestRunner{pass: true, output: "All tests passed"}

	tools := NewEASTools(taskReg, testRunner)
	
	// First claim the task
	claimTool, _ := tools.Get("eas_task_claim")
	claimTool.Execute(Args{"task_id": "ua-001"})

	// Then complete it
	completeTool, _ := tools.Get("eas_task_complete")
	output, err := completeTool.Execute(Args{"task_id": "ua-001"})
	if err != nil {
		t.Fatalf("complete failed: %v", err)
	}

	if !strings.Contains(output, "complete") {
		t.Errorf("expected success message, got '%s'", output)
	}

	// Verify status changed
	completed, _ := taskReg.Get("ua-001")
	if completed.Status != task.StatusComplete {
		t.Errorf("expected status 'complete', got '%s'", completed.Status)
	}
}

func TestEASTaskCompleteTestsFail(t *testing.T) {
	taskReg := setupTestRegistry()

	// Create a mock test runner that fails
	testRunner := &MockTestRunner{pass: false, output: "FAIL: TestAuth"}

	tools := NewEASTools(taskReg, testRunner)
	
	// Claim first
	claimTool, _ := tools.Get("eas_task_claim")
	claimTool.Execute(Args{"task_id": "ua-001"})

	// Try to complete
	completeTool, _ := tools.Get("eas_task_complete")
	_, err := completeTool.Execute(Args{"task_id": "ua-001"})
	if err == nil {
		t.Error("expected error when tests fail")
	}

	// Verify status NOT changed
	task1, _ := taskReg.Get("ua-001")
	if task1.Status == task.StatusComplete {
		t.Error("task should not be complete when tests fail")
	}
}

func TestEASRunTests(t *testing.T) {
	taskReg := setupTestRegistry()
	testRunner := &MockTestRunner{pass: true, output: "PASS: 5 tests"}

	tools := NewEASTools(taskReg, testRunner)
	tool, _ := tools.Get("eas_run_tests")

	output, err := tool.Execute(Args{"task_id": "ua-001"})
	if err != nil {
		t.Fatalf("run_tests failed: %v", err)
	}

	if !strings.Contains(output, "PASS") {
		t.Errorf("expected test output, got '%s'", output)
	}
}

// MockTestRunner is a test double for the test runner
type MockTestRunner struct {
	pass   bool
	output string
}

func (m *MockTestRunner) Run(taskID string) (bool, string, error) {
	return m.pass, m.output, nil
}
