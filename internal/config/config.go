// Package config handles Fev configuration — YAML files, defaults, and project config loading.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ModelConfig holds LLM connection settings.
type ModelConfig struct {
	BaseURL       string  `yaml:"base_url"`
	ModelName     string  `yaml:"model_name"`
	Temperature   float64 `yaml:"temperature"`
	ContextLength int     `yaml:"context_length"`
}

// AgentConfig holds agent behavior settings.
type AgentConfig struct {
	MaxTurns         int    `yaml:"max_turns"`
	ConfirmDangerous bool   `yaml:"confirm_dangerous"`
	AutoCompact      bool   `yaml:"auto_compact"`
	SystemPrompt     string `yaml:"system_prompt"`
	WorkspaceRoot    string `yaml:"workspace_root"`
}

// Config is the top-level configuration for Fev.
type Config struct {
	Model ModelConfig `yaml:"model"`
	Agent AgentConfig `yaml:"agent"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Model: ModelConfig{
			BaseURL:       "http://localhost:11434/v1",
			ModelName:     "qwen2.5-coder:14b",
			Temperature:   0.7,
			ContextLength: 0, // 0 = auto-detect
		},
		Agent: AgentConfig{
			MaxTurns:         25,
			ConfirmDangerous: true,
			AutoCompact:      true,
		},
	}
}

// Load reads config from a YAML file. Returns defaults if file doesn't exist.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes config to a YAML file.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600) // restrictive — config may contain credentials
}

// ConfigDir returns the Fev config directory (~/.fev/).
func ConfigDir() (string, error) {
	if dir := os.Getenv("FEV_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fev"), nil
}

// LoadProjectConfig reads FEV.md and/or CLAUDE.md from a project directory.
// FEV.md content comes first; CLAUDE.md is appended if present.
// Returns the merged content as a string for injection into the system prompt.
func LoadProjectConfig(projectDir string) (string, error) {
	var parts []string

	fevPath := filepath.Join(projectDir, "FEV.md")
	if data, err := os.ReadFile(fevPath); err == nil {
		parts = append(parts, string(data))
	}

	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	if data, err := os.ReadFile(claudePath); err == nil {
		parts = append(parts, string(data))
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}
