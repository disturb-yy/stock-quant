// Package apperror defines stable application error codes and preserves causes.
package apperror

import (
	"errors"
	"fmt"
)

// Code classifies failures without exposing infrastructure details to API clients.
type Code string

const (
	CodeInvalidArgument     Code = "INVALID_ARGUMENT"
	CodeDataIncomplete      Code = "DATA_INCOMPLETE"
	CodePermissionDenied    Code = "PERMISSION_DENIED"
	CodeRateLimited         Code = "RATE_LIMITED"
	CodeUpstreamUnavailable Code = "UPSTREAM_UNAVAILABLE"
	CodeStrategyFailed      Code = "STRATEGY_FAILED"
	CodeTimeout             Code = "TIMEOUT"
	CodeNotFound            Code = "NOT_FOUND"
	CodeConflict            Code = "CONFLICT"
	CodeDataUntrusted       Code = "DATA_UNTRUSTED"
	CodeInvalidFactorInput  Code = "INVALID_FACTOR_INPUT"
	CodeInternal            Code = "INTERNAL"
)

// Error carries a stable code and the original cause for diagnostics.
type Error struct {
	code  Code
	cause error
}

// New creates a classified error while retaining its underlying cause.
func New(code Code, cause error) *Error {
	return &Error{code: code, cause: cause}
}

// Error formats the code and cause for internal diagnostics.
func (err *Error) Error() string {
	if err == nil {
		return "<nil>"
	}
	if err.cause == nil {
		return string(err.code)
	}
	return fmt.Sprintf("%s: %v", err.code, err.cause)
}

// Unwrap exposes the original cause to errors.Is and errors.As.
func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

// Code returns the stable classification carried by this error.
func (err *Error) Code() Code {
	if err == nil {
		return ""
	}
	return err.code
}

// CodeOf returns an error's classification, defaulting unclassified failures to INTERNAL.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	var classified *Error
	if errors.As(err, &classified) {
		return classified.Code()
	}
	return CodeInternal
}
