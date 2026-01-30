// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

// Package main is the entrypoint for arc-tmux.
package main

import (
	"fmt"
	"os"

	"github.com/yourorg/arc-tmux/internal/cmd"
)

func main() {
	root := cmd.NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
