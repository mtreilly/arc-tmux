// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

func newInterruptCmd() *cobra.Command {
	var paneArg string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:     "interrupt",
		Short:   "Send Ctrl+C to a pane",
		Long:    "Gracefully stop the foreground program in a pane by sending Ctrl+C.",
		Example: `  arc-tmux interrupt --pane=fe:api.0`,
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
			if err := tmux.Interrupt(target); err != nil {
				return err
			}
			result := actionResult{PaneID: target, Action: "interrupt"}
			return writeActionResult(cmd, outputOpts, result, "Sent Ctrl+C")
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

func newEscapeCmd() *cobra.Command {
	var paneArg string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:     "escape",
		Short:   "Send Escape key to a pane",
		Long:    "Inject a literal Escape keystroke.",
		Example: `  arc-tmux escape --pane=fe:2.0`,
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
			if err := tmux.Escape(target); err != nil {
				return err
			}
			result := actionResult{PaneID: target, Action: "escape"}
			return writeActionResult(cmd, outputOpts, result, "Sent Escape")
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

type actionResult struct {
	PaneID string `json:"pane_id" yaml:"pane_id"`
	Action string `json:"action" yaml:"action"`
}

func writeActionResult(cmd *cobra.Command, outputOpts output.OutputOptions, result actionResult, message string) error {
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
	_, _ = fmt.Fprintln(out, message)
	return nil
}
