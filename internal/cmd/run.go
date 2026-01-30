// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

func newRunCmd() *cobra.Command {
	var paneArg string
	var idle, timeout float64
	var lines int
	var exitCode bool
	var exitTag string
	var exitPropagate bool
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "run [command]",
		Short: "Send, wait idle, then capture output",
		Long:  "Fire-and-forget automation: send a command, wait until the pane quiets down, then print the captured output.",
		Example: `  # Run tests and capture the logs
  arc-tmux run "npm test" --pane=fe:2.0 --timeout=180

  # Long-running lint with custom idle threshold
  arc-tmux run "make lint" --pane=fe:2.0 --idle=5 --timeout=600

  # Capture output and exit code in JSON
  arc-tmux run "npm test" --pane=fe:2.0 --exit-code --output json`,
		Args: cobra.MinimumNArgs(1),
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

			text := strings.Join(args, " ")
			if exitCode {
				text = wrapCommandForExit(text, exitTag)
			}

			if err := tmux.SendLiteral(target, text, true, 0); err != nil {
				return err
			}

			if timeout <= 0 {
				timeout = 60
			}

			waitErr := tmux.WaitIdle(target, time.Duration(idle*float64(time.Second)), time.Duration(timeout*float64(time.Second)))

			s, err := tmux.Capture(target, lines)
			if err != nil {
				return err
			}

			capture := s
			var codePtr *int
			var found bool
			if exitCode {
				capture, codePtr, found = extractExitCode(capture, exitTag)
			}

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				result := runResult{
					Output:    capture,
					ExitCode:  codePtr,
					ExitFound: found,
				}
				if waitErr != nil {
					result.WaitError = waitErr.Error()
				}
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return err
				}
				return combineRunErrors(waitErr, exitPropagate, codePtr, found)

			case outputOpts.Is(output.OutputYAML):
				result := runResult{
					Output:    capture,
					ExitCode:  codePtr,
					ExitFound: found,
				}
				if waitErr != nil {
					result.WaitError = waitErr.Error()
				}
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				if err := enc.Encode(result); err != nil {
					return err
				}
				return combineRunErrors(waitErr, exitPropagate, codePtr, found)

			case outputOpts.Is(output.OutputQuiet):
				if exitCode && codePtr != nil {
					_, _ = fmt.Fprintln(out, *codePtr)
				}
				return combineRunErrors(waitErr, exitPropagate, codePtr, found)
			}

			if _, err := fmt.Fprint(out, capture); err != nil {
				return err
			}
			if exitCode {
				if codePtr != nil {
					_, _ = fmt.Fprintf(out, "\nExit code: %d\n", *codePtr)
				} else {
					_, _ = fmt.Fprintln(out, "\nExit code: unknown")
				}
			}
			return combineRunErrors(waitErr, exitPropagate, codePtr, found)
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	cmd.Flags().Float64Var(&idle, "idle", 2.0, "Seconds of inactivity to consider idle")
	cmd.Flags().Float64Var(&timeout, "timeout", 60.0, "Maximum seconds to wait")
	cmd.Flags().IntVar(&lines, "lines", 200, "Limit capture to last N lines (0 for full)")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Emit and parse a sentinel exit code")
	cmd.Flags().StringVar(&exitTag, "exit-tag", "__ARC_TMUX_EXIT:", "Sentinel tag for exit code parsing")
	cmd.Flags().BoolVar(&exitPropagate, "exit-propagate", false, "Return a non-zero exit when the parsed exit code is non-zero")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

type runResult struct {
	Output    string `json:"output" yaml:"output"`
	ExitCode  *int   `json:"exit_code,omitempty" yaml:"exit_code,omitempty"`
	ExitFound bool   `json:"exit_found" yaml:"exit_found"`
	WaitError string `json:"wait_error,omitempty" yaml:"wait_error,omitempty"`
}

func wrapCommandForExit(command string, tag string) string {
	if strings.TrimSpace(tag) == "" {
		tag = "__ARC_TMUX_EXIT:"
	}
	inner := fmt.Sprintf("%s; printf \"\\n%s%%d\\n\" $?", command, tag)
	return "sh -lc " + shellQuoteSingle(inner)
}

func shellQuoteSingle(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func extractExitCode(output string, tag string) (string, *int, bool) {
	if tag == "" {
		return output, nil, false
	}
	hadTrailingNewline := strings.HasSuffix(output, "\n")
	lines := splitLines(output)
	for i := len(lines) - 1; i >= 0; i-- {
		if !strings.Contains(lines[i], tag) {
			continue
		}
		idx := strings.Index(lines[i], tag)
		if idx < 0 {
			continue
		}
		codeStr := strings.TrimSpace(lines[i][idx+len(tag):])
		if codeStr == "" {
			continue
		}
		code, err := strconv.Atoi(codeStr)
		if err != nil {
			continue
		}
		lines = append(lines[:i], lines[i+1:]...)
		clean := strings.Join(lines, "\n")
		if hadTrailingNewline {
			clean += "\n"
		}
		return clean, &code, true
	}
	return output, nil, false
}

func combineRunErrors(waitErr error, exitPropagate bool, code *int, found bool) error {
	if waitErr != nil {
		return waitErr
	}
	if exitPropagate && found && code != nil && *code != 0 {
		return newCodedError(errCommandExit, fmt.Sprintf("command exited with %d", *code), nil)
	}
	return nil
}
