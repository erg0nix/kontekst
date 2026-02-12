package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultsCreatesAllAgents(t *testing.T) {
	baseDir := t.TempDir()

	if err := EnsureDefaults(baseDir); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	for _, name := range []string{"default", "coder", "fantasy"} {
		agentDir := filepath.Join(baseDir, "agents", name)

		configPath := filepath.Join(agentDir, "config.toml")
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("agent %q: config.toml not found: %v", name, err)
		}

		promptPath := filepath.Join(agentDir, "agent.md")
		if _, err := os.Stat(promptPath); err != nil {
			t.Errorf("agent %q: agent.md not found: %v", name, err)
		}

		cfg, err := LoadTOML(configPath)
		if err != nil {
			t.Errorf("agent %q: LoadTOML: %v", name, err)
			continue
		}
		if cfg.Name == "" {
			t.Errorf("agent %q: config has empty name", name)
		}

		prompt, err := LoadPrompt(promptPath)
		if err != nil {
			t.Errorf("agent %q: LoadPrompt: %v", name, err)
			continue
		}
		if prompt == "" {
			t.Errorf("agent %q: prompt is empty", name)
		}
	}
}

func TestEnsureDefaultsIdempotent(t *testing.T) {
	baseDir := t.TempDir()

	if err := EnsureDefaults(baseDir); err != nil {
		t.Fatalf("first call: %v", err)
	}

	if err := EnsureDefaults(baseDir); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestEnsureDefaultsSkipsExisting(t *testing.T) {
	baseDir := t.TempDir()

	agentDir := filepath.Join(baseDir, "agents", "coder")
	os.MkdirAll(agentDir, 0o755)
	customConfig := []byte("name = \"My Custom Coder\"\n")
	os.WriteFile(filepath.Join(agentDir, "config.toml"), customConfig, 0o644)

	if err := EnsureDefaults(baseDir); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(agentDir, "config.toml"))
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	if string(data) != string(customConfig) {
		t.Error("EnsureDefaults overwrote existing agent config")
	}
}
