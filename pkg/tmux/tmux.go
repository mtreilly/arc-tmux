// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

// Package tmux provides a small wrapper around the tmux CLI.
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
	// ErrNoTmuxServer indicates no tmux server is running.
	ErrNoTmuxServer = errors.New("no tmux server running")
	// ErrSessionNotFound indicates the requested tmux session does not exist.
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

// Session represents a tmux session.
type Session struct {
	Name       string    `json:"name"`
	Windows    int       `json:"windows"`
	Attached   int       `json:"attached"`
	CreatedAt  time.Time `json:"created_at"`
	ActivityAt time.Time `json:"activity_at"`
}

// PaneDetails represents a tmux pane with extended metadata.
type PaneDetails struct {
	Session      string    `json:"session"`
	WindowIndex  int       `json:"window_index"`
	WindowName   string    `json:"window_name"`
	WindowActive bool      `json:"window_active"`
	PaneIndex    int       `json:"pane_index"`
	PaneID       string    `json:"pane_id"`
	Active       bool      `json:"active"`
	Command      string    `json:"command"`
	Title        string    `json:"title"`
	Path         string    `json:"path"`
	PID          int       `json:"pid"`
	ActivityAt   time.Time `json:"activity_at"`
}

// ProcessInfo represents a process from ps output.
type ProcessInfo struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
}

// ProcessNode represents a process in a tree with depth.
type ProcessNode struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
	Depth   int    `json:"depth"`
}

func ensureTmux() (string, error) {
	return exec.LookPath("tmux")
}

// InTmux reports whether running inside a tmux session.
func InTmux() bool { return os.Getenv("TMUX") != "" }

// HasSession reports whether the named session exists.
func HasSession(name string) (bool, error) {
	if _, err := ensureTmux(); err != nil {
		return false, fmt.Errorf("tmux not found in PATH: %w", err)
	}
	target := exactSessionTarget(name)
	cmd := exec.Command("tmux", "has-session", "-t", target)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	msg := strings.TrimSpace(errBuf.String())
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "no server running"),
		strings.Contains(lower, "can't find session"):
		return false, nil
	case msg != "":
		return false, fmt.Errorf("tmux has-session: %s", msg)
	default:
		return false, fmt.Errorf("tmux has-session: %w", err)
	}
}

func exactSessionTarget(name string) string {
	if strings.HasPrefix(name, "=") {
		return name
	}
	return "=" + name
}

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
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, wrapListPanesError(err, errBuf.String())
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

func wrapListPanesError(runErr error, stderr string) error {
	msg := strings.TrimSpace(stderr)
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "no server running"):
		return ErrNoTmuxServer
	default:
		if msg != "" {
			return fmt.Errorf("tmux list-panes: %s", msg)
		}
		return fmt.Errorf("tmux list-panes: %w", runErr)
	}
}

func parseSessionsOutput(output string) ([]Session, error) {
	var sessions []Session
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue
		}
		windows, _ := strconv.Atoi(parts[1])
		attached, _ := strconv.Atoi(parts[2])
		created := parseEpoch(parts[3])
		activity := parseEpoch(parts[4])
		sessions = append(sessions, Session{
			Name:       parts[0],
			Windows:    windows,
			Attached:   attached,
			CreatedAt:  created,
			ActivityAt: activity,
		})
	}
	return sessions, scanner.Err()
}

func parsePaneDetailsOutput(output string) ([]PaneDetails, error) {
	var panes []PaneDetails
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 12 {
			continue
		}
		winIdx, _ := strconv.Atoi(parts[1])
		winActive := parts[3] == "1"
		paneIdx, _ := strconv.Atoi(parts[4])
		paneActive := parts[6] == "1"
		pid, _ := strconv.Atoi(parts[10])
		activity := parseEpoch(parts[11])
		panes = append(panes, PaneDetails{
			Session:      parts[0],
			WindowIndex:  winIdx,
			WindowName:   parts[2],
			WindowActive: winActive,
			PaneIndex:    paneIdx,
			PaneID:       parts[5],
			Active:       paneActive,
			Command:      parts[7],
			Title:        parts[8],
			Path:         parts[9],
			PID:          pid,
			ActivityAt:   activity,
		})
	}
	return panes, scanner.Err()
}

func parseEpoch(raw string) time.Time {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return time.Time{}
	}
	if secs <= 0 {
		return time.Time{}
	}
	return time.Unix(secs, 0)
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

// ListSessions lists tmux sessions.
func ListSessions() ([]Session, error) {
	if _, err := ensureTmux(); err != nil {
		return nil, err
	}
	format := strings.Join([]string{
		"#{session_name}",
		"#{session_windows}",
		"#{session_attached}",
		"#{session_created}",
		"#{session_activity}",
	}, "\t")
	cmd := exec.Command("tmux", "list-sessions", "-F", format)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, wrapListSessionsError(err, errBuf.String())
	}
	return parseSessionsOutput(out.String())
}

func wrapListSessionsError(runErr error, stderr string) error {
	msg := strings.TrimSpace(stderr)
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "no server running"):
		return ErrNoTmuxServer
	default:
		if msg != "" {
			return fmt.Errorf("tmux list-sessions: %s", msg)
		}
		return fmt.Errorf("tmux list-sessions: %w", runErr)
	}
}

// ListPanesDetailed returns panes across all sessions with extended metadata.
func ListPanesDetailed() ([]PaneDetails, error) {
	if _, err := ensureTmux(); err != nil {
		return nil, fmt.Errorf("tmux not found in PATH: %w", err)
	}
	format := strings.Join([]string{
		"#{session_name}",
		"#{window_index}",
		"#{window_name}",
		"#{?window_active,1,0}",
		"#{pane_index}",
		"#{pane_id}",
		"#{?pane_active,1,0}",
		"#{pane_current_command}",
		"#{pane_title}",
		"#{pane_current_path}",
		"#{pane_pid}",
		"#{pane_activity}",
	}, "\t")
	cmd := exec.Command("tmux", "list-panes", "-a", "-F", format)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, wrapListPanesError(err, errBuf.String())
	}
	return parsePaneDetailsOutput(out.String())
}

// PaneDetailsForTarget returns extended metadata for a specific pane.
func PaneDetailsForTarget(target string) (PaneDetails, error) {
	if _, err := ensureTmux(); err != nil {
		return PaneDetails{}, fmt.Errorf("tmux not found in PATH: %w", err)
	}
	format := strings.Join([]string{
		"#{session_name}",
		"#{window_index}",
		"#{window_name}",
		"#{?window_active,1,0}",
		"#{pane_index}",
		"#{pane_id}",
		"#{?pane_active,1,0}",
		"#{pane_current_command}",
		"#{pane_title}",
		"#{pane_current_path}",
		"#{pane_pid}",
		"#{pane_activity}",
	}, "\t")
	cmd := exec.Command("tmux", "display-message", "-p", "-t", target, format)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return PaneDetails{}, fmt.Errorf("tmux display-message: %w", err)
	}
	panes, err := parsePaneDetailsOutput(out.String())
	if err != nil {
		return PaneDetails{}, err
	}
	if len(panes) == 0 {
		return PaneDetails{}, errors.New("no pane details returned")
	}
	return panes[0], nil
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

// SendKeys sends tmux key names to the pane (e.g., C-x, Enter, Down).
func SendKeys(target string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	args := append([]string{"send-keys", "-t", target}, keys...)
	if err := exec.Command("tmux", args...).Run(); err != nil {
		return fmt.Errorf("tmux send-keys: %w", err)
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

// CaptureJoined returns the visible content of a pane, joining wrapped lines.
func CaptureJoined(target string, lines int) (string, error) {
	if _, err := ensureTmux(); err != nil {
		return "", fmt.Errorf("tmux not found in PATH: %w", err)
	}
	args := []string{"capture-pane", "-p", "-J", "-t", target}
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

// PaneActivity returns the last activity time for a pane.
func PaneActivity(target string) (time.Time, error) {
	if _, err := ensureTmux(); err != nil {
		return time.Time{}, fmt.Errorf("tmux not found in PATH: %w", err)
	}
	cmd := exec.Command("tmux", "display-message", "-p", "-t", target, "#{pane_activity}")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return time.Time{}, fmt.Errorf("tmux display-message: %w", err)
	}
	raw := strings.TrimSpace(out.String())
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("tmux pane_activity parse: %w", err)
	}
	return time.Unix(secs, 0), nil
}

// ProcessTree returns the process tree rooted at pid, including the root.
func ProcessTree(pid int) ([]ProcessNode, error) {
	if pid <= 0 {
		return nil, errors.New("invalid pid")
	}
	procs, err := listProcesses()
	if err != nil {
		return nil, err
	}
	nodes := buildProcessTree(pid, procs)
	if len(nodes) == 0 {
		return nil, fmt.Errorf("pid %d not found", pid)
	}
	return nodes, nil
}

func listProcesses() ([]ProcessInfo, error) {
	cmd := exec.Command("ps", "-o", "pid=,ppid=,command=", "-A")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}
	return parseProcessList(out.String())
}

func parseProcessList(output string) ([]ProcessInfo, error) {
	var procs []ProcessInfo
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		ppid, _ := strconv.Atoi(fields[1])
		cmd := strings.Join(fields[2:], " ")
		procs = append(procs, ProcessInfo{
			PID:     pid,
			PPID:    ppid,
			Command: cmd,
		})
	}
	return procs, scanner.Err()
}

func buildProcessTree(rootPID int, procs []ProcessInfo) []ProcessNode {
	byPID := make(map[int]ProcessInfo, len(procs))
	children := make(map[int][]ProcessInfo)
	for _, p := range procs {
		byPID[p.PID] = p
		children[p.PPID] = append(children[p.PPID], p)
	}
	if _, ok := byPID[rootPID]; !ok {
		return nil
	}
	var nodes []ProcessNode
	var walk func(pid int, depth int)
	walk = func(pid int, depth int) {
		p := byPID[pid]
		nodes = append(nodes, ProcessNode{
			PID:     p.PID,
			PPID:    p.PPID,
			Command: p.Command,
			Depth:   depth,
		})
		for _, child := range children[pid] {
			if child.PID == pid {
				continue
			}
			walk(child.PID, depth+1)
		}
	}
	walk(rootPID, 0)
	return nodes
}

// WaitIdle waits until pane output is stable for idleDur or timeout hits.
func WaitIdle(target string, idleDur time.Duration, timeout time.Duration) error {
	if _, err := ensureTmux(); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	poll := 300 * time.Millisecond
	deadline := time.Now().Add(timeout)
	if lastActivity, err := PaneActivity(target); err == nil {
		for {
			if time.Now().After(deadline) {
				return errors.New("timeout waiting for idle")
			}
			current, err := PaneActivity(target)
			if err != nil {
				break
			}
			if current.After(lastActivity) {
				lastActivity = current
			}
			if time.Since(lastActivity) >= idleDur {
				return nil
			}
			time.Sleep(poll)
		}
	}
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
	if exists, err := HasSession(name); err != nil {
		return err
	} else if exists {
		return nil
	}
	if err := exec.Command("tmux", "new-session", "-d", "-s", name).Run(); err != nil {
		return err
	}
	if strings.HasPrefix(name, "arc-") {
		if err := ApplyAgentSessionStyle(name, DefaultAgentSessionMeta()); err != nil {
			return err
		}
	}
	return nil
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

func shellCommand(cmdStr string) []string {
	if strings.TrimSpace(cmdStr) == "" {
		return nil
	}
	return []string{"sh", "-lc", cmdStr}
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
		if shellArgs := shellCommand(cmdStr); len(shellArgs) > 0 {
			args = append(args, shellArgs...)
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
	if shellArgs := shellCommand(cmdStr); len(shellArgs) > 0 {
		args = append(args, shellArgs...)
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", fmt.Errorf("tmux new-window: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
