package clientlib

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/licensing/lib/errors"
)

// Do is a shortcut for creating and executing an http request.
func Do(ctx context.Context, method, urlStr string, opts ...RequestOption) (*http.Request, *http.Response, error) {
	r, err := New(ctx, method, urlStr, opts...)
	if err != nil {
		return nil, nil, err
	}
	res, err := r.Do()
	return r.Request, res, err
}

// New creates and returns a new Request, potentially configured via a vector of RequestOption's.
func New(ctx context.Context, method, urlStr string, opts ...RequestOption) (*Request, error) {
	req, err := http.NewRequest(method, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	r := &Request{
		Request:            req,
		Client:             &http.Client{},
		ErrorCheck:         DefaultErrorCheck,
		ErrorBodyMaxLength: defaultErrBodyMaxLength,
		ErrorSummary:       DefaultErrorSummary,
		RequestPrepare:     DefaultRequestPrepare,
		ResponseHandle:     DefaultResponseHandle,
	}

	for _, o := range opts {
		o(r)
	}

	return r, nil
}

// Request encompasses an http.Request, plus configured behavior options.
type Request struct {
	*http.Request
	Client             *http.Client
	ErrorCheck         ErrorCheck
	ErrorBodyMaxLength int64
	ErrorSummary       ErrorSummary
	ResponseHandle     ResponseHandle
	RequestPrepare     RequestPrepare
}

// Do executes the Request. The Request.ErrorCheck to determine
// if this attempt has failed, and transform the returned error.
// Otherwise, Request.ResponseHandler will examine the response.
// It's expected that the ResponseHandler has been configured via
// a RequestOption to perform response parsing and storing.
func (r *Request) Do() (*http.Response, error) {
	err := r.RequestPrepare(r)
	if err != nil {
		return nil, err
	}
	res, err := r.Client.Do(r.Request)
	err = r.ErrorCheck(r, err, res)
	if err != nil {
		return res, err
	}
	return res, r.ResponseHandle(r, res)
}

// SetBody mirrors the ReadCloser config in http.NewRequest,
// ensuring that a ReadCloser is used for http.Request.Body.
func (r *Request) SetBody(body io.Reader) {
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
	}
	r.Body = rc
}

// ErrorFields returns error annotation fields for the request.
func (r *Request) ErrorFields() map[string]interface{} {
	return map[string]interface{}{
		"url":    r.URL.String(),
		"method": r.Method,
	}
}

// ErrorCheck is the signature for the function that is passed
// the error & response immediately from http.Client.Do().
type ErrorCheck func(r *Request, doErr error, res *http.Response) error

// DefaultErrorCheck is the default error checker used if none is
// configured on a Request. doErr and res are the return values of
// executing http.Client.Do(), so any implementation should first
// check doErr for non-nill & react appropriately. If an http response
// was received, if a non-200 class status was also received, then
// the response body will be read (up to a const limit) and passed
// to request.ErrorSummary to attempt to parse out the error body,
// which will be passed as the "detail" flag on the returned error.
func DefaultErrorCheck(r *Request, doErr error, res *http.Response) error {
	if doErr != nil {
		return errors.Wrap(doErr, r.ErrorFields())
	}
	status := res.StatusCode
	if status >= 200 && status < 300 {
		return nil
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(io.LimitReader(res.Body, r.ErrorBodyMaxLength))
	detail := r.ErrorSummary(body)

	message := fmt.Sprintf("%s %s returned %d : %s", r.Method, r.URL.String(), status, detail)
	return errors.NewHTTPError(status, message).
		With(r.ErrorFields()).
		With(map[string]interface{}{
			"http_status": status,
			"detail":      detail,
		})
}

// Default error response max length, in bytes
const defaultErrBodyMaxLength = 256

// ErrorSummary is the signature for the function that is passed
// the fully read body of an error response.
type ErrorSummary func([]byte) string

// DefaultErrorSummary just returns the string of the received
// error body. Note that the body passed in is potentially truncated
// before this call.
func DefaultErrorSummary(body []byte) string {
	return string(body)
}

// RequestPrepare is the signature for the function called
// before calling http.Client.Do, to perform any preparation
// needed before executing the request, eg. marshaling the body.
type RequestPrepare func(r *Request) error

// DefaultRequestPrepare does nothing.
func DefaultRequestPrepare(*Request) error {
	return nil
}

// ResponseHandle is the signature for the function called if
// ErrorCheck returns a nil error, and is responsible for performing
// any reads or stores from the request & response.
type ResponseHandle func(*Request, *http.Response) error

// DefaultResponseHandle merely closes the response body.
func DefaultResponseHandle(r *Request, res *http.Response) error {
	res.Body.Close()
	return nil
}

// RequestOption is the signature for functions that can perform
// some configuration of a Request.
type RequestOption func(*Request)

// SendJSON returns a RequestOption that will marshal
// and set the json body & headers on a request.
func SendJSON(sends interface{}) RequestOption {
	return func(r *Request) {
		r.Header.Set("Content-Type", "application/json")
		r.RequestPrepare = func(r *Request) error {
			bits, err := json.Marshal(sends)
			if err != nil {
				return errors.Wrap(err, r.ErrorFields())
			}
			body := bytes.NewReader(bits)
			r.SetBody(body)
			r.ContentLength = int64(body.Len())
			return nil
		}
	}
}

// RecvJSON returns a RequestOption that will set the json headers
// on a request, and set a ResponseHandler that will unmarshal
// the response body to the given interface{}.
func RecvJSON(recvs interface{}) RequestOption {
	return func(r *Request) {
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Accept-Charset", "utf-8")
		r.ResponseHandle = func(r *Request, res *http.Response) error {
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return errors.Wrap(err, r.ErrorFields())
			}

			err = json.Unmarshal(body, recvs)
			if err != nil {
				return errors.Wrap(err, r.ErrorFields())
			}

			return nil
		}
	}
}

// SendXML returns a RequestOption that will marshal
// and set the xml body & headers on a request.
func SendXML(sends interface{}) RequestOption {
	return func(r *Request) {
		r.Header.Set("Content-Type", "application/xml")
		r.RequestPrepare = func(r *Request) error {
			bits, err := xml.Marshal(sends)
			if err != nil {
				return errors.Wrap(err, r.ErrorFields())
			}
			body := bytes.NewReader(bits)
			r.SetBody(body)
			r.ContentLength = int64(body.Len())
			return nil
		}
	}
}

// RecvXML returns a RequestOption that will set the xml headers
// on a request, and set a ResponseHandler that will unmarshal
// the response body to the given interface{}.
func RecvXML(recvs interface{}) RequestOption {
	return func(r *Request) {
		r.Header.Set("Accept", "application/xml")
		r.Header.Set("Accept-Charset", "utf-8")
		r.ResponseHandle = func(r *Request, res *http.Response) error {
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return errors.Wrap(err, r.ErrorFields())
			}

			err = xml.Unmarshal(body, recvs)
			if err != nil {
				return errors.Wrap(err, r.ErrorFields())
			}

			return nil
		}
	}
}

// SendText returns a RequestOption that will marshal
// and set the text body & headers on a request.
func SendText(sends string) RequestOption {
	return func(r *Request) {
		r.Header.Set("Content-Type", "text/plain")
		r.RequestPrepare = func(r *Request) error {
			body := strings.NewReader(sends)
			r.SetBody(body)
			r.ContentLength = int64(body.Len())
			return nil
		}
	}
}

// RecvText returns a RequestOption that will set the text headers
// on a request, and set a ResponseHandler that will unmarshal
// the response body to the given string.
func RecvText(recvs *string) RequestOption {
	return func(r *Request) {
		r.Header.Set("Accept", "text/plain")
		r.Header.Set("Accept-Charset", "utf-8")
		r.ResponseHandle = func(r *Request, res *http.Response) error {
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return errors.Wrap(err, r.ErrorFields())
			}

			sbody := string(body)
			*recvs = sbody

			return nil
		}
	}
}

// DontClose sets the ResponseBody to an empty function, so that the
// response body is not automatically closed. Users of this should be
// sure to call res.Body.Close().
func DontClose() RequestOption {
	return func(r *Request) {
		r.ResponseHandle = func(*Request, *http.Response) error {
			return nil
		}
	}
}
