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

	cmd := &cobra.Command{
		Use:   "follow",
		Short: "Follow output from a tmux pane",
		Long:  "Continuously poll a tmux pane and stream any new output lines.",
		Example: `  arc-tmux follow --pane=fe:2.0
  arc-tmux follow --pane=fe:2.0 --output json
  arc-tmux follow --pane=fe:2.0 --from-start`,
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

			if interval <= 0 {
				interval = 1
			}
			if fromStart && !cmd.Flags().Changed("lines") {
				lines = 0
			}

			out := cmd.OutOrStdout()
			var jsonEnc *json.Encoder
			var yamlEnc *yaml.Encoder
			if outputOpts.Is(output.OutputJSON) {
				jsonEnc = json.NewEncoder(out)
			}
			if outputOpts.Is(output.OutputYAML) {
				yamlEnc = yaml.NewEncoder(out)
				defer yamlEnc.Close()
			}

			var prev []string
			maxPrev := 50
			ticker := time.NewTicker(time.Duration(interval * float64(time.Second)))
			defer ticker.Stop()

			for {
				capture, err := tmux.Capture(target, lines)
				if err != nil {
					return err
				}
				curr := splitLines(capture)
				emit := curr
				if len(prev) > 0 {
					emit = diffLines(prev, curr)
				}
				prev = tailLines(curr, maxPrev)

				if err := emitFollow(out, outputOpts, jsonEnc, yamlEnc, emit); err != nil {
					return err
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
	if len(prev) > len(curr) {
		return curr
	}
	for i := 0; i+len(prev) <= len(curr); i++ {
		if equalSlice(curr[i:i+len(prev)], prev) {
			return curr[i+len(prev):]
		}
	}
	return curr
}

func tailLines(lines []string, max int) []string {
	if max <= 0 || len(lines) <= max {
		return lines
	}
	return lines[len(lines)-max:]
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
