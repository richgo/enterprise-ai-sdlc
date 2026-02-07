// Package config manages EAS feature configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the feature configuration.
type Config struct {
	Feature   string                `yaml:"feature"`
	Version   int                   `yaml:"version"`
	Backend   string                `yaml:"backend"`
	Claude    *ClaudeConfig         `yaml:"claude,omitempty"`
	Copilot   *CopilotConfig        `yaml:"copilot,omitempty"`
	TDD       TDDConfig             `yaml:"tdd"`
	Repos     map[string]Repo       `yaml:"repos,omitempty"`
	TaskTypes map[string]TaskType   `yaml:"taskTypes,omitempty"`
}

// ClaudeConfig holds Claude-specific settings.
type ClaudeConfig struct {
	CLIPath   string   `yaml:"cli_path,omitempty"`
	Model     string   `yaml:"model,omitempty"`
	ExtraArgs []string `yaml:"extra_args,omitempty"`
}

// CopilotConfig holds Copilot-specific settings.
type CopilotConfig struct {
	CLIPath  string          `yaml:"cli_path,omitempty"`
	Model    string          `yaml:"model,omitempty"`
	Provider *ProviderConfig `yaml:"provider,omitempty"`
}

// ProviderConfig holds BYOK provider settings.
type ProviderConfig struct {
	Type      string `yaml:"type"`
	BaseURL   string `yaml:"base_url"`
	APIKeyEnv string `yaml:"api_key_env,omitempty"`
}

// TDDConfig holds TDD enforcement settings.
type TDDConfig struct {
	Enforce           bool   `yaml:"enforce"`
	TestCommand       string `yaml:"test_command,omitempty"`
	CoverageThreshold int    `yaml:"coverage_threshold,omitempty"`
}

// Repo represents a linked repository.
type Repo struct {
	URL    string `yaml:"url"`
	Branch string `yaml:"branch,omitempty"`
	Path   string `yaml:"path,omitempty"`
}

// TaskType represents configuration for a task type.
type TaskType struct {
	Model    string `yaml:"model"`
	Thinking string `yaml:"thinking,omitempty"`
}

// New creates a new Config with default values.
func New(feature string) *Config {
	return &Config{
		Feature: feature,
		Version: 1,
		Backend: "claude",
		TDD: TDDConfig{
			Enforce:     true,
			TestCommand: "go test ./...",
		},
		TaskTypes: defaultTaskTypes(),
	}
}

// defaultTaskTypes returns the default task type configurations.
// Organized by phase: Discovery → Design → Build → Quality → Document
func defaultTaskTypes() map[string]TaskType {
	return map[string]TaskType{
		// Discovery - understand the problem space
		"research": {
			Model:    "claude/opus",
			Thinking: "extended",
		},
		"explore": {
			Model:    "claude/sonnet",
			Thinking: "normal",
		},
		
		// Design - plan the solution
		"architecture": {
			Model:    "claude/opus",
			Thinking: "extended",
		},
		"api-design": {
			Model:    "claude/sonnet",
			Thinking: "normal",
		},
		"visual-design": {
			Model:    "gemini/pro",
			Thinking: "normal",
		},
		"data-model": {
			Model: "claude/sonnet",
		},
		
		// Build - implement the solution
		"build": {
			Model: "claude/sonnet",
		},
		"refactor": {
			Model: "copilot/gpt-4",
		},
		"migrate": {
			Model:    "claude/sonnet",
			Thinking: "normal",
		},
		
		// Quality - verify and improve
		"test": {
			Model: "claude/sonnet",
		},
		"fix": {
			Model: "copilot/gpt-4",
		},
		"security": {
			Model:    "claude/opus",
			Thinking: "extended",
		},
		"performance": {
			Model: "claude/sonnet",
		},
		
		// Document - explain and review
		"docs": {
			Model: "claude/haiku",
		},
		"review": {
			Model: "claude/sonnet",
		},
	}
}

// Validate checks if the config is valid.
func (c *Config) Validate() error {
	if c.Feature == "" {
		return fmt.Errorf("feature name is required")
	}

	if c.Backend != "claude" && c.Backend != "copilot" {
		return fmt.Errorf("backend must be 'claude' or 'copilot', got '%s'", c.Backend)
	}

	return nil
}

// Load reads a config from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	return &cfg, nil
}

// Save writes the config to a YAML file.
func (c *Config) Save(path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// applyDefaults sets default values for optional fields.
func (c *Config) applyDefaults() {
	if c.Version == 0 {
		c.Version = 1
	}
	if c.Backend == "" {
		c.Backend = "claude"
	}
	if c.TDD.TestCommand == "" {
		c.TDD.TestCommand = "go test ./..."
	}
	if c.TaskTypes == nil {
		c.TaskTypes = defaultTaskTypes()
	}
	// TDD.Enforce defaults to true when not explicitly set
	// Note: YAML unmarshals missing bool as false, so we can't distinguish
	// between explicit false and missing. We default to true for new configs.
	// For loaded configs, we trust the file value.
}

// GetBackendConfig returns the backend-specific config.
func (c *Config) GetBackendConfig() any {
	switch c.Backend {
	case "claude":
		return c.Claude
	case "copilot":
		return c.Copilot
	default:
		return nil
	}
}

// DefaultConfigPath returns the default config path for a directory.
func DefaultConfigPath(dir string) string {
	return filepath.Join(dir, ".flo", "config.yaml")
}
