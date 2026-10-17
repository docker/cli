// Package hint attaches actionable user guidance to errors.
//
// A hint describes what the user can do about a failure ("Run
// 'docker rm foo' to remove it.") and is rendered separately from the
// error's message by the top-level error handler. It is not part of
// [error.Error], so substring matching on the error message stays
// stable as hints are added or reworded.
//
// Use [Wrap] to attach a hint at the call site, and [Of] (or
// [errors.As] against [Hinter]) to extract it for rendering.
package hint

import "errors"

// Hinter is implemented by errors that carry actionable user guidance.
type Hinter interface {
	Hint() string
}

type errWithHint struct {
	error
	hint string
}

func (e *errWithHint) Hint() string  { return e.hint }
func (e *errWithHint) Unwrap() error { return e.error }

// Wrap attaches actionable guidance to err. It returns nil if err is
// nil. The hint does not appear in the wrapped error's [error.Error]
// output; it is read out of the chain by the top-level renderer via
// [Of] (or [errors.As] against [Hinter]).
func Wrap(err error, hint string) error {
	if err == nil {
		return nil
	}
	return &errWithHint{error: err, hint: hint}
}

// Of returns the first hint in the error chain, or "" if none of the wrapped
// errors implement [Hinter].
func Of(err error) string {
	var h Hinter
	if errors.As(err, &h) {
		return h.Hint()
	}
	return ""
}
