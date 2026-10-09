package apperror

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorPreservesCodeAndCause(t *testing.T) {
	cause := errors.New("provider unavailable")
	err := fmt.Errorf("load snapshot: %w", New(CodeUpstreamUnavailable, cause))

	if got := CodeOf(err); got != CodeUpstreamUnavailable {
		t.Fatalf("CodeOf() = %q, want %q", got, CodeUpstreamUnavailable)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is(%v, cause) = false", err)
	}
}

func TestCodeOfUnknownErrorDefaultsToInternal(t *testing.T) {
	if got := CodeOf(errors.New("unexpected")); got != CodeInternal {
		t.Fatalf("CodeOf() = %q, want %q", got, CodeInternal)
	}
}

func TestErrorFormatsCodeAndCause(t *testing.T) {
	err := New(CodeInvalidArgument, errors.New("bad date"))
	if got, want := err.Error(), "INVALID_ARGUMENT: bad date"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
