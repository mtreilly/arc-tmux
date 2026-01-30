// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

func newWaitCmd() *cobra.Command {
	var paneArg string
	var idle, timeout float64
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait until pane becomes idle",
		Long:  "Poll a pane until it stops printing output.",
		Example: `  # Wait up to 2 minutes for a compile step
  arc-tmux wait --pane=fe:2.0 --idle=2 --timeout=120`,
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

			if timeout <= 0 {
				timeout = 60
			}

			waitErr := tmux.WaitIdle(target, time.Duration(idle*float64(time.Second)), time.Duration(timeout*float64(time.Second)))
			result := waitResult{PaneID: target}
			if waitErr != nil {
				result.WaitError = waitErr.Error()
				if isTimeout(waitErr) {
					result.TimedOut = true
				}
			} else {
				result.Idle = true
			}

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return err
				}
				return waitErr
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				if err := enc.Encode(result); err != nil {
					return err
				}
				return waitErr
			case outputOpts.Is(output.OutputQuiet):
				if result.Idle {
					fmt.Fprintln(out, "idle")
					return waitErr
				}
				if result.TimedOut {
					fmt.Fprintln(out, "timeout")
					return waitErr
				}
				return waitErr
			}
			if result.Idle {
				fmt.Fprintf(out, "Pane %s is idle.\n", target)
			} else if result.TimedOut {
				fmt.Fprintf(out, "Pane %s did not become idle in time.\n", target)
			}
			return waitErr
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().Float64Var(&idle, "idle", 2.0, "Seconds of inactivity to consider idle")
	cmd.Flags().Float64Var(&timeout, "timeout", 60.0, "Maximum seconds to wait")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

type waitResult struct {
	PaneID    string `json:"pane_id" yaml:"pane_id"`
	Idle      bool   `json:"idle" yaml:"idle"`
	TimedOut  bool   `json:"timed_out" yaml:"timed_out"`
	WaitError string `json:"wait_error,omitempty" yaml:"wait_error,omitempty"`
}
