package cmd

import (
	"strings"
	"testing"
)

func TestParseEnvVars(t *testing.T) {
	vars, err := parseEnvVars([]string{"FOO=bar", "HELLO=world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vars) != 2 {
		t.Fatalf("expected 2 vars, got %d", len(vars))
	}
	if vars[0].Key != "FOO" || vars[0].Value != "bar" {
		t.Fatalf("unexpected first var: %+v", vars[0])
	}
}

func TestParseEnvVarsInvalid(t *testing.T) {
	if _, err := parseEnvVars([]string{"=oops"}); err == nil {
		t.Fatal("expected error for invalid env")
	}
}

func TestBuildRunCommandCwdOnly(t *testing.T) {
	cmd := buildRunCommand("", "/tmp/project", nil)
	if cmd == "" {
		t.Fatal("expected command for cwd-only")
	}
	if !strings.Contains(cmd, "cd '/tmp/project'") {
		t.Fatalf("expected cd in command, got: %s", cmd)
	}
	if !strings.Contains(cmd, "exec \"${SHELL:-sh}\"") {
		t.Fatalf("expected exec shell in command, got: %s", cmd)
	}
}

func TestBuildRunCommandEnvOnly(t *testing.T) {
	cmd := buildRunCommand("", "", []envVar{{Key: "FOO", Value: "bar"}})
	if cmd == "" {
		t.Fatal("expected command for env-only")
	}
	if !strings.Contains(cmd, "FOO='bar'") {
		t.Fatalf("expected env assignment, got: %s", cmd)
	}
	if !strings.Contains(cmd, "exec \"${SHELL:-sh}\"") {
		t.Fatalf("expected exec shell in command, got: %s", cmd)
	}
}

func TestBuildRunCommandWithCommand(t *testing.T) {
	cmd := buildRunCommand("echo hi", "/srv", []envVar{{Key: "FOO", Value: "bar"}})
	if !strings.Contains(cmd, "cd '/srv' && FOO='bar' echo hi") {
		t.Fatalf("unexpected command: %s", cmd)
	}
}
