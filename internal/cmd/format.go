// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import "time"

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
