package agent

import (
	"testing"
)

func TestRegisterBackend(t *testing.T) {
	// Register a test backend
	testFactory := func(config any) Backend {
		return NewMockBackend()
	}
	
	RegisterBackend("test-backend", testFactory)
	
	if !IsRegistered("test-backend") {
		t.Error("backend should be registered")
	}
}

func TestGetBackend(t *testing.T) {
	tests := []struct {
		name        string
		backendName string
		wantErr     bool
	}{
		{
			name:        "get claude backend",
			backendName: "claude",
			wantErr:     false,
		},
		{
			name:        "get copilot backend",
			backendName: "copilot",
			wantErr:     false,
		},
		{
			name:        "get mock backend",
			backendName: "mock",
			wantErr:     false,
		},
		{
			name:        "get unknown backend",
			backendName: "unknown",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := GetBackend(tt.backendName, nil)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if backend != nil {
					t.Error("expected nil backend on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if backend == nil {
					t.Error("expected backend, got nil")
				}
				if backend.Name() != tt.backendName {
					t.Errorf("expected backend name %s, got %s", tt.backendName, backend.Name())
				}
			}
		})
	}
}

func TestListBackends(t *testing.T) {
	backends := ListBackends()
	
	if len(backends) == 0 {
		t.Error("expected at least one backend")
	}

	// Check for built-in backends
	hasClaudeOrCopilot := false
	for _, name := range backends {
		if name == "claude" || name == "copilot" || name == "mock" {
			hasClaudeOrCopilot = true
			break
		}
	}
	
	if !hasClaudeOrCopilot {
		t.Error("expected at least one built-in backend (claude, copilot, or mock)")
	}
}

func TestIsRegistered(t *testing.T) {
	tests := []struct {
		name         string
		backendName  string
		wantExists   bool
	}{
		{
			name:        "claude is registered",
			backendName: "claude",
			wantExists:  true,
		},
		{
			name:        "copilot is registered",
			backendName: "copilot",
			wantExists:  true,
		},
		{
			name:        "unknown not registered",
			backendName: "unknown",
			wantExists:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists := IsRegistered(tt.backendName)
			if exists != tt.wantExists {
				t.Errorf("IsRegistered(%s) = %v, want %v", tt.backendName, exists, tt.wantExists)
			}
		})
	}
}

func TestBackendFactoryReplacement(t *testing.T) {
	// Register initial backend
	callCount := 0
	factory1 := func(config any) Backend {
		callCount++
		return NewMockBackend()
	}
	
	RegisterBackend("replaceable", factory1)
	
	// Get backend once
	GetBackend("replaceable", nil)
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
	
	// Replace with new factory
	factory2 := func(config any) Backend {
		callCount += 10
		return NewMockBackend()
	}
	
	RegisterBackend("replaceable", factory2)
	
	// Get backend again - should use new factory
	GetBackend("replaceable", nil)
	if callCount != 11 {
		t.Errorf("expected 11 calls total, got %d", callCount)
	}
}
