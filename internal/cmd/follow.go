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

type followEvent struct {
	Time string `json:"time" yaml:"time"`
	Line string `json:"line" yaml:"line"`
}

func newFollowCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var paneArg string
	var lines int
	var interval float64
	var fromStart bool
	var duration float64
	var once bool

	cmd := &cobra.Command{
		Use:   "follow",
		Short: "Follow output from a tmux pane",
		Long:  "Continuously poll a tmux pane and stream any new output lines.",
		Example: `  arc-tmux follow --pane=fe:2.0
  arc-tmux follow --pane=fe:2.0 --output json
  arc-tmux follow --pane=fe:2.0 --from-start
  arc-tmux follow --pane=fe:2.0 --duration 10
  arc-tmux follow --pane=fe:2.0 --once`,
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

			if interval <= 0 {
				interval = 1
			}
			if fromStart && !cmd.Flags().Changed("lines") {
				lines = 0
			}
			if duration < 0 {
				duration = 0
			}

			out := cmd.OutOrStdout()
			var jsonEnc *json.Encoder
			var yamlEnc *yaml.Encoder
			if outputOpts.Is(output.OutputJSON) {
				jsonEnc = json.NewEncoder(out)
			}
			if outputOpts.Is(output.OutputYAML) {
				yamlEnc = yaml.NewEncoder(out)
				defer func() { _ = yamlEnc.Close() }()
			}

			var prev []string
			prevCount := 0
			initialized := false
			var deadline time.Time
			if duration > 0 {
				deadline = time.Now().Add(time.Duration(duration * float64(time.Second)))
			}
			ticker := time.NewTicker(time.Duration(interval * float64(time.Second)))
			defer ticker.Stop()

			for {
				capture, err := tmux.Capture(target, lines)
				if err != nil {
					return err
				}
				curr := splitLines(capture)
				var emit []string
				if !initialized {
					if fromStart {
						emit = curr
					}
					initialized = true
					if lines == 0 {
						prevCount = len(curr)
					} else {
						prev = curr
					}
				} else if lines == 0 {
					emit = diffLinesByCount(curr, &prevCount)
				} else {
					emit = diffLines(prev, curr)
					prev = curr
				}

				if err := emitFollow(out, outputOpts, jsonEnc, yamlEnc, emit); err != nil {
					return err
				}

				if once {
					return nil
				}
				if !deadline.IsZero() && time.Now().After(deadline) {
					return nil
				}
				<-ticker.C
			}
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().IntVar(&lines, "lines", 200, "Limit capture to last N lines (0 for full)")
	cmd.Flags().Float64Var(&interval, "interval", 1.0, "Polling interval in seconds")
	cmd.Flags().BoolVar(&fromStart, "from-start", false, "Emit the full buffer before streaming new lines")
	cmd.Flags().Float64Var(&duration, "duration", 0, "Stop after N seconds (0 to run indefinitely)")
	cmd.Flags().Float64Var(&duration, "timeout", 0, "Alias for --duration")
	cmd.Flags().BoolVar(&once, "once", false, "Capture once and exit")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

func emitFollow(out interface{ Write([]byte) (int, error) }, outputOpts output.OutputOptions, jsonEnc *json.Encoder, yamlEnc *yaml.Encoder, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	for _, line := range lines {
		ts := time.Now().UTC().Format(time.RFC3339Nano)
		event := followEvent{Time: ts, Line: line}
		switch {
		case outputOpts.Is(output.OutputJSON):
			if err := jsonEnc.Encode(event); err != nil {
				return err
			}
		case outputOpts.Is(output.OutputYAML):
			if err := yamlEnc.Encode(event); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprintf(out, "%s\n", line); err != nil {
				return err
			}
		}
	}
	return nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func diffLines(prev []string, curr []string) []string {
	if len(prev) == 0 {
		return curr
	}
	max := len(prev)
	if len(curr) < max {
		max = len(curr)
	}
	for k := max; k > 0; k-- {
		if equalSlice(prev[len(prev)-k:], curr[:k]) {
			return curr[k:]
		}
	}
	return curr
}

func diffLinesByCount(curr []string, prevCount *int) []string {
	if prevCount == nil {
		return curr
	}
	if *prevCount <= 0 {
		*prevCount = len(curr)
		return curr
	}
	if len(curr) < *prevCount {
		*prevCount = len(curr)
		return curr
	}
	emit := curr[*prevCount:]
	*prevCount = len(curr)
	return emit
}

func equalSlice(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
