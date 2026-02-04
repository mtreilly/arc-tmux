// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strings"
)

type envVar struct {
	Key   string
	Value string
}

func shellQuoteSingle(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func parseEnvVars(raw []string) ([]envVar, error) {
	vars := make([]envVar, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("invalid env value: empty")
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env %q; expected KEY=VAL", item)
		}
		key := strings.TrimSpace(parts[0])
		if key == "" {
			return nil, fmt.Errorf("invalid env %q; key is required", item)
		}
		if !isValidEnvKey(key) {
			return nil, fmt.Errorf("invalid env key %q", key)
		}
		vars = append(vars, envVar{Key: key, Value: parts[1]})
	}
	return vars, nil
}

func isValidEnvKey(key string) bool {
	for i, r := range key {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r == '_':
		case r >= '0' && r <= '9':
			if i == 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func envAssignments(vars []envVar) string {
	if len(vars) == 0 {
		return ""
	}
	parts := make([]string, 0, len(vars))
	for _, v := range vars {
		parts = append(parts, v.Key+"="+shellQuoteSingle(v.Value))
	}
	return strings.Join(parts, " ")
}

func buildRunCommand(command string, cwd string, env []envVar) string {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		if cwd == "" && len(env) == 0 {
			return ""
		}
		trimmed = "exec \"${SHELL:-sh}\""
	}
	if cwd == "" && len(env) == 0 {
		return trimmed
	}
	combined := buildCommandWithEnv(trimmed, env)
	if cwd != "" {
		combined = "cd " + shellQuoteSingle(cwd) + " && " + combined
	}
	return "( " + combined + " )"
}

func buildCommandWithEnv(command string, env []envVar) string {
	if len(env) == 0 {
		return command
	}
	assignments := envAssignments(env)
	if strings.TrimSpace(command) == "" {
		return assignments + " exec \"${SHELL:-sh}\""
	}
	return assignments + " " + command
}
