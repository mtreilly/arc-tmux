// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

type inspectSnapshot struct {
	Pane        tmux.PaneDetails   `json:"pane" yaml:"pane"`
	ProcessTree []tmux.ProcessNode `json:"process_tree" yaml:"process_tree"`
}

func newInspectCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var paneArg string

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect a tmux pane",
		Long:  "Inspect a tmux pane and return metadata plus the process tree for its PID.",
		Example: `  arc-tmux inspect --pane=fe:2.0
  arc-tmux inspect --pane=fe:2.0 --output json`,
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

			pane, err := tmux.PaneDetailsForTarget(target)
			if err != nil {
				return err
			}

			var tree []tmux.ProcessNode
			if pane.PID > 0 {
				tree, _ = tmux.ProcessTree(pane.PID)
			}

			snap := inspectSnapshot{Pane: pane, ProcessTree: tree}
			out := cmd.OutOrStdout()

			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(snap)

			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(snap)

			case outputOpts.Is(output.OutputQuiet):
				_, _ = fmt.Fprintf(out, "%s:%d.%d\n", pane.Session, pane.WindowIndex, pane.PaneIndex)
				return nil
			}

			paneID := fmt.Sprintf("%s:%d.%d", pane.Session, pane.WindowIndex, pane.PaneIndex)
			_, _ = fmt.Fprintf(out, "Pane: %s (id=%s)\n", paneID, pane.PaneID)
			_, _ = fmt.Fprintf(out, "  active=%t  window=%s:%d (%s)  window_active=%t\n",
				pane.Active,
				pane.Session,
				pane.WindowIndex,
				pane.WindowName,
				pane.WindowActive,
			)
			_, _ = fmt.Fprintf(out, "  cmd=%s  title=%s  path=%s  pid=%d  activity=%s\n",
				pane.Command,
				pane.Title,
				pane.Path,
				pane.PID,
				formatRelative(pane.ActivityAt),
			)

			if len(tree) == 0 {
				_, _ = fmt.Fprintln(out, "Process tree: (not available)")
				return nil
			}

			_, _ = fmt.Fprintln(out, "Process tree:")
			for _, node := range tree {
				indent := strings.Repeat("  ", node.Depth)
				_, _ = fmt.Fprintf(out, "%s- %d  %s\n", indent, node.PID, node.Command)
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	_ = cmd.MarkFlagRequired("pane")
	return cmd
}
