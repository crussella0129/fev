package core

import (
	"errors"
	"testing"
)

func TestClassifyError_Retryable(t *testing.T) {
	cases := []string{
		"rate limit exceeded",
		"connection refused",
		"request timeout",
		"server returned 503",
		"server returned 429",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			err := errors.New(msg)
			if ClassifyError(err) != ErrorRetryable {
				t.Errorf("expected %q to be retryable", msg)
			}
		})
	}
}

func TestClassifyError_Fatal(t *testing.T) {
	cases := []string{
		"invalid API key",
		"model not found",
		"permission denied",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			err := errors.New(msg)
			if ClassifyError(err) != ErrorFatal {
				t.Errorf("expected %q to be fatal", msg)
			}
		})
	}
}

func TestWorkspaceBoundaryError(t *testing.T) {
	err := &WorkspaceBoundaryError{Path: "/etc/passwd", Root: "/home/user/project"}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}
