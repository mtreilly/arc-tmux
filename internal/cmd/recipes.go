// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"gopkg.in/yaml.v3"
)

type recipe struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Command     string `json:"command" yaml:"command"`
}

func newRecipesCmd() *cobra.Command {
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "recipes",
		Short: "Common workflows",
		Long:  "Show common arc-tmux workflows for agents and developers.",
		Example: `  arc-tmux recipes
  arc-tmux recipes --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			recipes := defaultRecipes()
			sort.Slice(recipes, func(i, j int) bool { return recipes[i].Name < recipes[j].Name })
			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(recipes)
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(recipes)
			case outputOpts.Is(output.OutputQuiet):
				for _, r := range recipes {
					_, _ = fmt.Fprintln(out, r.Name)
				}
				return nil
			}

			if len(recipes) == 0 {
				_, _ = fmt.Fprintln(out, "No recipes available.")
				return nil
			}
			_, _ = fmt.Fprintln(out, "Recipes:")
			for _, r := range recipes {
				_, _ = fmt.Fprintf(out, "  %s\n    %s\n    %s\n", r.Name, r.Description, r.Command)
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	return cmd
}

func defaultRecipes() []recipe {
	return []recipe{
		{
			Name:        "run-and-capture-json",
			Description: "Run a command, wait idle, capture output and exit code in JSON.",
			Command:     "arc-tmux run \"npm test\" --pane=@current --exit-code --exit-propagate --output json",
		},
		{
			Name:        "follow-live-output",
			Description: "Stream new output from a pane.",
			Command:     "arc-tmux follow --pane=@current --lines 200",
		},
		{
			Name:        "monitor-idle-hash",
			Description: "Check if a pane is idle and get an output hash.",
			Command:     "arc-tmux monitor --pane=@current --idle 5 --lines 200 --output json",
		},
		{
			Name:        "graceful-stop",
			Description: "Send Ctrl+C and kill on timeout.",
			Command:     "arc-tmux stop --pane=@current --timeout 20 --idle 3",
		},
		{
			Name:        "locate-by-path",
			Description: "Find panes by cwd path substring.",
			Command:     "arc-tmux locate --field path /srv --output json",
		},
		{
			Name:        "alias-current-pane",
			Description: "Create an alias for the current pane.",
			Command:     "arc-tmux alias set api --pane=@current",
		},
	}
}
