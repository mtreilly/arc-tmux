package cmd

import (
	"regexp"
	"testing"

	"github.com/yourorg/arc-tmux/pkg/tmux"
)

func TestMatchesQuerySubstring(t *testing.T) {
	if !matchesQuery("node server", "NODE", nil, false) {
		t.Fatalf("expected case-insensitive substring match")
	}
}

func TestMatchesQueryRegex(t *testing.T) {
	re := regexp.MustCompile("node|python")
	if !matchesQuery("python app", "", re, false) {
		t.Fatalf("expected regex match")
	}
}

func TestLocateMatchesField(t *testing.T) {
	pane := tmux.PaneDetails{Command: "bash", Title: "build", Path: "/srv/api"}
	if !locateMatches(pane, "title", "build", nil, false) {
		t.Fatalf("expected title match")
	}
	if locateMatches(pane, "command", "node", nil, false) {
		t.Fatalf("did not expect command match")
	}
}

func TestFuzzyMatch(t *testing.T) {
	if !fuzzyMatch("node server", "ns") {
		t.Fatalf("expected fuzzy match")
	}
	if fuzzyMatch("node server", "zz") {
		t.Fatalf("did not expect fuzzy match")
	}
}
