package cli

import (
	"strconv"
)

// StatusError reports an unsuccessful exit by a command.
type StatusError struct {
	Cause      error
	Status     string
	StatusCode int
}

// Error formats the error for printing. If a custom Status is provided,
// it is returned as-is, otherwise it generates a generic error-message
// based on the StatusCode.
func (e StatusError) Error() string {
	if e.Status != "" {
		return e.Status
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "exit status " + strconv.Itoa(e.StatusCode)
}

func (e StatusError) Unwrap() error {
	return e.Cause
}
