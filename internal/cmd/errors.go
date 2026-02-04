// Copyright (c) 2025 Arc Engineering
// SPDX-License-Identifier: MIT

package cmd

import "fmt"

type codedError struct {
	Code    string
	Message string
	Cause   error
}

func (e *codedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *codedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func newCodedError(code string, message string, cause error) error {
	return &codedError{Code: code, Message: message, Cause: cause}
}

const (
	errPaneRequired      = "ERR_PANE_REQUIRED"
	errInvalidPane       = "ERR_INVALID_PANE"
	errUnknownSelector   = "ERR_UNKNOWN_SELECTOR"
	errNoActivePane      = "ERR_NO_ACTIVE_PANE"
	errNoCurrentPane     = "ERR_NO_CURRENT_PANE"
	errNoTmuxClient      = "ERR_NOT_IN_TMUX"
	errSignalUnsupported = "ERR_SIGNAL_UNSUPPORTED"
	errCommandExit       = "ERR_COMMAND_EXIT"
	errInvalidEnv        = "ERR_INVALID_ENV"
)
