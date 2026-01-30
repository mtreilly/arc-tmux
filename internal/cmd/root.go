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
  sessions  List tmux sessions
  panes     List tmux panes with metadata
  list      List available tmux panes
  locate    Locate panes by metadata
  alias     Manage pane aliases
  recipes   Show common workflows
  send      Send text to a pane
  capture   Capture pane output
  follow    Stream pane output
  run       Send -> wait for idle -> capture
  monitor   Snapshot pane activity/output hash
  signal    Send a signal to a pane PID
  stop      Interrupt then kill on timeout
  wait      Block until a pane quiets down
  kill      Safely kill a pane
  attach    Attach to a session
  launch    Open a new pane/window
  windows   List windows for a session
  inspect   Inspect a pane and process tree
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
		newPanesCmd(),
		newSessionsCmd(),
		newLocateCmd(),
		newAliasCmd(),
		newRecipesCmd(),
		newSendCmd(),
		newCaptureCmd(),
		newWaitCmd(),
		newRunCmd(),
		newMonitorCmd(),
		newSignalCmd(),
		newStopCmd(),
		newInterruptCmd(),
		newEscapeCmd(),
		newKillCmd(),
		newInspectCmd(),
		newFollowCmd(),
		newAttachCmd(),
		newCleanupCmd(),
		newLaunchCmd(),
		newWindowsCmd(),
		newStatusCmd(),
	)

	return root
}
