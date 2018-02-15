package errors

import (
	"fmt"
	"net/http"
)

// For now, this is used when the proxy directly replies to the client
// with an error, whithout even proxying the request.

// HTTPError is an error that can be used as a response to an HTTP request.
// It contains an error message and an HTTP status code.
type HTTPError struct {
	HttpStatusCode int
	ErrorMessage   string
}

// NewHTTPError returns a newly created HTTPError containing
// the http status code and error message provided
func NewHTTPError(httpCode int, errorMsg string) *HTTPError {
	return &HTTPError{
		HttpStatusCode: httpCode,
		ErrorMessage:   errorMsg,
	}
}

// Error returns a string representing the error
func (h *HTTPError) Error() string {
	return fmt.Sprintf("%v: %s", h.HttpStatusCode, h.ErrorMessage)
}

// Write writes the error in a http.ResponseWriter
func (h *HTTPError) Write(w http.ResponseWriter) {
	http.Error(w, h.ErrorMessage, h.HttpStatusCode)
}
