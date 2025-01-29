package cli

import (
	"github.com/docker/cli/internal"
)

// StatusCodeError is a custom error type that reports an unsuccessful
// exit by a command.
//
// It is preferable to use this interface when seeking the status code
// of an error returned by the CLI.
//
//	var statusCodeError cli.StatusCodeError
//	err := someFunction()
//	errors.As(err, &statusCodeError)
//	fmt.Println(statusCodeError.GetStatusCode())
//
// Internal to the CLI can use the [internal.StatusError]
// implementation directly.
type StatusCodeError interface {
	error
	Unwrap() error

	// GetStatusCode returns the status code of the error.
	// The status code will never be 0.
	GetStatusCode() int
}

// StatusError implements [cli.StatusCodeError] and reports
// an unsuccessful exit by a command.
//
// StatusCode must be non-zero.
// Status/Cause may be empty/nil if a generic error-message is desired.
//
// Note: This error type is used by CLI plugins to report
// the exit code of a command and is thus discouraged from being
// modified.
//
// It is usually discouraged to use this error type directly as the
// [cli.StatusCodeError] interface should be used instead.
type StatusError struct {
	// Deprecated: StatusError.Status is deprecated and should not be
	// used. Instead, use StatusError.Cause.
	Status string

	// Cause is the underlying error that caused the failure. It may be
	// set to nil if a generic error-message is desired.
	Cause error

	// StatusCode is the exit code of the command. This field must
	// be non-zero.
	StatusCode int
}

// Error formats the error for printing. If a custom Status/Cause
// is provided, it is returned as-is, otherwise it generates a generic
// error-message based on the StatusCode.
func (e StatusError) Error() string {
	if e.Status != "" {
		return e.Status
	}
	return internal.StatusError{
		Cause:      e.Cause,
		StatusCode: e.StatusCode,
	}.Error()
}

func (e StatusError) Unwrap() error {
	return e.Cause
}

func (e StatusError) GetStatusCode() int {
	return e.StatusCode
}

// Pin the exported StatusCodeError interface to the
// [internal.StatusError] type.
// This is necessary to ensure that the internal.StatusError type does
// not break the compatibility of the exported interface.
var (
	_ StatusCodeError = internal.StatusError{}
	_ StatusCodeError = StatusError{}
)
