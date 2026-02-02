// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

// AgentSessionMeta captures identifying metadata for agent-managed sessions.
type AgentSessionMeta struct {
	Owner     string
	Host      string
	CreatedAt string
}

// DefaultAgentSessionMeta builds metadata from the current environment.
func DefaultAgentSessionMeta() AgentSessionMeta {
	owner := strings.TrimSpace(os.Getenv("USER"))
	if u, err := user.Current(); err == nil && strings.TrimSpace(u.Username) != "" {
		owner = strings.TrimSpace(u.Username)
	}
	host, _ := os.Hostname()
	return AgentSessionMeta{
		Owner:     owner,
		Host:      host,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// ApplyAgentSessionStyle applies a distinctive style and metadata to a session.
func ApplyAgentSessionStyle(session string, meta AgentSessionMeta) error {
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	owner := strings.TrimSpace(meta.Owner)
	if owner == "" {
		owner = "agent"
	}
	statusLeft := fmt.Sprintf(" #[fg=colour16,bg=colour220,bold] ARC-TMUX #[default] %s ", owner)
	statusRight := " #[fg=colour245]agent#[default] "
	commands := [][]string{
		{"set-option", "-t", session, "@arc_tmux", "1"},
		{"set-option", "-t", session, "@arc_tmux_owner", meta.Owner},
		{"set-option", "-t", session, "@arc_tmux_host", meta.Host},
		{"set-option", "-t", session, "@arc_tmux_created_at", meta.CreatedAt},
		{"set-environment", "-t", session, "ARC_TMUX", "1"},
		{"set-environment", "-t", session, "ARC_TMUX_OWNER", meta.Owner},
		{"set-environment", "-t", session, "ARC_TMUX_HOST", meta.Host},
		{"set-option", "-t", session, "status-style", "bg=colour236,fg=colour15"},
		{"set-option", "-t", session, "status-left", statusLeft},
		{"set-option", "-t", session, "status-right", statusRight},
		{"set-option", "-t", session, "status-left-length", "40"},
		{"set-option", "-t", session, "status-right-length", "40"},
		{"set-option", "-t", session, "default-command", "sh"},
	}
	for _, args := range commands {
		if err := exec.Command("tmux", args...).Run(); err != nil {
			return fmt.Errorf("tmux %s: %w", args[0], err)
		}
	}

	windows, err := ListWindows(session)
	if err != nil {
		return err
	}
	for _, w := range windows {
		if err := ApplyAgentWindowStyle(session, w.WindowIndex); err != nil {
			return err
		}
	}
	return nil
}

// ApplyAgentWindowStyle applies pane border styles to a specific window.
func ApplyAgentWindowStyle(session string, windowIndex int) error {
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	target := fmt.Sprintf("%s:%d", session, windowIndex)
	commands := [][]string{
		{"set-window-option", "-t", target, "window-status-style", "fg=colour250,bg=colour236"},
		{"set-window-option", "-t", target, "window-status-format", " #I:#W "},
		{"set-window-option", "-t", target, "window-status-current-style", "fg=colour16,bg=colour214,bold"},
		{"set-window-option", "-t", target, "window-status-current-format", " #[fg=colour16,bg=colour214,bold]#I:#W#[default] "},
		{"set-window-option", "-t", target, "pane-border-style", "fg=colour240"},
		{"set-window-option", "-t", target, "pane-active-border-style", "fg=colour208,bold"},
	}
	for _, args := range commands {
		if err := exec.Command("tmux", args...).Run(); err != nil {
			return fmt.Errorf("tmux %s: %w", args[0], err)
		}
	}
	return nil
}
