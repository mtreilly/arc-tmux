// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

func newSendCmd() *cobra.Command {
	var paneArg string
	var enter bool
	var delayEnter float64
	var keys []string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "send [text]",
		Short: "Send text to a tmux pane",
		Long:  "Send literal text or tmux key names to a pane. By default we press Enter after the text.",
		Example: `  # Basic send (auto-enter)
  arc-tmux send "npm test" --pane=fe:2.0

  # Send without pressing Enter
  arc-tmux send "export SECRET=" --pane=fe:2.0 --enter=false

  # Send raw tmux keys
  arc-tmux send --pane=fe:2.0 --key C-x --key C-c`,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 && len(keys) == 0 {
				return fmt.Errorf("requires text or at least one --key")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
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

			d := time.Duration(delayEnter * float64(time.Second))
			text := strings.Join(args, " ")
			if text != "" {
				if err := tmux.SendLiteral(target, text, enter, d); err != nil {
					return err
				}
			}
			if len(keys) > 0 {
				if err := tmux.SendKeys(target, keys); err != nil {
					return err
				}
			}

			result := sendResult{
				PaneID:    target,
				Text:      text,
				Keys:      keys,
				Enter:     enter,
				DelaySecs: delayEnter,
			}
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
			_, _ = fmt.Fprintln(out, "Text sent")
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().StringArrayVar(&keys, "key", nil, "Send tmux key names (repeatable, e.g., C-x, Up, Enter)")
	cmd.Flags().BoolVar(&enter, "enter", true, "Press Enter after sending text")
	cmd.Flags().Float64Var(&delayEnter, "delay-enter", 1.0, "Delay in seconds before pressing Enter")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

type sendResult struct {
	PaneID    string   `json:"pane_id" yaml:"pane_id"`
	Text      string   `json:"text" yaml:"text"`
	Keys      []string `json:"keys,omitempty" yaml:"keys,omitempty"`
	Enter     bool     `json:"enter" yaml:"enter"`
	DelaySecs float64  `json:"delay_secs" yaml:"delay_secs"`
}
