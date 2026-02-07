package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := New("my-feature")

	if cfg.Feature != "my-feature" {
		t.Errorf("expected feature 'my-feature', got '%s'", cfg.Feature)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Backend != "claude" {
		t.Errorf("expected backend 'claude', got '%s'", cfg.Backend)
	}
	if !cfg.TDD.Enforce {
		t.Error("expected TDD.Enforce to be true by default")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  New("my-feature"),
			wantErr: false,
		},
		{
			name:    "empty feature",
			config:  &Config{Feature: "", Backend: "claude"},
			wantErr: true,
			errMsg:  "feature name",
		},
		{
			name:    "invalid backend",
			config:  &Config{Feature: "test", Backend: "invalid"},
			wantErr: true,
			errMsg:  "backend",
		},
		{
			name:    "claude backend valid",
			config:  &Config{Feature: "test", Backend: "claude"},
			wantErr: false,
		},
		{
			name:    "copilot backend valid",
			config:  &Config{Feature: "test", Backend: "copilot"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".eas", "config.yaml")

	// Create and save config
	cfg := New("test-feature")
	cfg.Backend = "copilot"
	cfg.TDD.TestCommand = "npm test"
	cfg.Repos = map[string]Repo{
		"android": {
			URL:    "git@github.com:org/android.git",
			Branch: "feature/auth",
		},
	}

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	// Load into new config
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Verify contents
	if loaded.Feature != "test-feature" {
		t.Errorf("feature mismatch: %s", loaded.Feature)
	}
	if loaded.Backend != "copilot" {
		t.Errorf("backend mismatch: %s", loaded.Backend)
	}
	if loaded.TDD.TestCommand != "npm test" {
		t.Errorf("test command mismatch: %s", loaded.TDD.TestCommand)
	}
	if len(loaded.Repos) != 1 {
		t.Errorf("expected 1 repo, got %d", len(loaded.Repos))
	}
	if loaded.Repos["android"].URL != "git@github.com:org/android.git" {
		t.Error("repo URL mismatch")
	}
}

func TestConfigLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestConfigLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write minimal config with explicit TDD enforce (YAML can't distinguish false from missing)
	os.WriteFile(configPath, []byte("feature: minimal\ntdd:\n  enforce: true\n"), 0644)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Check defaults applied
	if cfg.Version != 1 {
		t.Errorf("expected default version 1, got %d", cfg.Version)
	}
	if cfg.Backend != "claude" {
		t.Errorf("expected default backend 'claude', got '%s'", cfg.Backend)
	}
	if !cfg.TDD.Enforce {
		t.Error("expected TDD.Enforce to be true")
	}
}

func TestConfigWithClaudeSettings(t *testing.T) {
	cfg := New("test")
	cfg.Claude = &ClaudeConfig{
		CLIPath:   "/usr/local/bin/claude",
		Model:     "claude-sonnet-4-5-20250514",
		ExtraArgs: []string{"--dangerously-skip-permissions"},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg.Save(configPath)
	loaded, _ := Load(configPath)

	if loaded.Claude == nil {
		t.Fatal("Claude config not preserved")
	}
	if loaded.Claude.CLIPath != "/usr/local/bin/claude" {
		t.Errorf("CLI path mismatch: %s", loaded.Claude.CLIPath)
	}
	if loaded.Claude.Model != "claude-sonnet-4-5-20250514" {
		t.Errorf("model mismatch: %s", loaded.Claude.Model)
	}
	if len(loaded.Claude.ExtraArgs) != 1 {
		t.Errorf("extra args not preserved")
	}
}

func TestConfigWithCopilotSettings(t *testing.T) {
	cfg := New("test")
	cfg.Backend = "copilot"
	cfg.Copilot = &CopilotConfig{
		CLIPath: "copilot",
		Model:   "gpt-4.1",
		Provider: &ProviderConfig{
			Type:      "azure",
			BaseURL:   "https://mycompany.openai.azure.com/openai/v1/",
			APIKeyEnv: "AZURE_OPENAI_KEY",
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg.Save(configPath)
	loaded, _ := Load(configPath)

	if loaded.Copilot == nil {
		t.Fatal("Copilot config not preserved")
	}
	if loaded.Copilot.Provider == nil {
		t.Fatal("Provider config not preserved")
	}
	if loaded.Copilot.Provider.Type != "azure" {
		t.Errorf("provider type mismatch: %s", loaded.Copilot.Provider.Type)
	}
	if loaded.Copilot.Provider.APIKeyEnv != "AZURE_OPENAI_KEY" {
		t.Errorf("API key env mismatch: %s", loaded.Copilot.Provider.APIKeyEnv)
	}
}

func TestConfigTaskTypes(t *testing.T) {
	cfg := New("test")

	// Check that default task types are initialized
	if cfg.TaskTypes == nil {
		t.Fatal("TaskTypes should be initialized by default")
	}

	// Check for expected task types
	expectedTypes := []string{"architecture", "build", "refactor", "test", "fix", "docs", "review"}
	for _, typeName := range expectedTypes {
		taskType, exists := cfg.TaskTypes[typeName]
		if !exists {
			t.Errorf("expected task type %q to exist", typeName)
			continue
		}
		if taskType.Model == "" {
			t.Errorf("task type %q should have a model", typeName)
		}
	}

	// Verify specific models
	if cfg.TaskTypes["architecture"].Model != "claude/opus" {
		t.Errorf("architecture type should use claude/opus, got %q", cfg.TaskTypes["architecture"].Model)
	}
	if cfg.TaskTypes["architecture"].Thinking != "extended" {
		t.Errorf("architecture type should have extended thinking, got %q", cfg.TaskTypes["architecture"].Thinking)
	}
	if cfg.TaskTypes["build"].Model != "claude/sonnet" {
		t.Errorf("build type should use claude/sonnet, got %q", cfg.TaskTypes["build"].Model)
	}
	if cfg.TaskTypes["docs"].Model != "claude/haiku" {
		t.Errorf("docs type should use claude/haiku, got %q", cfg.TaskTypes["docs"].Model)
	}
}

func TestConfigTaskTypesPersistence(t *testing.T) {
	cfg := New("test")
	cfg.TaskTypes["custom"] = TaskType{
		Model:    "claude/sonnet",
		Thinking: "normal",
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if loaded.TaskTypes == nil {
		t.Fatal("TaskTypes not loaded")
	}

	customType, exists := loaded.TaskTypes["custom"]
	if !exists {
		t.Fatal("custom task type not preserved")
	}
	if customType.Model != "claude/sonnet" {
		t.Errorf("custom type model mismatch: got %q", customType.Model)
	}
	if customType.Thinking != "normal" {
		t.Errorf("custom type thinking mismatch: got %q", customType.Thinking)
	}
}
