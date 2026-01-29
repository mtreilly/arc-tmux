// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package tmux

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoTmuxServer    = errors.New("no tmux server running")
	ErrSessionNotFound = errors.New("tmux session not found")
)

// Pane represents a tmux pane with canonical identifiers.
type Pane struct {
	Session     string `json:"session"`
	WindowIndex int    `json:"window_index"`
	PaneIndex   int    `json:"pane_index"`
	Active      bool   `json:"active"`
	Command     string `json:"command"`
	Title       string `json:"title"`
}

// FormattedID returns session:window.pane
func (p Pane) FormattedID() string {
	return fmt.Sprintf("%s:%d.%d", p.Session, p.WindowIndex, p.PaneIndex)
}

// Window represents a tmux window.
type Window struct {
	Session     string `json:"session"`
	WindowIndex int    `json:"window_index"`
	Active      bool   `json:"active"`
	Name        string `json:"name"`
}

func ensureTmux() (string, error) {
	return exec.LookPath("tmux")
}

// InTmux reports whether running inside a tmux session.
func InTmux() bool { return os.Getenv("TMUX") != "" }

// ListPanes returns panes across all sessions.
func ListPanes() ([]Pane, error) {
	if _, err := ensureTmux(); err != nil {
		return nil, fmt.Errorf("tmux not found in PATH: %w", err)
	}
	format := strings.Join([]string{
		"#{session_name}",
		"#{window_index}",
		"#{pane_index}",
		"#{?pane_active,1,0}",
		"#{pane_current_command}",
		"#{pane_title}",
	}, "\t")
	cmd := exec.Command("tmux", "list-panes", "-a", "-F", format)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tmux list-panes: %v\n%s", err, out.String())
	}
	var panes []Pane
	s := bufio.NewScanner(&out)
	for s.Scan() {
		line := s.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 6 {
			continue
		}
		win, _ := strconv.Atoi(parts[1])
		pane, _ := strconv.Atoi(parts[2])
		act := parts[3] == "1"
		panes = append(panes, Pane{
			Session:     parts[0],
			WindowIndex: win,
			PaneIndex:   pane,
			Active:      act,
			Command:     parts[4],
			Title:       parts[5],
		})
	}
	return panes, s.Err()
}

// ListWindows lists windows for a session (or all if session=="").
func ListWindows(session string) ([]Window, error) {
	if _, err := ensureTmux(); err != nil {
		return nil, err
	}
	format := strings.Join([]string{
		"#{session_name}",
		"#{window_index}",
		"#{?window_active,1,0}",
		"#{window_name}",
	}, "\t")
	args := []string{"list-windows", "-F", format}
	if session != "" {
		args = append(args, "-t", session)
	}
	cmd := exec.Command("tmux", args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, wrapListWindowsError(err, errBuf.String())
	}
	var wins []Window
	s := bufio.NewScanner(&out)
	for s.Scan() {
		parts := strings.Split(s.Text(), "\t")
		if len(parts) < 4 {
			continue
		}
		idx, _ := strconv.Atoi(parts[1])
		wins = append(wins, Window{Session: parts[0], WindowIndex: idx, Active: parts[2] == "1", Name: parts[3]})
	}
	return wins, s.Err()
}

func wrapListWindowsError(runErr error, stderr string) error {
	msg := strings.TrimSpace(stderr)
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "no server running"):
		return ErrNoTmuxServer
	case strings.Contains(lower, "can't find session"), strings.Contains(lower, "no current session"):
		return ErrSessionNotFound
	default:
		if msg != "" {
			return fmt.Errorf("tmux list-windows: %s", msg)
		}
		return fmt.Errorf("tmux list-windows: %w", runErr)
	}
}

// ValidateTarget performs basic sanity checks on a target id.
func ValidateTarget(target string) error {
	if strings.Count(target, ":") != 1 || strings.Count(target, ".") != 1 {
		return errors.New("invalid pane id; expected session:window.pane")
	}
	return nil
}

// SendLiteral sends literal text to the pane; if enter is true, sends Enter with optional delay.
func SendLiteral(target string, text string, enter bool, delayEnter time.Duration) error {
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	if err := exec.Command("tmux", "send-keys", "-t", target, "-l", text).Run(); err != nil {
		return fmt.Errorf("tmux send-keys: %w", err)
	}
	if enter {
		if delayEnter > 0 {
			time.Sleep(delayEnter)
		}
		if err := exec.Command("tmux", "send-keys", "-t", target, "C-m").Run(); err != nil {
			return fmt.Errorf("tmux send-keys enter: %w", err)
		}
	}
	return nil
}

// Capture returns the visible content of a pane.
func Capture(target string, lines int) (string, error) {
	if _, err := ensureTmux(); err != nil {
		return "", fmt.Errorf("tmux not found in PATH: %w", err)
	}
	args := []string{"capture-pane", "-p", "-t", target}
	if lines > 0 {
		args = append(args, "-S", fmt.Sprintf("-%d", lines))
	}
	cmd := exec.Command("tmux", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux capture-pane: %w", err)
	}
	return out.String(), nil
}

// WaitIdle waits until pane output is stable for idleDur or timeout hits.
func WaitIdle(target string, idleDur time.Duration, timeout time.Duration) error {
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	poll := 300 * time.Millisecond
	deadline := time.Now().Add(timeout)
	var lastHash [20]byte
	lastChange := time.Now()
	for {
		if time.Now().After(deadline) {
			return errors.New("timeout waiting for idle")
		}
		s, err := Capture(target, 200)
		if err != nil {
			return err
		}
		h := sha1.Sum([]byte(s))
		if h != lastHash {
			lastHash = h
			lastChange = time.Now()
		} else {
			if time.Since(lastChange) >= idleDur {
				return nil
			}
		}
		time.Sleep(poll)
	}
}

// Interrupt sends Ctrl+C to the target pane.
func Interrupt(target string) error {
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	return exec.Command("tmux", "send-keys", "-t", target, "C-c").Run()
}

// Escape sends Escape key to the target pane.
func Escape(target string) error {
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	return exec.Command("tmux", "send-keys", "-t", target, "Escape").Run()
}

// Kill kills the target pane, guarded against self-kill.
func Kill(target string) error {
	self, _ := CurrentPaneID()
	if self != "" && self == strings.TrimSpace(target) {
		return errors.New("refusing to kill the current pane")
	}
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	return exec.Command("tmux", "kill-pane", "-t", target).Run()
}

// CurrentPaneID returns the current pane id in session:window.pane format.
func CurrentPaneID() (string, error) {
	if _, err := ensureTmux(); err != nil {
		return "", fmt.Errorf("tmux not found in PATH: %w", err)
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}:#{window_index}.#{pane_index}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux display-message: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// CurrentLocation returns session, window index, pane index, and formatted pane id.
func CurrentLocation() (string, int, int, string, error) {
	if _, err := ensureTmux(); err != nil {
		return "", 0, 0, "", fmt.Errorf("tmux not found in PATH: %w", err)
	}
	format := "#{session_name}\t#{window_index}\t#{pane_index}\t#{session_name}:#{window_index}.#{pane_index}"
	cmd := exec.Command("tmux", "display-message", "-p", format)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", 0, 0, "", fmt.Errorf("tmux display-message: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(out.String()), "\t")
	if len(parts) < 4 {
		return "", 0, 0, "", errors.New("failed to parse current location")
	}
	wi, _ := strconv.Atoi(parts[1])
	pi, _ := strconv.Atoi(parts[2])
	return parts[0], wi, pi, parts[3], nil
}

// EnsureSession ensures a session exists; if not, creates it detached.
func EnsureSession(name string) error {
	if _, err := ensureTmux(); err != nil {
		return err
	}
	if err := exec.Command("tmux", "has-session", "-t", name).Run(); err == nil {
		return nil
	}
	return exec.Command("tmux", "new-session", "-d", "-s", name).Run()
}

// Attach attaches to a session.
func Attach(name string) error {
	if _, err := ensureTmux(); err != nil {
		return err
	}
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Cleanup kills a session.
func Cleanup(name string) error {
	if _, err := ensureTmux(); err != nil {
		return err
	}
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

// Launch creates a new pane/window and runs cmd. Returns the new pane formatted id.
func Launch(managedSession string, cmdStr string, split string) (string, error) {
	if _, err := ensureTmux(); err != nil {
		return "", err
	}
	format := "#{session_name}:#{window_index}.#{pane_index}"
	if InTmux() {
		args := []string{"split-window", "-P", "-F", format}
		if split == "h" {
			args = append(args, "-h")
		}
		if split == "v" {
			args = append(args, "-v")
		}
		if strings.TrimSpace(cmdStr) != "" {
			args = append(args, cmdStr)
		}
		out, err := exec.Command("tmux", args...).Output()
		if err != nil {
			return "", fmt.Errorf("tmux split-window: %w", err)
		}
		return strings.TrimSpace(string(out)), nil
	}
	if managedSession == "" {
		managedSession = "arc-tmux"
	}
	if err := EnsureSession(managedSession); err != nil {
		return "", err
	}
	args := []string{"new-window", "-t", managedSession, "-P", "-F", format}
	if strings.TrimSpace(cmdStr) != "" {
		args = append(args, cmdStr)
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", fmt.Errorf("tmux new-window: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
