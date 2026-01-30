package cmd

import "testing"

func TestSplitLines(t *testing.T) {
	lines := splitLines("a\nb\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestDiffLinesOverlap(t *testing.T) {
	prev := []string{"a", "b", "c"}
	curr := []string{"a", "b", "c", "d"}
	diff := diffLines(prev, curr)
	if len(diff) != 1 || diff[0] != "d" {
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

func TestDiffLinesSuffixPrefix(t *testing.T) {
	prev := []string{"a", "b", "c"}
	curr := []string{"b", "c", "d"}
	diff := diffLines(prev, curr)
	if len(diff) != 1 || diff[0] != "d" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}

func TestDiffLinesByCount(t *testing.T) {
	prevCount := 2
	curr := []string{"a", "b", "c"}
	diff := diffLinesByCount(curr, &prevCount)
	if len(diff) != 1 || diff[0] != "c" {
		t.Fatalf("unexpected diff: %#v", diff)
	}
	if prevCount != 3 {
		t.Fatalf("unexpected count: %d", prevCount)
	}
}
