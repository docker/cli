// Copyright 2026 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rpc

import (
	"errors"
	"net/http"
)

// Code is a Connect error code: https://connectrpc.com/docs/protocol#error-codes
type Code string

const (
	CodeCanceled           Code = "canceled"
	CodeUnknown            Code = "unknown"
	CodeInvalidArgument    Code = "invalid_argument"
	CodeDeadlineExceeded   Code = "deadline_exceeded"
	CodeNotFound           Code = "not_found"
	CodeAlreadyExists      Code = "already_exists"
	CodePermissionDenied   Code = "permission_denied"
	CodeResourceExhausted  Code = "resource_exhausted"
	CodeFailedPrecondition Code = "failed_precondition"
	CodeAborted            Code = "aborted"
	CodeOutOfRange         Code = "out_of_range"
	CodeUnimplemented      Code = "unimplemented"
	CodeInternal           Code = "internal"
	CodeUnavailable        Code = "unavailable"
	CodeDataLoss           Code = "data_loss"
	CodeUnauthenticated    Code = "unauthenticated"
)

func (c Code) valid() bool {
	switch c {
	case CodeCanceled, CodeUnknown, CodeInvalidArgument, CodeDeadlineExceeded,
		CodeNotFound, CodeAlreadyExists, CodePermissionDenied, CodeResourceExhausted,
		CodeFailedPrecondition, CodeAborted, CodeOutOfRange, CodeUnimplemented,
		CodeInternal, CodeUnavailable, CodeDataLoss, CodeUnauthenticated:
		return true
	default:
		return false
	}
}

// Error is a failed call. A handler returns one to choose the code its caller
// sees. A [Client] returns one for every call that fails.
type Error struct {
	code Code
	err  error
}

// NewError annotates err with code.
func NewError(code Code, err error) *Error {
	return &Error{code: code, err: err}
}

func (e *Error) Code() Code { return e.code }

// Message returns the text of the underlying error, without the code.
func (e *Error) Message() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *Error) Error() string {
	if msg := e.Message(); msg != "" {
		return string(e.code) + ": " + msg
	}
	return string(e.code)
}

func (e *Error) Unwrap() error { return e.err }

// CodeOf returns the code of the first [Error] in err's chain, and
// CodeUnknown if the chain holds none.
func CodeOf(err error) Code {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.code
	}
	return CodeUnknown
}

type wireError struct {
	Code    Code   `json:"code"`
	Message string `json:"message,omitempty"`
}

// https://connectrpc.com/docs/protocol#http-to-error-code
func codeFromStatus(status int) Code {
	switch status {
	case http.StatusBadRequest:
		return CodeInternal
	case http.StatusUnauthorized:
		return CodeUnauthenticated
	case http.StatusForbidden:
		return CodePermissionDenied
	case http.StatusNotFound:
		return CodeUnimplemented
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return CodeUnavailable
	default:
		return CodeUnknown
	}
}
