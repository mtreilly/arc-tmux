// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"gopkg.in/yaml.v3"
)

func newAliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Manage pane aliases",
		Long:  "Create, list, resolve, and delete pane aliases for quick targeting.",
		Example: `  arc-tmux alias set api --pane=@current
  arc-tmux alias list
  arc-tmux send "npm test" --pane=@api`,
	}

	cmd.AddCommand(
		newAliasListCmd(),
		newAliasSetCmd(),
		newAliasUnsetCmd(),
		newAliasResolveCmd(),
	)

	return cmd
}

func newAliasListCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var file string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List aliases",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			path := aliasPath(file)
			aliases, err := loadAliases(path)
			if err != nil {
				return err
			}
			entries := aliasesToEntries(aliases)
			out := cmd.OutOrStdout()

			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(entries)

			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				return enc.Encode(entries)

			case outputOpts.Is(output.OutputQuiet):
				for _, entry := range entries {
					fmt.Fprintln(out, entry.Name)
				}
				return nil
			}

			if len(entries) == 0 {
				fmt.Fprintln(out, "No aliases defined.")
				return nil
			}
			fmt.Fprintln(out, "Aliases:")
			for _, entry := range entries {
				fmt.Fprintf(out, "  %s => %s\n", entry.Name, entry.Target)
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&file, "file", "", "Alias file path (default: ARC_TMUX_ALIASES or config dir)")
	return cmd
}

func newAliasSetCmd() *cobra.Command {
	var file string
	var paneArg string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "set <name> [pane]",
		Short: "Set an alias",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			name, err := normalizeAliasName(args[0])
			if err != nil {
				return err
			}

			paneInput := paneArg
			if paneInput == "" && len(args) > 1 {
				paneInput = args[1]
			}
			if paneInput == "" {
				return fmt.Errorf("pane target is required")
			}

			target, err := resolvePaneTarget(paneInput)
			if err != nil {
				return err
			}
			if err := validatePaneTarget(target); err != nil {
				return err
			}

			path := aliasPath(file)
			aliases, err := loadAliases(path)
			if err != nil {
				return err
			}
			aliases[name] = target
			if err := saveAliases(path, aliases); err != nil {
				return err
			}
			entry := aliasEntry{Name: name, Target: target}
			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(entry)
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				return enc.Encode(entry)
			case outputOpts.Is(output.OutputQuiet):
				fmt.Fprintln(out, entry.Name)
				return nil
			}
			fmt.Fprintf(out, "Alias %s => %s\n", name, target)
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&file, "file", "", "Alias file path (default: ARC_TMUX_ALIASES or config dir)")
	cmd.Flags().StringVar(&paneArg, "pane", "", "Target tmux pane (e.g., fe:4.1, @current, @active)")
	return cmd
}

func newAliasUnsetCmd() *cobra.Command {
	var file string
	var outputOpts output.OutputOptions

	cmd := &cobra.Command{
		Use:   "unset <name>",
		Short: "Remove an alias",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			name, err := normalizeAliasName(args[0])
			if err != nil {
				return err
			}
			path := aliasPath(file)
			aliases, err := loadAliases(path)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if _, ok := aliases[name]; !ok {
				result := aliasUnsetResult{Name: name, Removed: false}
				return writeAliasUnset(out, outputOpts, result)
			}
			delete(aliases, name)
			if err := saveAliases(path, aliases); err != nil {
				return err
			}
			result := aliasUnsetResult{Name: name, Removed: true}
			return writeAliasUnset(out, outputOpts, result)
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&file, "file", "", "Alias file path (default: ARC_TMUX_ALIASES or config dir)")
	return cmd
}

func newAliasResolveCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var file string

	cmd := &cobra.Command{
		Use:   "resolve <name>",
		Short: "Resolve an alias to a pane id",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}
			name, err := normalizeAliasName(args[0])
			if err != nil {
				return err
			}
			path := aliasPath(file)
			aliases, err := loadAliases(path)
			if err != nil {
				return err
			}
			target, ok := aliases[name]
			if !ok {
				return fmt.Errorf("alias %s not found", name)
			}

			entry := aliasEntry{Name: name, Target: target}
			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(entry)
			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer enc.Close()
				return enc.Encode(entry)
			case outputOpts.Is(output.OutputQuiet):
				fmt.Fprintln(out, entry.Target)
				return nil
			}
			fmt.Fprintf(out, "%s => %s\n", entry.Name, entry.Target)
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&file, "file", "", "Alias file path (default: ARC_TMUX_ALIASES or config dir)")
	return cmd
}

func aliasPath(file string) string {
	if file != "" {
		return file
	}
	return defaultAliasFile()
}

type aliasUnsetResult struct {
	Name    string `json:"name" yaml:"name"`
	Removed bool   `json:"removed" yaml:"removed"`
}

func writeAliasUnset(out interface{ Write([]byte) (int, error) }, outputOpts output.OutputOptions, result aliasUnsetResult) error {
	switch {
	case outputOpts.Is(output.OutputJSON):
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	case outputOpts.Is(output.OutputYAML):
		enc := yaml.NewEncoder(out)
		defer enc.Close()
		return enc.Encode(result)
	case outputOpts.Is(output.OutputQuiet):
		if result.Removed {
			fmt.Fprintln(out, result.Name)
		}
		return nil
	}
	if result.Removed {
		fmt.Fprintf(out, "Alias %s removed.\n", result.Name)
		return nil
	}
	fmt.Fprintf(out, "Alias %s not found.\n", result.Name)
	return nil
}
