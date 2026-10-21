package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStrictYAML(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(p, []byte("resolvers:\n  - 127.0.0.1\ntimeout: 1s\nformat: json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Resolvers) != 1 || cfg.Format != "json" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
func TestLoadRejectsUnknownFields(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	_ = os.WriteFile(p, []byte("resolvers: [1.1.1.1]\ntimeout: 1s\nformat: json\nunknown: true\n"), 0600)
	if _, err := Load(p); err == nil {
		t.Fatal("expected unknown field error")
	}
}
