// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

func newKillCmd() *cobra.Command {
	var paneArg string
	var yes bool
	var dryRun bool
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "kill",
		Short: "Kill a tmux pane (safe by default)",
		Long:  "Kill a pane after confirming the target.",
		Example: `  # Preview which pane would be killed
  arc-tmux kill --pane=fe:2.0 --dry-run

  # Kill without prompting (useful in scripts)
  arc-tmux kill --pane=fe:2.0 --yes`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			target, err := resolvePaneTarget(paneArg)
			if err != nil {
				return err
			}
			if err := validatePaneTarget(target); err != nil {
				return err
			}

			if dryRun {
				return writeKillResult(cmd, outputOpts, killResult{PaneID: target, DryRun: true}, "[dry-run] Would kill tmux pane")
			}

			if !yes {
				confirmed, err := confirmPrompt(cmd, fmt.Sprintf("Kill tmux pane %s? [y/N]: ", target))
				if err != nil {
					return err
				}
				if !confirmed {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted. No panes were killed.")
					return nil
				}
			}

			if err := tmux.Kill(target); err != nil {
				return err
			}
			return writeKillResult(cmd, outputOpts, killResult{PaneID: target, Killed: true}, "Killed tmux pane")
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
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
		if _, err := fmt.Fprint(cmd.OutOrStdout(), prompt); err != nil {
			return false, err
		}
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
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Please answer 'y' or 'n'.")
		}
	}
}

type killResult struct {
	PaneID string `json:"pane_id" yaml:"pane_id"`
	DryRun bool   `json:"dry_run" yaml:"dry_run"`
	Killed bool   `json:"killed" yaml:"killed"`
}

func writeKillResult(cmd *cobra.Command, outputOpts output.OutputOptions, result killResult, message string) error {
	out := cmd.OutOrStdout()
	switch {
	case outputOpts.Is(output.OutputJSON):
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case outputOpts.Is(output.OutputYAML):
		enc := yaml.NewEncoder(out)
		defer func() { _ = enc.Close() }()
		return enc.Encode(result)
	case outputOpts.Is(output.OutputQuiet):
		return nil
	}
	if result.DryRun {
		_, _ = fmt.Fprintf(out, "%s %s\n", message, result.PaneID)
		return nil
	}
	_, _ = fmt.Fprintf(out, "%s %s\n", message, result.PaneID)
	return nil
}
