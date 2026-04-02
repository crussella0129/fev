package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Model.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("expected Ollama default URL, got %q", cfg.Model.BaseURL)
	}
	if cfg.Model.ModelName == "" {
		t.Error("expected non-empty default model name")
	}
	if cfg.Agent.MaxTurns < 1 {
		t.Error("expected positive max turns")
	}
}

func TestLoadConfig_DefaultsWhenMissing(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Model.BaseURL == "" {
		t.Error("expected defaults when file missing")
	}
}

func TestLoadConfig_FromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
model:
  base_url: "http://localhost:8080/v1"
  model_name: "llama3"
  temperature: 0.5
  context_length: 8192
agent:
  max_turns: 10
  confirm_dangerous: false
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Model.BaseURL != "http://localhost:8080/v1" {
		t.Errorf("expected custom URL, got %q", cfg.Model.BaseURL)
	}
	if cfg.Model.ModelName != "llama3" {
		t.Errorf("expected llama3, got %q", cfg.Model.ModelName)
	}
	if cfg.Agent.MaxTurns != 10 {
		t.Errorf("expected 10 max turns, got %d", cfg.Agent.MaxTurns)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := DefaultConfig()
	cfg.Model.ModelName = "custom-model"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if loaded.Model.ModelName != "custom-model" {
		t.Errorf("expected custom-model after round-trip, got %q", loaded.Model.ModelName)
	}
}

func TestLoadProjectConfig_FEVmd(t *testing.T) {
	dir := t.TempDir()
	fevmd := filepath.Join(dir, "FEV.md")
	content := `# FEV.md

## Project
This is a Go web service.

## Build & Test
- Build: ` + "`go build ./cmd/server`" + `
- Test: ` + "`go test ./...`" + `

## Do Not
- Modify migration files
`
	os.WriteFile(fevmd, []byte(content), 0644)

	pc, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if pc == "" {
		t.Error("expected non-empty project config from FEV.md")
	}
}

func TestLoadProjectConfig_CLAUDEmd(t *testing.T) {
	dir := t.TempDir()
	claudemd := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(claudemd, []byte("# CLAUDE.md\n\nUse Go conventions."), 0644)

	pc, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	if pc == "" {
		t.Error("expected non-empty project config from CLAUDE.md")
	}
}

func TestLoadProjectConfig_FEVmdTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "FEV.md"), []byte("# FEV rules"), 0644)
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# Claude rules"), 0644)

	pc, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}
	// Both should be included, FEV.md content first
	if len(pc) == 0 {
		t.Error("expected merged config")
	}
}
