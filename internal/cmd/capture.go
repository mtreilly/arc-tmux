// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func newCaptureCmd() *cobra.Command {
	var paneArg string
	var lines int

	cmd := &cobra.Command{
		Use:   "capture",
		Short: "Capture output from a tmux pane",
		Long:  "Capture the visible scrollback from a pane (default last 200 lines).",
		Example: `  # Tail the last 50 lines
  arc-tmux capture --pane=fe:2.0 | tail -50

  # Save entire buffer
  arc-tmux capture --pane=fe:2.0 --lines=0 > pane.log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolvePaneTarget(paneArg)
			if err != nil {
				return err
			}
			if err := tmux.ValidateTarget(target); err != nil {
				return err
			}

			s, err := tmux.Capture(target, lines)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), s)
			return nil
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().IntVar(&lines, "lines", 200, "Limit capture to last N lines (0 for full)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}
