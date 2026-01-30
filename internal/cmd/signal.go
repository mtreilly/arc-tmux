// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

type signalResult struct {
	PaneID string `json:"pane_id" yaml:"pane_id"`
	PID    int    `json:"pid" yaml:"pid"`
	Signal string `json:"signal" yaml:"signal"`
}

func newSignalCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var paneArg string
	var sig string

	cmd := &cobra.Command{
		Use:   "signal",
		Short: "Send a signal to a pane's PID",
		Long:  "Send a signal to the process running in a tmux pane.",
		Example: `  arc-tmux signal --pane=fe:2.0 --signal TERM
  arc-tmux signal --pane=@current --signal KILL`,
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
			if pane.PID <= 0 {
				return fmt.Errorf("pane PID not available")
			}

			parsed, name, err := parseSignal(sig)
			if err != nil {
				return err
			}

			if err := syscall.Kill(pane.PID, parsed); err != nil {
				return fmt.Errorf("signal %s to pid %d: %w", name, pane.PID, err)
			}

			result := signalResult{PaneID: target, PID: pane.PID, Signal: name}
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
				_, _ = fmt.Fprintln(out, result.PID)
				return nil
			}
			_, _ = fmt.Fprintf(out, "Sent %s to pid %d (%s)\n", name, pane.PID, target)
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active, @name)")
	cmd.Flags().StringVar(&sig, "signal", "TERM", "Signal name or number (e.g., TERM, KILL, INT)")
	_ = cmd.MarkFlagRequired("pane")
	return cmd
}

func parseSignal(raw string) (syscall.Signal, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "TERM"
	}
	upper := strings.ToUpper(trimmed)
	upper = strings.TrimPrefix(upper, "SIG")
	signalMap := map[string]syscall.Signal{
		"HUP":  syscall.SIGHUP,
		"INT":  syscall.SIGINT,
		"QUIT": syscall.SIGQUIT,
		"KILL": syscall.SIGKILL,
		"TERM": syscall.SIGTERM,
		"USR1": syscall.SIGUSR1,
		"USR2": syscall.SIGUSR2,
	}
	if sig, ok := signalMap[upper]; ok {
		return sig, "SIG" + upper, nil
	}
	if num, err := strconv.Atoi(upper); err == nil {
		return syscall.Signal(num), fmt.Sprintf("SIG%d", num), nil
	}
	return 0, "", newCodedError(errSignalUnsupported, fmt.Sprintf("unsupported signal: %s", raw), nil)
}
