package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	agentsFile := filepath.Join(dir, agentsMDFile)

	if err := os.WriteFile(agentsFile, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(origDir) })
	os.Chdir(dir)

	err := runInitCmd(nil, nil)
	if err == nil {
		t.Fatal("expected error when AGENTS.md exists")
	}

	if got := err.Error(); got != "AGENTS.md already exists; remove it first to regenerate" {
		t.Fatalf("unexpected error: %s", got)
	}
}
