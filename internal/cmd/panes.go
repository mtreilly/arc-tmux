// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

type paneSnapshot struct {
	Session      string    `json:"session" yaml:"session"`
	WindowIndex  int       `json:"window_index" yaml:"window_index"`
	WindowName   string    `json:"window_name" yaml:"window_name"`
	WindowActive bool      `json:"window_active" yaml:"window_active"`
	PaneIndex    int       `json:"pane_index" yaml:"pane_index"`
	PaneID       string    `json:"pane_id" yaml:"pane_id"`
	FormattedID  string    `json:"formatted_id" yaml:"formatted_id"`
	Active       bool      `json:"active" yaml:"active"`
	Command      string    `json:"command" yaml:"command"`
	Title        string    `json:"title" yaml:"title"`
	Path         string    `json:"path" yaml:"path"`
	PID          int       `json:"pid" yaml:"pid"`
	ActivityAt   time.Time `json:"activity_at" yaml:"activity_at"`
}

func newPanesCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var session string
	var window int
	var command string
	var title string
	var path string
	var fuzzy bool

	cmd := &cobra.Command{
		Use:   "panes",
		Short: "List tmux panes with metadata",
		Long:  "List tmux panes across sessions with PID, cwd, and activity timestamps.",
		Example: `  arc-tmux panes
  arc-tmux panes --session fe --window 2
  arc-tmux panes --command node --path /srv
  arc-tmux panes --command ndsr --fuzzy
  arc-tmux panes --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}

			resolvedSession, err := resolveSessionTarget(session)
			if err != nil {
				return err
			}
			session = resolvedSession

			panes, err := tmux.ListPanesDetailed()
			if err != nil {
				if err == tmux.ErrNoTmuxServer {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No tmux server is running.")
					return nil
				}
				return err
			}

			items := make([]paneSnapshot, 0, len(panes))
			for _, p := range panes {
				if session != "" && p.Session != session {
					continue
				}
				if window >= 0 && p.WindowIndex != window {
					continue
				}
				if !matchesFilter(p.Command, command, fuzzy) {
					continue
				}
				if !matchesFilter(p.Title, title, fuzzy) {
					continue
				}
				if !matchesFilter(p.Path, path, fuzzy) {
					continue
				}
				items = append(items, toPaneSnapshot(p))
			}

			sort.Slice(items, func(i, j int) bool {
				if items[i].Session != items[j].Session {
					return items[i].Session < items[j].Session
				}
				if items[i].WindowIndex != items[j].WindowIndex {
					return items[i].WindowIndex < items[j].WindowIndex
				}
				return items[i].PaneIndex < items[j].PaneIndex
			})

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
				for _, p := range items {
					_, _ = fmt.Fprintln(out, p.FormattedID)
				}
				return nil
			}

			if len(items) == 0 {
				_, _ = fmt.Fprintln(out, "No tmux panes found.")
				return nil
			}

			_, _ = fmt.Fprintln(out, "Panes:")
			for _, p := range items {
				active := "inactive"
				if p.Active {
					active = "active"
				}
				winActive := "inactive"
				if p.WindowActive {
					winActive = "active"
				}
				windowLabel := fmt.Sprintf("%s:%d", p.Session, p.WindowIndex)
				if strings.TrimSpace(p.WindowName) != "" {
					windowLabel = fmt.Sprintf("%s (%s)", windowLabel, p.WindowName)
				}
				_, _ = fmt.Fprintf(out, "  %s  %s  pane=%d  pid=%d  cmd=%s  path=%s  title=%s  win=%s (%s)  activity=%s\n",
					p.FormattedID,
					active,
					p.PaneIndex,
					p.PID,
					p.Command,
					p.Path,
					p.Title,
					windowLabel,
					winActive,
					formatRelative(p.ActivityAt),
				)
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&session, "session", "", "Filter by session name or selector (@current|@managed)")
	cmd.Flags().IntVar(&window, "window", -1, "Filter by window index")
	cmd.Flags().StringVar(&command, "command", "", "Filter by current command (substring)")
	cmd.Flags().StringVar(&title, "title", "", "Filter by pane title (substring)")
	cmd.Flags().StringVar(&path, "path", "", "Filter by pane path (substring)")
	cmd.Flags().BoolVar(&fuzzy, "fuzzy", false, "Use fuzzy matching for command/title/path filters")
	return cmd
}

func matchesFilter(value string, filter string, fuzzy bool) bool {
	if filter == "" {
		return true
	}
	if fuzzy {
		return fuzzyMatch(value, filter)
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
}

func toPaneSnapshot(p tmux.PaneDetails) paneSnapshot {
	return paneSnapshot{
		Session:      p.Session,
		WindowIndex:  p.WindowIndex,
		WindowName:   p.WindowName,
		WindowActive: p.WindowActive,
		PaneIndex:    p.PaneIndex,
		PaneID:       p.PaneID,
		FormattedID:  fmt.Sprintf("%s:%d.%d", p.Session, p.WindowIndex, p.PaneIndex),
		Active:       p.Active,
		Command:      p.Command,
		Title:        p.Title,
		Path:         p.Path,
		PID:          p.PID,
		ActivityAt:   p.ActivityAt,
	}
}
