package agent

import (
	"context"
	"sync"

	"github.com/richgo/flo/pkg/task"
)

// MockBackend is a test backend that records calls and returns configured responses.
type MockBackend struct {
	mu       sync.Mutex
	calls    []Call
	response Result
	events   []Event
}

// NewMockBackend creates a new mock backend.
func NewMockBackend() *MockBackend {
	return &MockBackend{
		response: Result{Success: true},
	}
}

func (m *MockBackend) Name() string {
	return "mock"
}

func (m *MockBackend) Start(ctx context.Context) error {
	return nil
}

func (m *MockBackend) Stop() error {
	return nil
}

func (m *MockBackend) CreateSession(ctx context.Context, t *task.Task, worktree string) (Session, error) {
	return &MockSession{
		backend:  m,
		task:     t,
		worktree: worktree,
		events:   make(chan Event, 100),
	}, nil
}

// SetResponse configures the response to return.
func (m *MockBackend) SetResponse(r Result) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.response = r
}

// SetEvents configures the events to emit.
func (m *MockBackend) SetEvents(events []Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = events
}

// GetCalls returns recorded calls.
func (m *MockBackend) GetCalls() []Call {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Call{}, m.calls...)
}

func (m *MockBackend) recordCall(call Call) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, call)
}

func (m *MockBackend) getResponse() Result {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.response
}

func (m *MockBackend) getEvents() []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Event{}, m.events...)
}

// MockSession is a mock session for testing.
type MockSession struct {
	backend  *MockBackend
	task     *task.Task
	worktree string
	events   chan Event
}

func (s *MockSession) Run(ctx context.Context, prompt string) (*Result, error) {
	// Record the call
	s.backend.recordCall(Call{
		TaskID:   s.task.ID,
		Worktree: s.worktree,
		Prompt:   prompt,
	})

	// Emit events
	for _, event := range s.backend.getEvents() {
		s.events <- event
	}
	close(s.events)

	// Return configured response
	result := s.backend.getResponse()
	return &result, nil
}

func (s *MockSession) Events() <-chan Event {
	return s.events
}

func (s *MockSession) Destroy(ctx context.Context) error {
	return nil
}
