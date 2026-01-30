package cmd

import (
	"strings"
	"testing"
)

func TestShellQuoteSingle(t *testing.T) {
	quoted := shellQuoteSingle("a'b")
	if quoted != "'a'\"'\"'b'" {
		t.Fatalf("unexpected quote: %s", quoted)
	}
}

func TestWrapCommandForExit(t *testing.T) {
	cmd := wrapCommandForExit("echo hi", "__TAG__:")
	if cmd == "" || cmd[:6] != "sh -lc" {
		t.Fatalf("unexpected wrapped command: %s", cmd)
	}
	if !strings.Contains(cmd, "__TAG__:") {
		t.Fatalf("expected tag in wrapped command: %s", cmd)
	}
}

func TestExtractExitCode(t *testing.T) {
	output := "line1\n__ARC_TMUX_EXIT:7\n"
	clean, code, found := extractExitCode(output, "__ARC_TMUX_EXIT:")
	if !found || code == nil || *code != 7 {
		t.Fatalf("expected exit code 7, got %v", code)
	}
	if clean != "line1\n" {
		t.Fatalf("unexpected clean output: %q", clean)
	}
}
