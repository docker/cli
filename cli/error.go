package cli

import (
	"github.com/docker/cli/internal"
)

// StatusError reports an unsuccessful exit by a command.
type StatusError interface {
	Error() string
	Unwrap() error

	// GetStatusCode returns the status code of the error.
	// The status code will never be 0.
	GetStatusCode() int
}

// Pin the exported StatusError interface to the internal.StatusError type.
// This is necessary to ensure that the internal.StatusError type does
// not break the compatibility of the exported interface.
var _ StatusError = internal.StatusError{}
