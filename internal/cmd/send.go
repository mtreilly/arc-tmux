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

func newSendCmd() *cobra.Command {
	var paneArg string
	var enter bool
	var delayEnter float64

	cmd := &cobra.Command{
		Use:   "send [text]",
		Short: "Send text to a tmux pane",
		Long:  "Send literal text to a pane. By default we press Enter after the text.",
		Example: `  # Basic send (auto-enter)
  arc-tmux send "npm test" --pane=fe:2.0

  # Send without pressing Enter
  arc-tmux send "export SECRET=" --pane=fe:2.0 --enter=false`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.Join(args, " ")

			target, err := resolvePaneTarget(paneArg)
			if err != nil {
				return err
			}
			if err := tmux.ValidateTarget(target); err != nil {
				return err
			}

			d := time.Duration(delayEnter * float64(time.Second))
			if err := tmux.SendLiteral(target, text, enter, d); err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Text sent")
			return nil
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().BoolVar(&enter, "enter", true, "Press Enter after sending text")
	cmd.Flags().Float64Var(&delayEnter, "delay-enter", 1.0, "Delay in seconds before pressing Enter")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}
