// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func newInterruptCmd() *cobra.Command {
	var paneArg string

	cmd := &cobra.Command{
		Use:   "interrupt",
		Short: "Send Ctrl+C to a pane",
		Long:  "Gracefully stop the foreground program in a pane by sending Ctrl+C.",
		Example: `  arc-tmux interrupt --pane=fe:api.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if paneArg == "" {
				return fmt.Errorf("--pane is required")
			}
			if err := tmux.ValidateTarget(paneArg); err != nil {
				return err
			}
			return tmux.Interrupt(paneArg)
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

func newEscapeCmd() *cobra.Command {
	var paneArg string

	cmd := &cobra.Command{
		Use:   "escape",
		Short: "Send Escape key to a pane",
		Long:  "Inject a literal Escape keystroke.",
		Example: `  arc-tmux escape --pane=fe:2.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if paneArg == "" {
				return fmt.Errorf("--pane is required")
			}
			if err := tmux.ValidateTarget(paneArg); err != nil {
				return err
			}
			return tmux.Escape(paneArg)
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}
