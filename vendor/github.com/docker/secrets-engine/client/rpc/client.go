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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
)

const (
	contentTypeProto = "application/proto"
	contentTypeJSON  = "application/json"

	headerProtocolVersion = "Connect-Protocol-Version"
	headerTimeout         = "Connect-Timeout-Ms"

	protocolVersion = "1"
	maxTimeoutMs    = 9_999_999_999
)

// HTTPClient is the part of [http.Client] that a [Client] uses.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	httpClient HTTPClient
	baseURL    string
}

func NewClient(httpClient HTTPClient, baseURL string) *Client {
	return &Client{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

func (c *Client) Call(ctx context.Context, procedure string, req, res proto.Message) error {
	body, err := proto.Marshal(req)
	if err != nil {
		return NewError(CodeInternal, fmt.Errorf("marshal request: %w", err))
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+procedure, bytes.NewReader(body))
	if err != nil {
		return NewError(CodeUnknown, err)
	}
	httpReq.Header.Set("Content-Type", contentTypeProto)
	httpReq.Header.Set(headerProtocolVersion, protocolVersion)
	if deadline, ok := ctx.Deadline(); ok {
		if ms := ceilMilliseconds(time.Until(deadline)); ms > 0 && ms <= maxTimeoutMs {
			httpReq.Header.Set(headerTimeout, strconv.FormatInt(ms, 10))
		}
	}
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return transportError(ctx, err)
	}
	defer httpResp.Body.Close()
	contentType, _, _ := mime.ParseMediaType(httpResp.Header.Get("Content-Type"))
	if httpResp.StatusCode != http.StatusOK {
		return errorFromResponse(ctx, httpResp, contentType)
	}
	if contentType != contentTypeProto {
		return NewError(CodeUnknown, fmt.Errorf("invalid content-type %q: expecting %q", contentType, contentTypeProto))
	}
	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return transportError(ctx, fmt.Errorf("read response: %w", err))
	}
	if err := proto.Unmarshal(data, res); err != nil {
		return NewError(CodeInternal, fmt.Errorf("unmarshal response: %w", err))
	}
	return nil
}

func ceilMilliseconds(d time.Duration) int64 {
	ms := d.Milliseconds()
	if d%time.Millisecond > 0 {
		ms++
	}
	return ms
}

func transportError(ctx context.Context, err error) *Error {
	if e, ok := errors.AsType[*Error](err); ok && ctx.Err() == nil {
		return e
	}
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		err = urlErr.Err
	}
	if ctxErr := ctx.Err(); ctxErr != nil && !errors.Is(err, ctxErr) {
		err = fmt.Errorf("%w: %w", ctxErr, err)
	}
	switch {
	case errors.Is(err, context.Canceled):
		return NewError(CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):
		return NewError(CodeDeadlineExceeded, err)
	default:
		return NewError(CodeUnavailable, err)
	}
}

func errorFromResponse(ctx context.Context, resp *http.Response, contentType string) *Error {
	if err := ctx.Err(); err != nil {
		return transportError(ctx, err)
	}
	if deadline, ok := ctx.Deadline(); ok && !time.Now().Before(deadline) {
		return NewError(CodeDeadlineExceeded, context.DeadlineExceeded)
	}
	code := codeFromStatus(resp.StatusCode)
	if contentType != contentTypeJSON {
		return NewError(code, errors.New(resp.Status))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return transportError(ctx, fmt.Errorf("read response: %w", err))
	}
	var wire wireError
	if json.Unmarshal(body, &wire) != nil {
		return NewError(code, errors.New(resp.Status))
	}
	if wire.Code.valid() {
		code = wire.Code
	}
	return NewError(code, errors.New(wire.Message))
}
