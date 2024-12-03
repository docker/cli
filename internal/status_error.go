package internal

import "strconv"

// StatusError reports an unsuccessful exit by a command.
// StatusCode must be non-zero.
// Cause may be nil if a generic error-message is desired.
type StatusError struct {
	// Cause is the underlying error.
	// It may be nil to generate a generic error message.
	Cause error
	// StatusCode is the exit status code.
	// It must be non-zero.
	StatusCode int
}

// Error formats the error for printing. If a custom Status is provided,
// it is returned as-is, otherwise it generates a generic error-message
// based on the StatusCode.
func (e StatusError) Error() string {
	if e.Cause == nil {
		return "exit status " + strconv.Itoa(e.StatusCode)
	}
	return e.Cause.Error()
}

// Unwrap returns the wrapped error.
//
// This allows StatusError to be checked with errors.Is.
func (e StatusError) Unwrap() error {
	return e.Cause
}

func (e StatusError) GetStatusCode() int {
	return e.StatusCode
}
