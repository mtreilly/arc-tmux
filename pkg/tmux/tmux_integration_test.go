package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestIntegrationSessionsAndPanes(t *testing.T) {
	if os.Getenv("ARC_TMUX_IT") != "1" {
		t.Skip("set ARC_TMUX_IT=1 to run integration tests")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	tmp, err := os.MkdirTemp("/tmp", "arc-tmux-it-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	setEnv(t, "TMUX_TMPDIR", tmp)
	setEnv(t, "TMUX", "")

	session := fmt.Sprintf("arc-tmux-it-%d", time.Now().UnixNano())
	if err := tmuxCmd(t, "new-session", "-d", "-s", session, "sleep 300"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	t.Cleanup(func() {
		_ = tmuxCmd(t, "kill-session", "-t", session)
	})

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.Name == session {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("session %s not found", session)
	}

	panes, err := ListPanesDetailed()
	if err != nil {
		t.Fatalf("ListPanesDetailed error: %v", err)
	}
	var target PaneDetails
	for _, p := range panes {
		if p.Session == session {
			target = p
			break
		}
	}
	if target.Session == "" {
		t.Fatalf("no panes found for session %s", session)
	}
	if target.PID == 0 {
		t.Fatalf("expected non-zero pid")
	}

	pane, err := PaneDetailsForTarget(fmt.Sprintf("%s:%d.%d", target.Session, target.WindowIndex, target.PaneIndex))
	if err != nil {
		t.Fatalf("PaneDetailsForTarget error: %v", err)
	}
	if pane.Session != session {
		t.Fatalf("unexpected pane session: %s", pane.Session)
	}
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("setenv %s: %v", key, err)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func tmuxCmd(t *testing.T, args ...string) error {
	t.Helper()
	cmd := exec.Command("tmux", args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux %v: %v (%s)", args, err, errBuf.String())
	}
	return nil
}
