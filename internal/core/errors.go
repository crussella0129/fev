package core

import (
	"fmt"
	"strings"
)

// ErrorClass categorizes errors for retry decisions.
type ErrorClass int

const (
	ErrorRetryable ErrorClass = iota
	ErrorFatal
)

// ClassifyError determines if an error is retryable or fatal.
func ClassifyError(err error) ErrorClass {
	msg := strings.ToLower(err.Error())
	retryable := []string{"rate limit", "timeout", "connection refused", "503", "429", "temporary"}
	for _, pattern := range retryable {
		if strings.Contains(msg, pattern) {
			return ErrorRetryable
		}
	}
	return ErrorFatal
}

// WorkspaceBoundaryError is returned when a path escapes the workspace root.
type WorkspaceBoundaryError struct {
	Path string
	Root string
}

func (e *WorkspaceBoundaryError) Error() string {
	return fmt.Sprintf("path %q is outside workspace root %q", e.Path, e.Root)
}
