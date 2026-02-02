// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

func newAttachCmd() *cobra.Command {
	var sessionFlag string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "attach [session]",
		Short: "Attach to a tmux session",
		Long:  "Attach your terminal to a tmux session. Defaults to 'arc-tmux' managed session.",
		Example: `  # Attach to the managed session
  arc-tmux attach

  # Explicit session name
  arc-tmux attach prod`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			if tmux.InTmux() {
				return newCodedError(errNoTmuxClient, "already inside tmux; open a new terminal to attach", nil)
			}

			var target string
			if len(args) > 0 {
				target = strings.TrimSpace(args[0])
			}
			if target == "" {
				target = strings.TrimSpace(sessionFlag)
			}
			if target == "" {
				target = resolveManagedSession()
			}

			resolved, shouldStyle, err := resolveAgentSessionName(target)
			if err != nil {
				return err
			}
			target = resolved

			if err := tmux.EnsureSession(target); err != nil {
				return fmt.Errorf("failed to ensure session %q: %w", target, err)
			}
			if err := applyAgentStyleIfNeeded(target, shouldStyle); err != nil {
				return err
			}

			if !outputOpts.Is(output.OutputTable) {
				return writeAttachResult(cmd, outputOpts, attachResult{Session: target})
			}
			return tmux.Attach(target)
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&sessionFlag, "session", "", "Session to attach (default: arc-tmux)")

	return cmd
}

func newCleanupCmd() *cobra.Command {
	var session string
	var yes bool
	var dryRun bool
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Kill managed tmux session",
		Long:  "Force-kill the managed tmux session (defaults to 'arc-tmux').",
		Example: `  arc-tmux cleanup
  arc-tmux cleanup --session fe --yes`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			if session == "" {
				session = resolveManagedSession()
			}

			resolved, err := resolveExistingSessionName(session)
			if err != nil {
				return err
			}
			session = resolved

			if dryRun {
				return writeCleanupResult(cmd, outputOpts, cleanupResult{Session: session, DryRun: true})
			}

			if !yes {
				ok, err := confirmPrompt(cmd, fmt.Sprintf("Kill tmux session %q? [y/N]: ", session))
				if err != nil {
					return err
				}
				if !ok {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			if err := tmux.Cleanup(session); err != nil {
				return err
			}
			return writeCleanupResult(cmd, outputOpts, cleanupResult{Session: session, Killed: true})
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&session, "session", "", "Session to kill (default: arc-tmux)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without killing")

	return cmd
}

func newLaunchCmd() *cobra.Command {
	var split string
	var session string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "launch [command]",
		Short: "Launch a shell or command in a new pane/window",
		Long: `Launch a new tmux pane/window and immediately run a command.

Inside tmux: splits the current window.
Outside tmux: ensures the managed session exists and opens a fresh window there.
Commands are executed via "sh -lc", so full shell strings are supported.`,
		Example: `  # Split current tmux window
  arc-tmux launch "htop" --split v

  # Outside tmux, create/open the managed session
  arc-tmux launch`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			var command string
			if len(args) > 0 {
				command = args[0]
			}

			sess := session
			if !tmux.InTmux() && strings.TrimSpace(sess) == "" {
				sess = resolveManagedSession()
			}
			if !tmux.InTmux() {
				resolved, shouldStyle, err := resolveAgentSessionName(sess)
				if err != nil {
					return err
				}
				sess = resolved
				if err := tmux.EnsureSession(sess); err != nil {
					return fmt.Errorf("failed to ensure session %q: %w", sess, err)
				}
				if err := applyAgentStyleIfNeeded(sess, shouldStyle); err != nil {
					return err
				}
			}

			paneID, err := tmux.Launch(sess, command, split)
			if err != nil {
				return err
			}
			if isAgentSessionName(sess) {
				if details, err := tmux.PaneDetailsForTarget(paneID); err == nil {
					if err := tmux.ApplyAgentWindowStyle(details.Session, details.WindowIndex); err != nil {
						return err
					}
				}
			}

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				result := launchResult{PaneID: paneID}
				fillLaunchResult(&result, paneID)
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			case outputOpts.Is(output.OutputYAML):
				result := launchResult{PaneID: paneID}
				fillLaunchResult(&result, paneID)
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(result)
			case outputOpts.Is(output.OutputQuiet):
				_, _ = fmt.Fprintln(out, paneID)
				return nil
			}
			_, _ = fmt.Fprintln(out, paneID)
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&split, "split", "", "Inside tmux: split direction (h|v)")
	cmd.Flags().StringVar(&session, "session", "", "Managed session name when outside tmux")

	return cmd
}

func newWindowsCmd() *cobra.Command {
	var session string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "windows",
		Short: "List tmux windows",
		Long:  "List windows for the current session (inside tmux) or managed session (outside).",
		Example: `  arc-tmux windows
  arc-tmux windows --session fe`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			if session != "" {
				resolved, err := resolveSessionTarget(session)
				if err != nil {
					return err
				}
				session = resolved
			}
			if session == "" {
				if tmux.InTmux() {
					sess, _, _, _, err := tmux.CurrentLocation()
					if err != nil {
						return err
					}
					session = sess
				} else {
					session = resolveManagedSession()
				}
			}

			wins, err := tmux.ListWindows(session)
			if err != nil {
				if errors.Is(err, tmux.ErrNoTmuxServer) {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No tmux server is running.")
					return nil
				}
				if errors.Is(err, tmux.ErrSessionNotFound) {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Tmux session %q is not running.\n", session)
					return nil
				}
				return err
			}

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(wins)
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(wins)
			case outputOpts.Is(output.OutputQuiet):
				for _, w := range wins {
					_, _ = fmt.Fprintf(out, "%s:%d\n", w.Session, w.WindowIndex)
				}
				return nil
			}

			if len(wins) == 0 {
				_, _ = fmt.Fprintln(out, "No windows found")
				return nil
			}

			bySess := map[string][]tmux.Window{}
			for _, w := range wins {
				bySess[w.Session] = append(bySess[w.Session], w)
			}

			sessions := make([]string, 0, len(bySess))
			for s := range bySess {
				sessions = append(sessions, s)
			}
			sort.Strings(sessions)

			for _, s := range sessions {
				_, _ = fmt.Fprintf(out, "%s:\n", s)
				ws := bySess[s]
				sort.Slice(ws, func(i, j int) bool { return ws[i].WindowIndex < ws[j].WindowIndex })
				for _, w := range ws {
					status := "inactive"
					if w.Active {
						status = "active"
					}
					_, _ = fmt.Fprintf(out, "  %d  (%s)  %s\n", w.WindowIndex, status, w.Name)
				}
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&session, "session", "", "Session name or selector (@current|@managed)")

	return cmd
}

func resolveManagedSession() string {
	if env := strings.TrimSpace(os.Getenv("ARC_TMUX_SESSION")); env != "" {
		return env
	}
	return "arc-tmux"
}

type cleanupResult struct {
	Session string `json:"session" yaml:"session"`
	DryRun  bool   `json:"dry_run" yaml:"dry_run"`
	Killed  bool   `json:"killed" yaml:"killed"`
}

func writeCleanupResult(cmd *cobra.Command, outputOpts output.OutputOptions, result cleanupResult) error {
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
		return nil
	}
	if result.DryRun {
		_, _ = fmt.Fprintf(out, "Dry run: would kill tmux session %q\n", result.Session)
		return nil
	}
	if result.Killed {
		_, _ = fmt.Fprintf(out, "Killed tmux session %q\n", result.Session)
		return nil
	}
	return nil
}

type attachResult struct {
	Session string `json:"session" yaml:"session"`
}

func writeAttachResult(cmd *cobra.Command, outputOpts output.OutputOptions, result attachResult) error {
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
		_, _ = fmt.Fprintln(out, result.Session)
		return nil
	}
	_, _ = fmt.Fprintf(out, "Attach session %q\n", result.Session)
	return nil
}

type launchResult struct {
	PaneID      string `json:"pane_id" yaml:"pane_id"`
	Session     string `json:"session,omitempty" yaml:"session,omitempty"`
	WindowIndex int    `json:"window_index,omitempty" yaml:"window_index,omitempty"`
	PaneIndex   int    `json:"pane_index,omitempty" yaml:"pane_index,omitempty"`
}

func fillLaunchResult(result *launchResult, paneID string) {
	session, window, pane := parseFormattedPaneID(paneID)
	result.Session = session
	result.WindowIndex = window
	result.PaneIndex = pane
}
