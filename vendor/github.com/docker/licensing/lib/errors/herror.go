package errors

import "net/http"

// HTTPStatus is a convenience checker for a (possibly wrapped) HTTPStatus
// interface error. If err doesn't support the HTTPStatus interface, it will
// default to either StatusOK or StatusInternalServerError as appropriate.
func HTTPStatus(err error) (status int, ok bool) {
	type httperror interface {
		HTTPStatus() int
	}

	_, _, cause := Cause(err)
	he, ok := cause.(httperror)
	if ok {
		return he.HTTPStatus(), true
	}

	if err == nil {
		return http.StatusOK, false
	}
	return http.StatusInternalServerError, false
}

// HTTPError provides an easy way to specify the http status code equivalent
// for an error at error creation time, as well as inheriting the useful
// features from Base (call stacks, structured text & data).
type HTTPError struct {
	*Base
	Status int
}

// HTTPStatus returns the appropriate http status code for this error.
func (e *HTTPError) HTTPStatus() int {
	if e == nil {
		return http.StatusOK
	}
	if e.Status == 0 {
		return http.StatusInternalServerError
	}
	return e.Status
}

// With allows additional structured data fields to be added to this HTTPError.
func (e *HTTPError) With(fields Fields) *HTTPError {
	e.AddFields(fields)
	return e
}

// WithStatus allows setting the http status.
func (e *HTTPError) WithStatus(status int) *HTTPError {
	e.Status = status
	return e
}

// NewHTTPError constructs an error with the given http status.
func NewHTTPError(status int, text string) *HTTPError {
	return newHTTPErrorWithDepth(status, text)
}

// newHTTPErrorWithDepth constructs an error with the given http status and
// depth
func newHTTPErrorWithDepth(status int, text string) *HTTPError {
	return &HTTPError{
		Status: status,
		Base:   NewBase(2, text),
	}
}

// NotFound is returned when a resource was not found.
func NotFound(fields Fields, text string) *HTTPError {
	return newHTTPErrorWithDepth(http.StatusNotFound, text).With(fields)
}

// BadRequest is returned when a request did not pass validation, or is
// in appropriate for the state of the resource it would affect.
func BadRequest(fields Fields, text string) *HTTPError {
	return newHTTPErrorWithDepth(http.StatusBadRequest, text).With(fields)
}

// Conflict is returned when request could not be completed due to a conflict with
// the current state of the resource.
func Conflict(fields Fields, text string) *HTTPError {
	return newHTTPErrorWithDepth(http.StatusConflict, text).With(fields)
}

// PaymentRequired is returned if the requested resource must be purchased.
func PaymentRequired(fields Fields, text string) *HTTPError {
	return newHTTPErrorWithDepth(http.StatusPaymentRequired, text).With(fields)
}

// Forbidden is returned if the requesting user does not have the required
// permissions for a request.
func Forbidden(fields Fields, text string) *HTTPError {
	return newHTTPErrorWithDepth(http.StatusForbidden, text).With(fields)
}

// InternalError should only be returned if no other specific error applies.
func InternalError(fields Fields, text string) *HTTPError {
	return newHTTPErrorWithDepth(http.StatusInternalServerError, text).With(fields)
}
