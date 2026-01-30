package tmux

import "testing"

func TestParseSessionsOutput(t *testing.T) {
	input := "dev\t3\t1\t1700000000\t1700000100\n"
	sessions, err := parseSessionsOutput(input)
	if err != nil {
		t.Fatalf("parseSessionsOutput error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.Name != "dev" || s.Windows != 3 || s.Attached != 1 {
		t.Fatalf("unexpected session fields: %+v", s)
	}
	if s.CreatedAt.Unix() != 1700000000 || s.ActivityAt.Unix() != 1700000100 {
		t.Fatalf("unexpected timestamps: created=%d activity=%d", s.CreatedAt.Unix(), s.ActivityAt.Unix())
	}
}

func TestParsePaneDetailsOutput(t *testing.T) {
	input := "dev\t2\tapi\t1\t0\t%5\t1\tbash\tbuild\t/Users/me\t1234\t1700000200\n"
	panes, err := parsePaneDetailsOutput(input)
	if err != nil {
		t.Fatalf("parsePaneDetailsOutput error: %v", err)
	}
	if len(panes) != 1 {
		t.Fatalf("expected 1 pane, got %d", len(panes))
	}
	p := panes[0]
	if p.Session != "dev" || p.WindowIndex != 2 || p.PaneIndex != 0 {
		t.Fatalf("unexpected pane identity: %+v", p)
	}
	if p.WindowName != "api" || !p.WindowActive || !p.Active {
		t.Fatalf("unexpected active flags: %+v", p)
	}
	if p.Command != "bash" || p.Title != "build" || p.Path != "/Users/me" {
		t.Fatalf("unexpected pane metadata: %+v", p)
	}
	if p.PID != 1234 || p.ActivityAt.Unix() != 1700000200 {
		t.Fatalf("unexpected pid/activity: %+v", p)
	}
}

func TestParseProcessList(t *testing.T) {
	input := "123 1 /bin/bash -l\n456 123 node server.js\n"
	procs, err := parseProcessList(input)
	if err != nil {
		t.Fatalf("parseProcessList error: %v", err)
	}
	if len(procs) != 2 {
		t.Fatalf("expected 2 procs, got %d", len(procs))
	}
	if procs[0].PID != 123 || procs[0].PPID != 1 || procs[0].Command != "/bin/bash -l" {
		t.Fatalf("unexpected proc[0]: %+v", procs[0])
	}
	if procs[1].PID != 456 || procs[1].PPID != 123 || procs[1].Command != "node server.js" {
		t.Fatalf("unexpected proc[1]: %+v", procs[1])
	}
}

func TestBuildProcessTree(t *testing.T) {
	procs := []ProcessInfo{
		{PID: 1, PPID: 0, Command: "launchd"},
		{PID: 10, PPID: 1, Command: "bash"},
		{PID: 11, PPID: 10, Command: "node server.js"},
		{PID: 12, PPID: 10, Command: "grep"},
	}
	nodes := buildProcessTree(10, procs)
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[0].PID != 10 || nodes[0].Depth != 0 {
		t.Fatalf("unexpected root node: %+v", nodes[0])
	}
	if nodes[1].Depth != 1 || nodes[2].Depth != 1 {
		t.Fatalf("unexpected child depth: %+v", nodes)
	}
}
