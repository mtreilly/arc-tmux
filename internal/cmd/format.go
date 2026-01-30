// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"strconv"
	"strings"
	"time"
)

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func formatRelative(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	delta := time.Since(t)
	if delta < 0 {
		delta = 0
	}
	return delta.Round(time.Second).String() + " ago"
}

func parseFormattedPaneID(fid string) (session string, window int, pane int) {
	session = ""
	window = 0
	pane = 0
	if fid == "" {
		return
	}
	parts := strings.SplitN(fid, ":", 2)
	if len(parts) != 2 {
		return
	}
	session = parts[0]
	rest := parts[1]
	sub := strings.SplitN(rest, ".", 2)
	if len(sub) != 2 {
		return
	}
	window, _ = strconv.Atoi(sub[0])
	pane, _ = strconv.Atoi(sub[1])
	return
}
