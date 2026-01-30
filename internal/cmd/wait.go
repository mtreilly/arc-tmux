// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func newWaitCmd() *cobra.Command {
	var paneArg string
	var idle, timeout float64

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait until pane becomes idle",
		Long:  "Poll a pane until it stops printing output.",
		Example: `  # Wait up to 2 minutes for a compile step
  arc-tmux wait --pane=fe:2.0 --idle=2 --timeout=120`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolvePaneTarget(paneArg)
			if err != nil {
				return err
			}
			if err := tmux.ValidateTarget(target); err != nil {
				return err
			}

			if timeout <= 0 {
				timeout = 60
			}

			return tmux.WaitIdle(target, time.Duration(idle*float64(time.Second)), time.Duration(timeout*float64(time.Second)))
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().Float64Var(&idle, "idle", 2.0, "Seconds of inactivity to consider idle")
	cmd.Flags().Float64Var(&timeout, "timeout", 60.0, "Maximum seconds to wait")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}
