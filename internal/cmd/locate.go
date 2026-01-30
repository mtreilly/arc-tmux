// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourorg/arc-sdk/output"
	"github.com/yourorg/arc-tmux/pkg/tmux"
	"gopkg.in/yaml.v3"
)

func newLocateCmd() *cobra.Command {
	var outputOpts output.OutputOptions
	var query string
	var field string
	var useRegex bool
	var fuzzy bool
	var session string
	var window int

	cmd := &cobra.Command{
		Use:   "locate [query]",
		Short: "Locate panes by content",
		Long:  "Search pane metadata (command/title/path) and return matching panes.",
		Example: `  arc-tmux locate node
  arc-tmux locate --field title --regex "build|test"
  arc-tmux locate --field command --fuzzy ndsrv
  arc-tmux locate --session dev --field path /srv`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := outputOpts.Resolve(); err != nil {
				return err
			}

			q := strings.TrimSpace(query)
			if q == "" && len(args) > 0 {
				q = strings.Join(args, " ")
			}
			if q == "" {
				return fmt.Errorf("query is required")
			}

			field = strings.ToLower(strings.TrimSpace(field))
			if field == "" {
				field = "any"
			}
			if field != "any" && field != "command" && field != "title" && field != "path" {
				return fmt.Errorf("invalid field: %s", field)
			}
			if useRegex && fuzzy {
				return fmt.Errorf("use either --regex or --fuzzy, not both")
			}

			var re *regexp.Regexp
			if useRegex {
				var err error
				re, err = regexp.Compile(q)
				if err != nil {
					return fmt.Errorf("invalid regex: %w", err)
				}
			}

			resolvedSession, err := resolveSessionTarget(session)
			if err != nil {
				return err
			}
			session = resolvedSession

			panes, err := tmux.ListPanesDetailed()
			if err != nil {
				if err == tmux.ErrNoTmuxServer {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No tmux server is running.")
					return nil
				}
				return err
			}

			items := make([]paneSnapshot, 0, len(panes))
			for _, p := range panes {
				if session != "" && p.Session != session {
					continue
				}
				if window >= 0 && p.WindowIndex != window {
					continue
				}
				if !locateMatches(p, field, q, re, fuzzy) {
					continue
				}
				items = append(items, toPaneSnapshot(p))
			}

			sort.Slice(items, func(i, j int) bool {
				if items[i].Session != items[j].Session {
					return items[i].Session < items[j].Session
				}
				if items[i].WindowIndex != items[j].WindowIndex {
					return items[i].WindowIndex < items[j].WindowIndex
				}
				return items[i].PaneIndex < items[j].PaneIndex
			})

			out := cmd.OutOrStdout()
			switch {
			case outputOpts.Is(output.OutputJSON):
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(items)

			case outputOpts.Is(output.OutputYAML):
				enc := yaml.NewEncoder(out)
				defer func() { _ = enc.Close() }()
				return enc.Encode(items)

			case outputOpts.Is(output.OutputQuiet):
				for _, p := range items {
					_, _ = fmt.Fprintln(out, p.FormattedID)
				}
				return nil
			}

			if len(items) == 0 {
				_, _ = fmt.Fprintln(out, "No matching panes found.")
				return nil
			}

			_, _ = fmt.Fprintln(out, "Matching panes:")
			for _, p := range items {
				_, _ = fmt.Fprintf(out, "  %s  cmd=%s  title=%s  path=%s\n", p.FormattedID, p.Command, p.Title, p.Path)
			}
			return nil
		},
	}

	outputOpts.AddOutputFlags(cmd, output.OutputTable)
	cmd.Flags().StringVar(&query, "query", "", "Query string to match")
	cmd.Flags().StringVar(&field, "field", "any", "Field to search: any|command|title|path")
	cmd.Flags().BoolVar(&useRegex, "regex", false, "Interpret query as regex")
	cmd.Flags().BoolVar(&fuzzy, "fuzzy", false, "Use fuzzy matching instead of substring matching")
	cmd.Flags().StringVar(&session, "session", "", "Filter by session name or selector (@current|@managed)")
	cmd.Flags().IntVar(&window, "window", -1, "Filter by window index")
	return cmd
}

func locateMatches(p tmux.PaneDetails, field string, query string, re *regexp.Regexp, fuzzy bool) bool {
	var fields []string
	switch field {
	case "command":
		fields = []string{p.Command}
	case "title":
		fields = []string{p.Title}
	case "path":
		fields = []string{p.Path}
	default:
		fields = []string{p.Command, p.Title, p.Path}
	}
	for _, value := range fields {
		if matchesQuery(value, query, re, fuzzy) {
			return true
		}
	}
	return false
}

func matchesQuery(value string, query string, re *regexp.Regexp, fuzzy bool) bool {
	if re != nil {
		return re.MatchString(value)
	}
	if fuzzy {
		return fuzzyMatch(value, query)
	}
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}

func fuzzyMatch(value string, query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	v := strings.ToLower(value)
	qi := 0
	for _, r := range v {
		if qi >= len(q) {
			break
		}
		if r == rune(q[qi]) {
			qi++
		}
	}
	return qi == len(q)
}
