// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

type monitorSnapshot struct {
	PaneID       string    `json:"pane_id" yaml:"pane_id"`
	Session      string    `json:"session" yaml:"session"`
	WindowIndex  int       `json:"window_index" yaml:"window_index"`
	PaneIndex    int       `json:"pane_index" yaml:"pane_index"`
	Active       bool      `json:"active" yaml:"active"`
	Command      string    `json:"command" yaml:"command"`
	Title        string    `json:"title" yaml:"title"`
	Path         string    `json:"path" yaml:"path"`
	PID          int       `json:"pid" yaml:"pid"`
	ActivityAt   time.Time `json:"activity_at" yaml:"activity_at"`
	IdleSeconds  float64   `json:"idle_seconds" yaml:"idle_seconds"`
	Idle         bool      `json:"idle" yaml:"idle"`
	OutputHash   string    `json:"output_hash" yaml:"output_hash"`
	LinesChecked int       `json:"lines_checked" yaml:"lines_checked"`
}

func newMonitorCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var paneArg string
	var idle float64
	var lines int

	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Snapshot pane activity and output hash",
		Long:  "Return a single snapshot of pane activity, idle state, and output hash.",
		Example: `  arc-tmux monitor --pane=fe:2.0
  arc-tmux monitor --pane=@current --idle 5 --lines 200 --output json`,
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

			pane, err := tmux.PaneDetailsForTarget(target)
			if err != nil {
				return err
			}

			snapshot := monitorSnapshot{
				PaneID:       target,
				Session:      pane.Session,
				WindowIndex:  pane.WindowIndex,
				PaneIndex:    pane.PaneIndex,
				Active:       pane.Active,
				Command:      pane.Command,
				Title:        pane.Title,
				Path:         pane.Path,
				PID:          pane.PID,
				ActivityAt:   pane.ActivityAt,
				LinesChecked: lines,
			}

			if idle <= 0 {
				idle = 2
			}
			if !pane.ActivityAt.IsZero() {
				snapshot.IdleSeconds = time.Since(pane.ActivityAt).Seconds()
				snapshot.Idle = snapshot.IdleSeconds >= idle
			}

			capture, err := tmux.Capture(target, lines)
			if err != nil {
				return err
			}
			hash := sha1.Sum([]byte(capture))
			snapshot.OutputHash = hex.EncodeToString(hash[:])

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(snapshot)
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				return enc.Encode(snapshot)
			case outputOpts.Is(output.OutputQuiet):
				if snapshot.Idle {
					fmt.Fprintln(out, "idle")
					return nil
				}
				fmt.Fprintln(out, "busy")
				return nil
			}

			status := "busy"
			if snapshot.Idle {
				status = "idle"
			}
			fmt.Fprintf(out, "Pane %s is %s (idle %.1fs). hash=%s\n", target, status, snapshot.IdleSeconds, snapshot.OutputHash)
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active, @name)")
	cmd.Flags().Float64Var(&idle, "idle", 2.0, "Seconds of inactivity to consider idle")
	cmd.Flags().IntVar(&lines, "lines", 200, "Limit capture to last N lines for hashing (0 for full)")
	_ = cmd.MarkFlagRequired("pane")
	return cmd
}
