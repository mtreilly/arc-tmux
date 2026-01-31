// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"

	"github.com/yourorg/arc-tmux/pkg/tmux"
)

const agentSessionPrefix = "arc-"

// resolveAgentSessionName ensures new sessions use an agent-prefixed name.
// Returns the resolved session name and whether styling should be applied.
func resolveAgentSessionName(input string) (string, bool, error) {
	target := strings.TrimSpace(input)
	if target == "" {
		target = resolveManagedSession()
	}
	if strings.HasPrefix(target, agentSessionPrefix) {
		exists, err := tmux.HasSession(target)
		if err != nil {
			return "", false, err
		}
		return target, !exists, nil
	}

	exists, err := tmux.HasSession(target)
	if err != nil {
		return "", false, err
	}
	if exists {
		return target, false, nil
	}

	prefixed := agentSessionPrefix + target
	exists, err = tmux.HasSession(prefixed)
	if err != nil {
		return "", false, err
	}
	if exists {
		return prefixed, false, nil
	}
	return prefixed, true, nil
}

// resolveExistingSessionName tries the raw name, then the agent-prefixed name.
func resolveExistingSessionName(input string) (string, error) {
	target := strings.TrimSpace(input)
	if target == "" {
		target = resolveManagedSession()
	}
	if strings.HasPrefix(target, agentSessionPrefix) {
		return target, nil
	}

	exists, err := tmux.HasSession(target)
	if err != nil {
		return "", err
	}
	if exists {
		return target, nil
	}

	prefixed := agentSessionPrefix + target
	exists, err = tmux.HasSession(prefixed)
	if err != nil {
		return "", err
	}
	if exists {
		return prefixed, nil
	}
	return target, nil
}

func applyAgentStyleIfNeeded(session string, shouldStyle bool) error {
	if !shouldStyle {
		return nil
	}
	meta := tmux.DefaultAgentSessionMeta()
	return tmux.ApplyAgentSessionStyle(session, meta)
}
