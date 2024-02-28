// Copyright The OpenTelemetry Authors
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

package autoexport // import "go.opentelemetry.io/contrib/exporters/autoexport"

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
)

// noop is an implementation of trace.SpanExporter that performs no operations.
type noop struct{}

var _ trace.SpanExporter = noop{}

// ExportSpans is part of trace.SpanExporter interface.
func (e noop) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return nil
}

// Shutdown is part of trace.SpanExporter interface.
func (e noop) Shutdown(ctx context.Context) error {
	return nil
}

// IsNoneSpanExporter returns true for the exporter returned by [NewSpanExporter]
// when OTEL_TRACES_EXPORTER environment variable is set to "none".
func IsNoneSpanExporter(e trace.SpanExporter) bool {
	_, ok := e.(noop)
	return ok
}
