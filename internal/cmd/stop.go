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

type stopResult struct {
	PaneID      string `json:"pane_id" yaml:"pane_id"`
	Interrupted bool   `json:"interrupted" yaml:"interrupted"`
	Killed      bool   `json:"killed" yaml:"killed"`
	TimedOut    bool   `json:"timed_out" yaml:"timed_out"`
	WaitError   string `json:"wait_error,omitempty" yaml:"wait_error,omitempty"`
}

func newStopCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var paneArg string
	var idle, timeout float64
	var killOnTimeout bool

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Interrupt a pane and optionally kill if it hangs",
		Long:  "Send Ctrl+C to a pane, wait for idle, and kill on timeout unless disabled.",
		Example: `  arc-tmux stop --pane=fe:2.0
  arc-tmux stop --pane=@current --timeout 20 --idle 3
  arc-tmux stop --pane=@current --kill=false`,
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

			if timeout <= 0 {
				timeout = 30
			}
			if idle <= 0 {
				idle = 2
			}

			result := stopResult{PaneID: target}
			if err := tmux.Interrupt(target); err != nil {
				return err
			}
			result.Interrupted = true

			waitErr := tmux.WaitIdle(target, time.Duration(idle*float64(time.Second)), time.Duration(timeout*float64(time.Second)))
			if waitErr != nil {
				result.WaitError = waitErr.Error()
				if isTimeout(waitErr) {
					result.TimedOut = true
					if killOnTimeout {
						if err := tmux.Kill(target); err != nil {
							return err
						}
						result.Killed = true
					}
				} else {
					return waitErr
				}
			}

			out := cmd.OutOrStdout()
			retErr := waitErr
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return err
				}
				return retErr
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				if err := enc.Encode(result); err != nil {
					return err
				}
				return retErr
			case outputOpts.Is(output.OutputQuiet):
				if result.Killed {
					fmt.Fprintln(out, "killed")
					return retErr
				}
				fmt.Fprintln(out, "interrupted")
				return retErr
			}

			if result.Killed {
				fmt.Fprintf(out, "Pane %s interrupted and killed after timeout.\n", target)
				return retErr
			}
			if result.TimedOut {
				fmt.Fprintf(out, "Pane %s interrupted but did not become idle within timeout.\n", target)
				return retErr
			}
			fmt.Fprintf(out, "Pane %s interrupted.\n", target)
			return retErr
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active, @name)")
	cmd.Flags().Float64Var(&idle, "idle", 2.0, "Seconds of inactivity to consider idle")
	cmd.Flags().Float64Var(&timeout, "timeout", 30.0, "Maximum seconds to wait before kill")
	cmd.Flags().BoolVar(&killOnTimeout, "kill", true, "Kill the pane if it fails to become idle")
	_ = cmd.MarkFlagRequired("pane")
	return cmd
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}
