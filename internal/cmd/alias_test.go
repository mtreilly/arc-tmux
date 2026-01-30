package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeAliasName(t *testing.T) {
	name, err := normalizeAliasName(" Api-Service ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "api-service" {
		t.Fatalf("unexpected normalized name: %s", name)
	}
}

func TestAliasLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aliases.json")
	aliases := map[string]string{"api": "dev:1.0"}
	if err := saveAliases(path, aliases); err != nil {
		t.Fatalf("saveAliases error: %v", err)
	}
	loaded, err := loadAliases(path)
	if err != nil {
		t.Fatalf("loadAliases error: %v", err)
	}
	if loaded["api"] != "dev:1.0" {
		t.Fatalf("unexpected alias mapping: %#v", loaded)
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove error: %v", err)
	}
	loaded, err = loadAliases(path)
	if err != nil {
		t.Fatalf("load missing file error: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty aliases, got %#v", loaded)
	}
}
