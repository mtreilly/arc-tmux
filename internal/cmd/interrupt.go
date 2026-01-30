// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func newInterruptCmd() *cobra.Command {
	var paneArg string

	cmd := &cobra.Command{
		Use:     "interrupt",
		Short:   "Send Ctrl+C to a pane",
		Long:    "Gracefully stop the foreground program in a pane by sending Ctrl+C.",
		Example: `  arc-tmux interrupt --pane=fe:api.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolvePaneTarget(paneArg)
			if err != nil {
				return err
			}
			if err := tmux.ValidateTarget(target); err != nil {
				return err
			}
			return tmux.Interrupt(target)
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

func newEscapeCmd() *cobra.Command {
	var paneArg string

	cmd := &cobra.Command{
		Use:     "escape",
		Short:   "Send Escape key to a pane",
		Long:    "Inject a literal Escape keystroke.",
		Example: `  arc-tmux escape --pane=fe:2.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := resolvePaneTarget(paneArg)
			if err != nil {
				return err
			}
			if err := tmux.ValidateTarget(target); err != nil {
				return err
			}
			return tmux.Escape(target)
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}
