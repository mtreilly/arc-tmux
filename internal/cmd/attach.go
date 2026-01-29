// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func newAttachCmd() *cobra.Command {
	var sessionFlag string

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
			if tmux.InTmux() {
				return errors.New("already inside tmux; open a new terminal to attach")
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

			if err := tmux.EnsureSession(target); err != nil {
				return fmt.Errorf("failed to ensure session %q: %w", target, err)
			}

			return tmux.Attach(target)
		},
	}

	cmd.Flags().StringVar(&sessionFlag, "session", "", "Session to attach (default: arc-tmux)")

	return cmd
}

func newCleanupCmd() *cobra.Command {
	var session string
	var yes bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Kill managed tmux session",
		Long:  "Force-kill the managed tmux session (defaults to 'arc-tmux').",
		Example: `  arc-tmux cleanup
  arc-tmux cleanup --session fe --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if session == "" {
				session = resolveManagedSession()
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry run: would kill tmux session %q\n", session)
				return nil
			}

			if !yes {
				ok, err := confirmPrompt(cmd, fmt.Sprintf("Kill tmux session %q? [y/N]: ", session))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			return tmux.Cleanup(session)
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session to kill (default: arc-tmux)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview without killing")

	return cmd
}

func newLaunchCmd() *cobra.Command {
	var split string
	var session string

	cmd := &cobra.Command{
		Use:   "launch [command]",
		Short: "Launch a shell or command in a new pane/window",
		Long: `Launch a new tmux pane/window and immediately run a command.

Inside tmux: splits the current window.
Outside tmux: ensures the managed session exists and opens a fresh window there.`,
		Example: `  # Split current tmux window
  arc-tmux launch "htop" --split v

  # Outside tmux, create/open the managed session
  arc-tmux launch`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var command string
			if len(args) > 0 {
				command = args[0]
			}

			sess := session
			if !tmux.InTmux() && strings.TrimSpace(sess) == "" {
				sess = resolveManagedSession()
			}

			paneID, err := tmux.Launch(sess, command, split)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), paneID)
			return nil
		},
	}

	cmd.Flags().StringVar(&split, "split", "", "Inside tmux: split direction (h|v)")
	cmd.Flags().StringVar(&session, "session", "", "Managed session name when outside tmux")

	return cmd
}

func newWindowsCmd() *cobra.Command {
	var session string

	cmd := &cobra.Command{
		Use:   "windows",
		Short: "List tmux windows",
		Long:  "List windows for the current session (inside tmux) or managed session (outside).",
		Example: `  arc-tmux windows
  arc-tmux windows --session fe`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if session == "" && !tmux.InTmux() {
				session = resolveManagedSession()
			}

			wins, err := tmux.ListWindows(session)
			if err != nil {
				if errors.Is(err, tmux.ErrNoTmuxServer) {
					fmt.Fprintln(cmd.OutOrStdout(), "No tmux server is running.")
					return nil
				}
				if errors.Is(err, tmux.ErrSessionNotFound) {
					fmt.Fprintf(cmd.OutOrStdout(), "Tmux session %q is not running.\n", session)
					return nil
				}
				return err
			}

			if len(wins) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No windows found")
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
				fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", s)
				ws := bySess[s]
				sort.Slice(ws, func(i, j int) bool { return ws[i].WindowIndex < ws[j].WindowIndex })
				for _, w := range ws {
					status := "inactive"
					if w.Active {
						status = "active"
					}
					fmt.Fprintf(cmd.OutOrStdout(), "  %d  (%s)  %s\n", w.WindowIndex, status, w.Name)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&session, "session", "", "Session name")

	return cmd
}

func resolveManagedSession() string {
	if env := strings.TrimSpace(os.Getenv("ARC_TMUX_SESSION")); env != "" {
		return env
	}
	return "arc-tmux"
}
