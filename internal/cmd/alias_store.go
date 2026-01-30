// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type aliasEntry struct {
	Name   string `json:"name" yaml:"name"`
	Target string `json:"target" yaml:"target"`
}

func defaultAliasFile() string {
	if env := strings.TrimSpace(os.Getenv("ARC_TMUX_ALIASES")); env != "" {
		return env
	}
	if dir, err := os.UserConfigDir(); err == nil && strings.TrimSpace(dir) != "" {
		return filepath.Join(dir, "arc-tmux", "aliases.json")
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".arc-tmux-aliases.json")
	}
	return "aliases.json"
}

func normalizeAliasName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", errors.New("alias name is required")
	}
	trimmed = strings.TrimPrefix(trimmed, "@")
	trimmed = strings.ToLower(trimmed)
	if trimmed == "current" || trimmed == "active" {
		return "", fmt.Errorf("alias %q is reserved", trimmed)
	}
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("invalid alias name: %q", name)
	}
	return trimmed, nil
}

func loadAliases(path string) (map[string]string, error) {
	aliases := make(map[string]string)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return aliases, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return aliases, nil
	}
	if err := json.Unmarshal(data, &aliases); err != nil {
		return nil, err
	}
	return aliases, nil
}

func saveAliases(path string, aliases map[string]string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func aliasesToEntries(aliases map[string]string) []aliasEntry {
	entries := make([]aliasEntry, 0, len(aliases))
	for name, target := range aliases {
		entries = append(entries, aliasEntry{Name: name, Target: target})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
}
