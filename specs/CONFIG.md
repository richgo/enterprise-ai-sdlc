# Config Component Specification

## Overview

Config manages the feature configuration stored in `.eas/config.yaml`. It defines the backend, repos, and TDD settings.

## Data Model

```go
type Config struct {
    Feature string           // Feature name
    Version int              // Config version
    Backend string           // "claude" or "copilot"
    Claude  *ClaudeConfig    // Claude-specific settings
    Copilot *CopilotConfig   // Copilot-specific settings
    TDD     TDDConfig        // TDD settings
    Repos   map[string]Repo  // Linked repositories
}

type ClaudeConfig struct {
    CLIPath   string   // Path to claude binary
    Model     string   // Model name
    ExtraArgs []string // Additional CLI args
}

type CopilotConfig struct {
    CLIPath  string          // Path to copilot binary
    Model    string          // Model name
    Provider *ProviderConfig // BYOK settings
}

type ProviderConfig struct {
    Type      string // "openai" | "azure" | "anthropic"
    BaseURL   string // API endpoint
    APIKeyEnv string // Env var for API key
}

type TDDConfig struct {
    Enforce           bool   // Enforce tests before complete
    TestCommand       string // Command to run tests
    CoverageThreshold int    // Minimum coverage %
}

type Repo struct {
    URL    string // Git URL
    Branch string // Branch name
    Path   string // Local path (when cloned)
}
```

## Acceptance Criteria

### Load Config
- [ ] Load from .eas/config.yaml
- [ ] Return error if file not found
- [ ] Return error if YAML invalid
- [ ] Set defaults for optional fields

### Save Config
- [ ] Save to .eas/config.yaml
- [ ] Create directory if not exists
- [ ] Preserve comments (if possible)

### Validation
- [ ] Feature name required
- [ ] Backend must be "claude" or "copilot"
- [ ] Warn if backend config missing for selected backend

### Default Values
- [ ] Version defaults to 1
- [ ] Backend defaults to "claude"
- [ ] TDD.Enforce defaults to true
- [ ] TDD.TestCommand defaults to "go test ./..."
