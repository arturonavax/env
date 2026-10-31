package domain

import (
	"errors"
	"fmt"
)

// Canonical CLI exit codes matching Unix conventions and odd/specs/anx/cli-contract.md.
const (
	ExitSuccess           = 0
	ExitGeneralError      = 1
	ExitUsageError        = 2
	ExitPreflightFailed   = 3
	ExitConflictOrAudit   = 4
	ExitRollbackOrPartial = 5
	ExitSecurityViolation = 6
	ExitInterrupted       = 130
)

// ExitError wraps an error with a deterministic process exit code.
type ExitError struct {
	Code int
	Err  error
}

// NewExitError constructs an ExitError with the provided exit code and cause.
func NewExitError(code int, err error) *ExitError {
	return &ExitError{
		Code: code,
		Err:  err,
	}
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// ResolveExitCode maps any error to its canonical exit code.
func ResolveExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) && exitErr.Code > 0 {
		return exitErr.Code
	}
	return ExitGeneralError
}
