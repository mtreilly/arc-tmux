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
	cmd := wrapCommandForRun("echo hi", "__START__", "__END__", "__TAG__:", true)
	if cmd == "" || cmd[:6] != "sh -lc" {
		t.Fatalf("unexpected wrapped command: %s", cmd)
	}
	if !strings.Contains(cmd, "__START__") || !strings.Contains(cmd, "__END__") {
		t.Fatalf("expected markers in wrapped command: %s", cmd)
	}
}

func TestExtractRunWindow(t *testing.T) {
	output := "noise\n__START__\nline1\n__EXIT__:7\n__END__\n"
	clean, code, found, ok := extractRunWindow(output, "__START__", "__END__", "__EXIT__:", true)
	if !ok || !found || code == nil || *code != 7 {
		t.Fatalf("expected exit code 7, got %v", code)
	}
	if clean != "line1\n" {
		t.Fatalf("unexpected clean output: %q", clean)
	}
}
