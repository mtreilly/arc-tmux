package cmd

import (
	"os"
	"testing"
)

func TestResolveSessionTargetManaged(t *testing.T) {
	old := os.Getenv("ARC_TMUX_SESSION")
	_ = os.Setenv("ARC_TMUX_SESSION", "managed-test")
	t.Cleanup(func() { _ = os.Setenv("ARC_TMUX_SESSION", old) })

	resolved, err := resolveSessionTarget("@managed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "managed-test" {
		t.Fatalf("unexpected session: %s", resolved)
	}
}
