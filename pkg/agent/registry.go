// Package agent provides backend abstractions for AI agent execution.
package agent

import (
	"fmt"
	"sync"
)

// BackendFactory is a function that creates a backend instance.
type BackendFactory func(config any) Backend

var (
	// registry stores registered backend factories
	registry = make(map[string]BackendFactory)
	// mu protects the registry
	mu sync.RWMutex
)

func init() {
	// Register built-in backends
	RegisterBackend("claude", func(config any) Backend {
		if cfg, ok := config.(*ClaudeConfig); ok {
			return NewClaudeBackend(*cfg)
		}
		return NewClaudeBackend(ClaudeConfig{})
	})

	RegisterBackend("copilot", func(config any) Backend {
		if cfg, ok := config.(*CopilotConfig); ok {
			return NewCopilotBackend(*cfg)
		}
		return NewCopilotBackend(CopilotConfig{})
	})

	RegisterBackend("codex", func(config any) Backend {
		if cfg, ok := config.(*CodexConfig); ok {
			return NewCodexBackend(*cfg)
		}
		return NewCodexBackend(CodexConfig{})
	})

	RegisterBackend("gemini", func(config any) Backend {
		if cfg, ok := config.(*GeminiConfig); ok {
			return NewGeminiBackend(*cfg)
		}
		return NewGeminiBackend(GeminiConfig{})
	})

	RegisterBackend("mock", func(config any) Backend {
		return NewMockBackend()
	})
}

// RegisterBackend registers a backend factory with the given name.
// If a backend with the same name already exists, it will be replaced.
func RegisterBackend(name string, factory BackendFactory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// GetBackend returns a backend instance by name.
// Returns nil if the backend is not registered.
func GetBackend(name string, config any) (Backend, error) {
	mu.RLock()
	factory, exists := registry[name]
	mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("backend not registered: %s", name)
	}

	backend := factory(config)
	if backend == nil {
		return nil, fmt.Errorf("backend factory returned nil: %s", name)
	}

	return backend, nil
}

// ListBackends returns a list of all registered backend names.
func ListBackends() []string {
	mu.RLock()
	defer mu.RUnlock()

	backends := make([]string, 0, len(registry))
	for name := range registry {
		backends = append(backends, name)
	}
	return backends
}

// IsRegistered returns true if a backend with the given name is registered.
func IsRegistered(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, exists := registry[name]
	return exists
}
