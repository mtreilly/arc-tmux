// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

type ensureResult struct {
	Session        string `json:"session" yaml:"session"`
	Window         string `json:"window" yaml:"window"`
	WindowIndex    int    `json:"window_index" yaml:"window_index"`
	PaneID         string `json:"pane_id" yaml:"pane_id"`
	PaneTitle      string `json:"pane_title,omitempty" yaml:"pane_title,omitempty"`
	CreatedSession bool   `json:"created_session" yaml:"created_session"`
	CreatedWindow  bool   `json:"created_window" yaml:"created_window"`
	CreatedPane    bool   `json:"created_pane" yaml:"created_pane"`
	AddedPanes     int    `json:"added_panes" yaml:"added_panes"`
	LayoutApplied  bool   `json:"layout_applied" yaml:"layout_applied"`
}

func newEnsureCmd() *cobra.Command {
	var session string
	var window string
	var paneTitle string
	var panes int
	var layout string
	var split string
	var cwd string
	var envVars []string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "ensure [command]",
		Short: "Ensure a session/window/pane exists (idempotent)",
		Long: `Ensure a session, window, and optional pane exist without duplication.

If the target already exists, this is a no-op. When creating panes, optional
command/cwd/env are only applied to newly created panes.`,
		Example: `  # Ensure a window exists, run a command once if created
  arc-tmux ensure "npm test" --session dev --window build

  # Ensure a named pane exists with a layout
  arc-tmux ensure "npm run dev" --session dev --window api --pane-title server --panes 2 --layout tiled`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			window = strings.TrimSpace(window)
			if window == "" {
				return errors.New("--window is required")
			}
			if panes < 0 {
				return errors.New("--panes must be >= 0")
			}
			paneTitle = strings.TrimSpace(paneTitle)

			var command string
			if len(args) > 0 {
				command = args[0]
			}

			envPairs, err := parseEnvVars(envVars)
			if err != nil {
				return newCodedError(errInvalidEnv, err.Error(), err)
			}
			paneCommand := buildRunCommand(command, strings.TrimSpace(cwd), envPairs)
			spawnCommand := buildRunCommand("", strings.TrimSpace(cwd), envPairs)

			sess, shouldStyle, err := resolveEnsureSession(session)
			if err != nil {
				return err
			}

			createdSession := false
			exists, err := tmux.HasSession(sess)
			if err != nil {
				return err
			}
			if !exists {
				createdSession = true
			}
			if err := tmux.EnsureSession(sess); err != nil {
				return fmt.Errorf("failed to ensure session %q: %w", sess, err)
			}
			if createdSession {
				if err := applyAgentStyleIfNeeded(sess, shouldStyle); err != nil {
					return err
				}
			}

			result := ensureResult{Session: sess, Window: window, PaneTitle: paneTitle}

			wins, err := tmux.ListWindows(sess)
			if err != nil {
				return err
			}

			win, found := findWindowByName(wins, window)
			windowCreated := false
			paneCreated := false
			addedPanes := 0
			layoutApplied := false
			var windowIndex int
			var targetPaneID string
			windowTarget := ""

			if !found {
				paneID, err := tmux.NewWindow(sess, window, paneCommand)
				if err != nil {
					return err
				}
				windowCreated = true
				paneCreated = true
				targetPaneID = strings.TrimSpace(paneID)
				parsedSession, parsedWindow, _ := parseFormattedPaneID(targetPaneID)
				if parsedSession == "" {
					pane, err := tmux.PaneDetailsForTarget(targetPaneID)
					if err != nil {
						return err
					}
					windowIndex = pane.WindowIndex
				} else {
					windowIndex = parsedWindow
				}
				windowTarget = fmt.Sprintf("%s:%d", sess, windowIndex)

				if isAgentSessionName(sess) {
					if err := tmux.ApplyAgentWindowStyle(sess, windowIndex); err != nil {
						return err
					}
				}
				if paneTitle != "" {
					if err := tmux.SetPaneTitle(targetPaneID, paneTitle); err != nil {
						return err
					}
				}
				if panes > 1 {
					current := 1
					for current < panes {
						if _, err := tmux.SplitWindow(windowTarget, split, spawnCommand); err != nil {
							return err
						}
						addedPanes++
						current++
					}
				}
			} else {
				windowIndex = win.WindowIndex
				windowTarget = fmt.Sprintf("%s:%d", sess, windowIndex)

				panesList, err := panesForWindow(sess, windowIndex)
				if err != nil {
					return err
				}

				if paneTitle != "" {
					if match := findPaneByTitle(panesList, paneTitle); match != nil {
						targetPaneID = formattedPaneID(match)
					} else {
						paneID, err := tmux.SplitWindow(windowTarget, split, paneCommand)
						if err != nil {
							return err
						}
						paneCreated = true
						targetPaneID = strings.TrimSpace(paneID)
						if err := tmux.SetPaneTitle(targetPaneID, paneTitle); err != nil {
							return err
						}
					}
				}

				if targetPaneID == "" {
					paneID, err := pickPaneID(panesList, sess, windowIndex)
					if err != nil {
						return err
					}
					targetPaneID = paneID
				}

				current := len(panesList)
				if paneCreated {
					current++
				}
				if panes > 0 && current < panes {
					for current < panes {
						if _, err := tmux.SplitWindow(windowTarget, split, spawnCommand); err != nil {
							return err
						}
						addedPanes++
						current++
					}
				}
			}

			if layout != "" && (windowCreated || paneCreated || addedPanes > 0) {
				if err := tmux.SelectLayout(windowTarget, layout); err != nil {
					return err
				}
				layoutApplied = true
			}

			result.CreatedSession = createdSession
			result.CreatedWindow = windowCreated
			result.CreatedPane = paneCreated
			result.AddedPanes = addedPanes
			result.LayoutApplied = layoutApplied
			result.WindowIndex = windowIndex
			result.PaneID = targetPaneID

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(result)
			case outputOpts.Is(output.OutputQuiet):
				if result.PaneID != "" {
					_, _ = fmt.Fprintln(out, result.PaneID)
				}
				return nil
			}

			if result.CreatedWindow {
				_, _ = fmt.Fprintf(out, "Ensured window %q in session %q (index %d).\n", result.Window, result.Session, result.WindowIndex)
			} else {
				_, _ = fmt.Fprintf(out, "Window %q already exists in session %q (index %d).\n", result.Window, result.Session, result.WindowIndex)
			}
			if result.PaneID != "" {
				status := "existing"
				if result.CreatedPane {
					status = "created"
				}
				if result.PaneTitle != "" {
					_, _ = fmt.Fprintf(out, "Pane %s (%s, title=%q).\n", result.PaneID, status, result.PaneTitle)
				} else {
					_, _ = fmt.Fprintf(out, "Pane %s (%s).\n", result.PaneID, status)
				}
			}
			if result.AddedPanes > 0 {
				_, _ = fmt.Fprintf(out, "Added panes: %d\n", result.AddedPanes)
			}
			if result.LayoutApplied {
				_, _ = fmt.Fprintf(out, "Layout applied: %s\n", layout)
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&session, "session", "", "Session name or selector (@current|@managed)")
	cmd.Flags().StringVar(&window, "window", "", "Window name to ensure")
	cmd.Flags().StringVar(&paneTitle, "pane-title", "", "Pane title to ensure within the window")
	cmd.Flags().IntVar(&panes, "panes", 0, "Ensure at least N panes in the window (0 to skip)")
	cmd.Flags().StringVar(&layout, "layout", "", "Apply tmux layout when panes are created (e.g., tiled, even-horizontal)")
	cmd.Flags().StringVar(&split, "split", "", "Split direction when creating panes (h|v)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "Working directory for newly created panes")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Set environment variables for newly created panes (KEY=VAL). Repeatable.")

	return cmd
}

func resolveEnsureSession(raw string) (string, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "@") {
		resolved, err := resolveSessionTarget(trimmed)
		if err != nil {
			return "", false, err
		}
		trimmed = resolved
	}
	if trimmed == "" {
		if tmux.InTmux() {
			sess, _, _, _, err := tmux.CurrentLocation()
			if err != nil {
				return "", false, err
			}
			trimmed = sess
		} else {
			trimmed = resolveManagedSession()
		}
	}
	return resolveAgentSessionName(trimmed)
}

func findWindowByName(wins []tmux.Window, name string) (tmux.Window, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return tmux.Window{}, false
	}
	matches := make([]tmux.Window, 0)
	for _, w := range wins {
		if w.Name == trimmed {
			matches = append(matches, w)
		}
	}
	if len(matches) == 0 {
		return tmux.Window{}, false
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].WindowIndex < matches[j].WindowIndex })
	return matches[0], true
}

func panesForWindow(session string, windowIndex int) ([]tmux.PaneDetails, error) {
	panes, err := tmux.ListPanesDetailed()
	if err != nil {
		return nil, err
	}
	filtered := make([]tmux.PaneDetails, 0)
	for _, p := range panes {
		if p.Session == session && p.WindowIndex == windowIndex {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

func findPaneByTitle(panes []tmux.PaneDetails, title string) *tmux.PaneDetails {
	if strings.TrimSpace(title) == "" {
		return nil
	}
	matches := make([]tmux.PaneDetails, 0)
	for _, p := range panes {
		if p.Title == title {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].PaneIndex < matches[j].PaneIndex })
	return &matches[0]
}

func formattedPaneID(pane *tmux.PaneDetails) string {
	if pane == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d.%d", pane.Session, pane.WindowIndex, pane.PaneIndex)
}

func pickPaneID(panes []tmux.PaneDetails, session string, windowIndex int) (string, error) {
	if len(panes) == 0 {
		return "", errors.New("no panes found in window")
	}
	for _, p := range panes {
		if p.Active {
			return formattedPaneID(&p), nil
		}
	}
	sort.Slice(panes, func(i, j int) bool { return panes[i].PaneIndex < panes[j].PaneIndex })
	return fmt.Sprintf("%s:%d.%d", session, windowIndex, panes[0].PaneIndex), nil
}
