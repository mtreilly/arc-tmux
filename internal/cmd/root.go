// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command for arc-tmux.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "arc-tmux",
		Short: "Native tmux control (panes, windows, sessions)",
		Long: `Native control surface for tmux sessions, windows, and panes.

Sessions are the top-level workspaces; each session owns windows, and every
window can contain multiple panes.

Common subcommands:
  list      List available tmux panes
  send      Send text to a pane
  capture   Capture pane output
  run       Send -> wait for idle -> capture
  wait      Block until a pane quiets down
  kill      Safely kill a pane
  attach    Attach to a session
  launch    Open a new pane/window
  windows   List windows for a session
  status    Show current tmux location`,
		Example: `  arc-tmux list
  arc-tmux send "npm test" --pane=fe:2.0
  arc-tmux run "make lint" --pane=fe:2.0 --timeout 90s
  arc-tmux wait --pane=fe:2.0 --idle 2s --timeout 60s`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(
		newListCmd(),
		newSendCmd(),
		newCaptureCmd(),
		newWaitCmd(),
		newRunCmd(),
		newInterruptCmd(),
		newEscapeCmd(),
		newKillCmd(),
		newAttachCmd(),
		newCleanupCmd(),
		newLaunchCmd(),
		newWindowsCmd(),
		newStatusCmd(),
	)

	return root
}
