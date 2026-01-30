// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

type paneInfo struct {
	Title       string `json:"title" yaml:"title"`
	Active      bool   `json:"active" yaml:"active"`
	Command     string `json:"command" yaml:"command"`
	FormattedID string `json:"formatted_id" yaml:"formatted_id"`
}

func newListCmd() *cobra.Command {
	var flat bool
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available tmux panes",
		Long:  "Discover panes grouped under sessions/windows, including formatted IDs, commands, and active indicators.",
		Example: `  arc-tmux list
  arc-tmux list --flat
  arc-tmux list --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}

			rawPanes, err := tmux.ListPanes()
			if err != nil {
				if err == tmux.ErrNoTmuxServer {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No tmux server is running.")
					return nil
				}
				return err
			}

			panes := make([]paneInfo, 0, len(rawPanes))
			for _, p := range rawPanes {
				panes = append(panes, paneInfo{
					Title:       p.Title,
					Active:      p.Active,
					Command:     p.Command,
					FormattedID: p.FormattedID(),
				})
			}
			sort.Slice(panes, func(i, j int) bool { return panes[i].FormattedID < panes[j].FormattedID })

			out := cmd.OutOrStdout()

			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(panes)

			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(panes)

			case outputOpts.Is(output.OutputQuiet):
				for _, p := range panes {
					_, _ = fmt.Fprintln(out, p.FormattedID)
				}
				return nil
			}

			if len(panes) == 0 {
				_, _ = fmt.Fprintln(out, "No tmux panes found.")
				return nil
			}

			if flat {
				_, _ = fmt.Fprintln(out, "Available tmux panes:")
				for _, p := range panes {
					status := "inactive"
					if p.Active {
						status = "active"
					}
					_, _ = fmt.Fprintf(out, "  - %s  title=%s  cmd=%s  (%s)\n", p.FormattedID, p.Title, p.Command, status)
				}
				return nil
			}

			// Group by session:window
			grouped := groupPanesByWindow(panes)
			sessions := make([]string, 0, len(grouped))
			for s := range grouped {
				sessions = append(sessions, s)
			}
			sort.Strings(sessions)

			_, _ = fmt.Fprintln(out, "Tmux windows and panes:")
			for _, sess := range sessions {
				wins := grouped[sess]
				winKeys := make([]string, 0, len(wins))
				for k := range wins {
					winKeys = append(winKeys, k)
				}
				sort.Strings(winKeys)

				_, _ = fmt.Fprintf(out, "%s:\n", sess)
				for _, wkey := range winKeys {
					panesInWin := wins[wkey]
					winActive := false
					for _, p := range panesInWin {
						if p.Active {
							winActive = true
							break
						}
					}
					wstatus := "inactive"
					if winActive {
						wstatus = "active"
					}
					_, _ = fmt.Fprintf(out, "  %s  (%s)\n", wkey, wstatus)
					for _, p := range panesInWin {
						pstatus := "inactive"
						if p.Active {
							pstatus = "active"
						}
						_, _ = fmt.Fprintf(out, "    - %s  title=%s  cmd=%s  (%s)\n", p.FormattedID, p.Title, p.Command, pstatus)
					}
				}
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().BoolVar(&flat, "flat", false, "Print a flat list instead of grouping by window")

	return cmd
}

func groupPanesByWindow(panes []paneInfo) map[string]map[string][]paneInfo {
	result := make(map[string]map[string][]paneInfo)
	for _, p := range panes {
		sess, win := splitFormattedID(p.FormattedID)
		if sess == "" {
			sess = "?"
		}
		if _, ok := result[sess]; !ok {
			result[sess] = make(map[string][]paneInfo)
		}
		winKey := fmt.Sprintf("%s:%s", sess, win)
		result[sess][winKey] = append(result[sess][winKey], p)
	}
	return result
}
