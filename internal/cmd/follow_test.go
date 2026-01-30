package cmd

import "testing"

func TestSplitLines(t *testing.T) {
	lines := splitLines("a\nb\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestDiffLinesOverlap(t *testing.T) {
	prev := []string{"a", "b"}
	curr := []string{"x", "a", "b", "c"}
	diff := diffLines(prev, curr)
	if len(diff) != 1 || diff[0] != "c" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}

func TestDiffLinesNoMatch(t *testing.T) {
	prev := []string{"a", "b"}
	curr := []string{"x"}
	diff := diffLines(prev, curr)
	if len(diff) != 1 || diff[0] != "x" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}

func TestTailLines(t *testing.T) {
	lines := []string{"a", "b", "c", "d"}
	res := tailLines(lines, 2)
	if len(res) != 2 || res[0] != "c" || res[1] != "d" {
		t.Fatalf("unexpected tail: %#v", res)
	}
}
