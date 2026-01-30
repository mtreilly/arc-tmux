// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

type sessionInfo struct {
	Name       string    `json:"name" yaml:"name"`
	Windows    int       `json:"windows" yaml:"windows"`
	Attached   int       `json:"attached" yaml:"attached"`
	CreatedAt  time.Time `json:"created_at" yaml:"created_at"`
	ActivityAt time.Time `json:"activity_at" yaml:"activity_at"`
}

func newSessionsCmd() *cobra.Command {
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List tmux sessions",
		Long:  "List tmux sessions with window counts and activity timestamps.",
		Example: `  arc-tmux sessions
  arc-tmux sessions --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}

			sessions, err := tmux.ListSessions()
			if err != nil {
				if err == tmux.ErrNoTmuxServer {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No tmux server is running.")
					return nil
				}
				return err
			}

			items := make([]sessionInfo, 0, len(sessions))
			for _, s := range sessions {
				items = append(items, sessionInfo{
					Name:       s.Name,
					Windows:    s.Windows,
					Attached:   s.Attached,
					CreatedAt:  s.CreatedAt,
					ActivityAt: s.ActivityAt,
				})
			}
			sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(items)

			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(items)

			case outputOpts.Is(output.OutputQuiet):
				for _, s := range items {
					_, _ = fmt.Fprintln(out, s.Name)
				}
				return nil
			}

			if len(items) == 0 {
				_, _ = fmt.Fprintln(out, "No tmux sessions found.")
				return nil
			}

			_, _ = fmt.Fprintln(out, "Sessions:")
			for _, s := range items {
				_, _ = fmt.Fprintf(out, "  %s  windows=%d  attached=%d  created=%s  activity=%s\n",
					s.Name,
					s.Windows,
					s.Attached,
					formatTime(s.CreatedAt),
					formatRelative(s.ActivityAt),
				)
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	return cmd
}
