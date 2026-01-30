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

func newCaptureCmd() *cobra.Command {
	var paneArg string
	var lines int
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "capture",
		Short: "Capture output from a tmux pane",
		Long:  "Capture the visible scrollback from a pane (default last 200 lines).",
		Example: `  # Tail the last 50 lines
  arc-tmux capture --pane=fe:2.0 | tail -50

  # Save entire buffer
  arc-tmux capture --pane=fe:2.0 --lines=0 > pane.log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
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

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				result := captureResult{PaneID: target, Output: s}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			case outputOpts.Is(output.OutputYAML):
				result := captureResult{PaneID: target, Output: s}
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				return enc.Encode(result)
			case outputOpts.Is(output.OutputQuiet):
				_, err := fmt.Fprint(out, s)
				return err
			}
			_, err = fmt.Fprint(out, s)
			return err
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().IntVar(&lines, "lines", 200, "Limit capture to last N lines (0 for full)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

type captureResult struct {
	PaneID string `json:"pane_id" yaml:"pane_id"`
	Output string `json:"output" yaml:"output"`
}
