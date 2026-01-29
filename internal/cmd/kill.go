// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func newKillCmd() *cobra.Command {
	var paneArg string
	var yes bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "kill",
		Short: "Kill a tmux pane (safe by default)",
		Long:  "Kill a pane after confirming the target.",
		Example: `  # Preview which pane would be killed
  arc-tmux kill --pane=fe:2.0 --dry-run

  # Kill without prompting (useful in scripts)
  arc-tmux kill --pane=fe:2.0 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if paneArg == "" {
				return fmt.Errorf("--pane is required")
			}
			if err := tmux.ValidateTarget(paneArg); err != nil {
				return err
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would kill tmux pane %s\n", paneArg)
				return nil
			}

			if !yes {
				confirmed, err := confirmPrompt(cmd, fmt.Sprintf("Kill tmux pane %s? [y/N]: ", paneArg))
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted. No panes were killed.")
					return nil
				}
			}

			return tmux.Kill(paneArg)
		},
	}

	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without killing")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

func confirmPrompt(cmd *cobra.Command, prompt string) (bool, error) {
	in := cmd.InOrStdin()
	if f, ok := in.(*os.File); ok {
		if !isatty.IsTerminal(f.Fd()) {
			return false, fmt.Errorf("confirmation required; run in interactive terminal or pass --yes")
		}
	}

	reader := bufio.NewReader(in)
	for {
		fmt.Fprint(cmd.OutOrStdout(), prompt)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		resp := strings.TrimSpace(strings.ToLower(response))
		switch resp {
		case "y", "yes":
			return true, nil
		case "", "n", "no":
			return false, nil
		default:
			fmt.Fprintln(cmd.OutOrStdout(), "Please answer 'y' or 'n'.")
		}
	}
}
