package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveConfigUsesPrivateMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := DefaultConfig()
	cfg.LLM.APIKey = "should-not-appear-in-world-readable-file"

	if err := SaveConfig(cfg, path); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		t.Fatalf("config file must be 0600, got %#o", mode)
	}
}

func TestLoadConfigPrefersEnvAPIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := DefaultConfig()
	cfg.LLM.Provider = "openai"
	cfg.LLM.APIKey = ""
	if err := SaveConfig(cfg, path); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENAI_API_KEY", "from-env")
	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LLM.APIKey != "from-env" {
		t.Fatalf("expected env API key, got %q", loaded.LLM.APIKey)
	}
}
