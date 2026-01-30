// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func newRunCmd() *cobra.Command {
	var paneArg string
	var idle, timeout float64
	var lines int

	cmd := &cobra.Command{
		Use:   "run [command]",
		Short: "Send, wait idle, then capture output",
		Long:  "Fire-and-forget automation: send a command, wait until the pane quiets down, then print the captured output.",
		Example: `  # Run tests and capture the logs
  arc-tmux run "npm test" --pane=fe:2.0 --timeout=180

  # Long-running lint with custom idle threshold
  arc-tmux run "make lint" --pane=fe:2.0 --idle=5 --timeout=600`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolvePaneTarget(paneArg)
			if err != nil {
				return err
			}
			if err := tmux.ValidateTarget(target); err != nil {
				return err
			}

			text := strings.Join(args, " ")

			if err := tmux.SendLiteral(target, text, true, 0); err != nil {
				return err
			}

			if timeout <= 0 {
				timeout = 60
			}

			waitErr := tmux.WaitIdle(target, time.Duration(idle*float64(time.Second)), time.Duration(timeout*float64(time.Second)))

			s, err := tmux.Capture(target, lines)
			if err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), s)
			return waitErr
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().Float64Var(&idle, "idle", 2.0, "Seconds of inactivity to consider idle")
	cmd.Flags().Float64Var(&timeout, "timeout", 60.0, "Maximum seconds to wait")
	cmd.Flags().IntVar(&lines, "lines", 200, "Limit capture to last N lines (0 for full)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}
