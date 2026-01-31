// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"crypto/rand"
	"encoding/hex"
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
	var segment bool
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
			var startTag string
			var endTag string
			if exitCode || segment {
				runID := newRunID()
				startTag = fmt.Sprintf("__ARC_TMUX_RUN_START:%s__", runID)
				endTag = fmt.Sprintf("__ARC_TMUX_RUN_END:%s__", runID)
				text = wrapCommandForRun(text, startTag, endTag, exitTag, exitCode)
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
			if exitCode || segment {
				clean, code, ok, windowFound := extractRunWindow(capture, startTag, endTag, exitTag, exitCode)
				if !windowFound && lines > 0 {
					if full, err := tmux.Capture(target, 0); err == nil {
						clean, code, ok, windowFound = extractRunWindow(full, startTag, endTag, exitTag, exitCode)
					}
				}
				if windowFound {
					capture = clean
					codePtr = code
					found = ok
				}
				if exitCode && !found {
					hadTrailingNewline := strings.HasSuffix(capture, "\n")
					cleanLines, code, ok := extractExitFromLines(splitLines(capture), exitTag)
					if ok {
						capture = strings.Join(cleanLines, "\n")
						if hadTrailingNewline {
							capture += "\n"
						}
						codePtr = code
						found = true
					}
				}
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
				return combineRunErrors(waitErr, exitPropagate, exitCode, codePtr, found)

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
				return combineRunErrors(waitErr, exitPropagate, exitCode, codePtr, found)

			case outputOpts.Is(output.OutputQuiet):
				if exitCode && codePtr != nil {
					_, _ = fmt.Fprintln(out, *codePtr)
				}
				return combineRunErrors(waitErr, exitPropagate, exitCode, codePtr, found)
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
			return combineRunErrors(waitErr, exitPropagate, exitCode, codePtr, found)
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
	cmd.Flags().BoolVar(&segment, "segment", false, "Capture only output for this command by inserting sentinel markers (runs via sh -lc)")
	_ = cmd.MarkFlagRequired("pane")

	return cmd
}

type runResult struct {
	Output    string `json:"output" yaml:"output"`
	ExitCode  *int   `json:"exit_code,omitempty" yaml:"exit_code,omitempty"`
	ExitFound bool   `json:"exit_found" yaml:"exit_found"`
	WaitError string `json:"wait_error,omitempty" yaml:"wait_error,omitempty"`
}

func wrapCommandForRun(command string, startTag string, endTag string, exitTag string, includeExit bool) string {
	if strings.TrimSpace(startTag) == "" {
		startTag = "__ARC_TMUX_RUN_START__"
	}
	if strings.TrimSpace(endTag) == "" {
		endTag = "__ARC_TMUX_RUN_END__"
	}
	inner := fmt.Sprintf("printf \"\\n%s\\n\"; ( %s ); status=$?;", startTag, command)
	if includeExit {
		if strings.TrimSpace(exitTag) == "" {
			exitTag = "__ARC_TMUX_EXIT:"
		}
		inner += fmt.Sprintf(" printf \"\\n%s%%d\\n\" \"$status\";", exitTag)
	}
	inner += fmt.Sprintf(" printf \"\\n%s\\n\"", endTag)
	return "sh -lc " + shellQuoteSingle(inner)
}

func shellQuoteSingle(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func extractRunWindow(output string, startTag string, endTag string, exitTag string, parseExit bool) (string, *int, bool, bool) {
	if startTag == "" || endTag == "" {
		return output, nil, false, false
	}
	hadTrailingNewline := strings.HasSuffix(output, "\n")
	lines := splitLines(output)
	startIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], startTag) {
			startIdx = i
			break
		}
	}
	if startIdx == -1 {
		return output, nil, false, false
	}
	endIdx := -1
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], endTag) {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		endIdx = len(lines)
	}
	segment := append([]string(nil), lines[startIdx+1:endIdx]...)
	var code *int
	found := false
	if parseExit {
		segment, code, found = extractExitFromLines(segment, exitTag)
	}
	clean := strings.Join(segment, "\n")
	if hadTrailingNewline {
		clean += "\n"
	}
	return clean, code, found, true
}

func extractExitFromLines(lines []string, tag string) ([]string, *int, bool) {
	if tag == "" {
		return lines, nil, false
	}
	for i := len(lines) - 1; i >= 0; i-- {
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
		return lines, &code, true
	}
	return lines, nil, false
}

func newRunID() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func combineRunErrors(waitErr error, exitPropagate bool, exitRequested bool, code *int, found bool) error {
	if waitErr != nil {
		return waitErr
	}
	if exitPropagate {
		if !found && exitRequested {
			return newCodedError(errCommandExit, "exit code not found", nil)
		}
		if found && code != nil && *code != 0 {
			return newCodedError(errCommandExit, fmt.Sprintf("command exited with %d", *code), nil)
		}
	}
	return nil
}
