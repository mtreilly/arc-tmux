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

type statusSnapshot struct {
	InTmux         bool         `json:"in_tmux" yaml:"in_tmux"`
	Session        string       `json:"session,omitempty" yaml:"session,omitempty"`
	WindowIndex    int          `json:"window_index,omitempty" yaml:"window_index,omitempty"`
	WindowName     string       `json:"window_name,omitempty" yaml:"window_name,omitempty"`
	PaneIndex      int          `json:"pane_index,omitempty" yaml:"pane_index,omitempty"`
	PaneID         string       `json:"pane_id,omitempty" yaml:"pane_id,omitempty"`
	Panes          []statusPane `json:"panes,omitempty" yaml:"panes,omitempty"`
	ManagedSession string       `json:"managed_session,omitempty" yaml:"managed_session,omitempty"`
}

type statusPane struct {
	ID      string `json:"id" yaml:"id"`
	Title   string `json:"title,omitempty" yaml:"title,omitempty"`
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
	Active  bool   `json:"active" yaml:"active"`
}

func newStatusCmd() *cobra.Command {
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current tmux location",
		Long:  "Inside tmux: prints your current session/window plus all panes. Outside: shows managed session.",
		Example: `  arc-tmux status
  arc-tmux status --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}

			var snap statusSnapshot

			if tmux.InTmux() {
				sess, win, pane, fid, err := tmux.CurrentLocation()
				if err != nil {
					return err
				}

				winName := ""
				wins, err := tmux.ListWindows(sess)
				if err == nil {
					for _, w := range wins {
						if w.WindowIndex == win {
							winName = w.Name
							break
						}
					}
				}

				panes, _ := tmux.ListPanes()
				prefix := fmt.Sprintf("%s:%d.", sess, win)
				var currentPanes []statusPane
				for _, p := range panes {
					if strings.HasPrefix(p.FormattedID(), prefix) {
						currentPanes = append(currentPanes, statusPane{
							ID:      p.FormattedID(),
							Title:   p.Title,
							Command: p.Command,
							Active:  p.Active,
						})
					}
				}

				snap = statusSnapshot{
					InTmux:      true,
					Session:     sess,
					WindowIndex: win,
					WindowName:  winName,
					PaneIndex:   pane,
					PaneID:      fid,
					Panes:       currentPanes,
				}
			} else {
				snap = statusSnapshot{
					InTmux:         false,
					ManagedSession: resolveManagedSession(),
				}
			}

			out := cmd.OutOrStdout()

			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(snap)

			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				return enc.Encode(snap)

			case outputOpts.Is(output.OutputQuiet):
				if snap.PaneID != "" {
					fmt.Fprintln(out, snap.PaneID)
				} else if snap.ManagedSession != "" {
					fmt.Fprintln(out, snap.ManagedSession)
				}
				return nil

			default:
				if snap.InTmux {
					fmt.Fprintf(out, "Current: %s\n", snap.PaneID)
					fmt.Fprintf(out, "Window:  %s:%d", snap.Session, snap.WindowIndex)
					if snap.WindowName != "" {
						fmt.Fprintf(out, " (%s)", snap.WindowName)
					}
					fmt.Fprintln(out)

					if len(snap.Panes) > 0 {
						fmt.Fprintln(out, "\nPanes:")
						for _, p := range snap.Panes {
							mark := " "
							if p.Active {
								mark = "*"
							}
							fmt.Fprintf(out, "%s %-14s %-16s %s\n", mark, p.ID, p.Command, p.Title)
						}
					}
				} else {
					fmt.Fprintf(out, "Managed session: %s\n", snap.ManagedSession)
					fmt.Fprintln(out, "Not currently inside tmux.")
				}
				return nil
			}
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)

	return cmd
}

func splitFormattedID(fid string) (session string, window string) {
	if fid == "" {
		return "", ""
	}
	parts := strings.SplitN(fid, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	session = parts[0]
	rest := parts[1]
	parts2 := strings.SplitN(rest, ".", 2)
	if len(parts2) < 1 {
		return session, ""
	}
	window = parts2[0]
	return session, window
}
