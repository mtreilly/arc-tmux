// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func resolvePaneTarget(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("--pane is required")
	}
	if !strings.HasPrefix(trimmed, "@") {
		return trimmed, nil
	}
	switch trimmed {
	case "@current":
		id, err := tmux.CurrentPaneID()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(id) == "" {
			return "", errors.New("no current pane found")
		}
		return id, nil
	case "@active":
		panes, err := tmux.ListPanes()
		if err != nil {
			return "", err
		}
		var active []string
		for _, p := range panes {
			if p.Active {
				active = append(active, p.FormattedID())
			}
		}
		if len(active) == 0 {
			return "", errors.New("no active pane found")
		}
		sort.Strings(active)
		return active[0], nil
	default:
		alias := strings.TrimPrefix(trimmed, "@")
		name, err := normalizeAliasName(alias)
		if err != nil {
			return "", err
		}
		aliases, err := loadAliases(defaultAliasFile())
		if err != nil {
			return "", err
		}
		target, ok := aliases[name]
		if !ok {
			return "", fmt.Errorf("unknown pane selector: %s", trimmed)
		}
		return target, nil
	}
}

func resolveSessionTarget(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, "@") {
		return trimmed, nil
	}
	switch trimmed {
	case "@current":
		if !tmux.InTmux() {
			return "", errors.New("not inside tmux; @current requires a tmux client")
		}
		sess, _, _, _, err := tmux.CurrentLocation()
		if err != nil {
			return "", err
		}
		return sess, nil
	case "@managed":
		return resolveManagedSession(), nil
	default:
		return "", fmt.Errorf("unknown session selector: %s", trimmed)
	}
}
