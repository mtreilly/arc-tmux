package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func TestCLIWorkflowIntegration(t *testing.T) {
	if os.Getenv("ARC_TMUX_IT") != "1" {
		t.Skip("set ARC_TMUX_IT=1 to run integration tests")
	}
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}

	tmp, err := os.MkdirTemp("/tmp", "arc-tmux-cli-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	setEnv(t, "TMUX_TMPDIR", tmp)
	setEnv(t, "TMUX", "")

	session := fmt.Sprintf("arc-tmux-cli-%d", time.Now().UnixNano())
	if err := tmux.EnsureSession(session); err != nil {
		t.Fatalf("EnsureSession error: %v", err)
	}
	defer func() { _ = tmux.Cleanup(session) }()

	paneID, err := tmux.Launch(session, "", "")
	if err != nil {
		t.Fatalf("Launch error: %v", err)
	}

	out, err := runCLI("send", "printf 'ok\\n'", "--pane="+paneID, "--delay-enter=0", "--output", "json")
	if err != nil {
		t.Fatalf("send error: %v", err)
	}
	var sendRes sendResult
	if err := json.Unmarshal([]byte(out), &sendRes); err != nil {
		t.Fatalf("send json decode error: %v", err)
	}
	if sendRes.PaneID != paneID {
		t.Fatalf("unexpected pane id: %s", sendRes.PaneID)
	}

	out, err = runCLI("capture", "--pane="+paneID, "--output", "json")
	if err != nil {
		t.Fatalf("capture error: %v", err)
	}
	var capRes captureResult
	if err := json.Unmarshal([]byte(out), &capRes); err != nil {
		t.Fatalf("capture json decode error: %v", err)
	}
	if !bytes.Contains([]byte(capRes.Output), []byte("ok")) {
		t.Fatalf("expected output to contain ok")
	}

	out, err = runCLI("run", "echo hi; exit 7", "--pane="+paneID, "--exit-code", "--exit-propagate", "--output", "json", "--timeout", "5", "--idle", "1")
	if err == nil {
		t.Fatalf("expected non-zero exit error")
	}
	var runRes runResult
	if err := json.Unmarshal([]byte(out), &runRes); err != nil {
		t.Fatalf("run json decode error: %v", err)
	}
	if runRes.ExitCode == nil || *runRes.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %#v", runRes.ExitCode)
	}

	out, err = runCLI("monitor", "--pane="+paneID, "--output", "json")
	if err != nil {
		t.Fatalf("monitor error: %v", err)
	}
	var monitorRes monitorSnapshot
	if err := json.Unmarshal([]byte(out), &monitorRes); err != nil {
		t.Fatalf("monitor json decode error: %v", err)
	}
	if monitorRes.PaneID != paneID {
		t.Fatalf("unexpected monitor pane id: %s", monitorRes.PaneID)
	}

	out, err = runCLI("signal", "--pane="+paneID, "--signal", "0", "--output", "json")
	if err != nil {
		t.Fatalf("signal error: %v", err)
	}
	var signalRes signalResult
	if err := json.Unmarshal([]byte(out), &signalRes); err != nil {
		t.Fatalf("signal json decode error: %v", err)
	}
	if signalRes.PaneID != paneID {
		t.Fatalf("unexpected signal pane id: %s", signalRes.PaneID)
	}

	out, err = runCLI("stop", "--pane="+paneID, "--kill=false", "--timeout", "5", "--idle", "1", "--output", "json")
	if err != nil {
		t.Fatalf("stop error: %v", err)
	}
	var stopRes stopResult
	if err := json.Unmarshal([]byte(out), &stopRes); err != nil {
		t.Fatalf("stop json decode error: %v", err)
	}
	if stopRes.PaneID != paneID {
		t.Fatalf("unexpected stop pane id: %s", stopRes.PaneID)
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

func runCLI(args ...string) (string, error) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	var errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}
