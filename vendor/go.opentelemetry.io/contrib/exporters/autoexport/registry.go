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
	"errors"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	otelExporterOTLPProtoEnvKey = "OTEL_EXPORTER_OTLP_PROTOCOL"
)

// registry maintains a map of exporter names to SpanExporter factories
// func(context.Context) (trace.SpanExporter, error) that is safe for concurrent use by multiple
// goroutines without additional locking or coordination.
type registry struct {
	mu    sync.Mutex
	names map[string]func(context.Context) (trace.SpanExporter, error)
}

func newRegistry() registry {
	return registry{
		names: map[string]func(context.Context) (trace.SpanExporter, error){
			"":     buildOTLPExporter,
			"otlp": buildOTLPExporter,
			"none": func(ctx context.Context) (trace.SpanExporter, error) { return noop{}, nil },
		},
	}
}

var (
	// envRegistry is the package level registry of exporter registrations
	// and their mapping to a SpanExporter factory func(context.Context) (trace.SpanExporter, error).
	envRegistry = newRegistry()

	// errUnknownExporter is returned when an unknown exporter name is used in
	// the OTEL_*_EXPORTER environment variables.
	errUnknownExporter = errors.New("unknown exporter")

	// errInvalidOTLPProtocol is returned when an invalid protocol is used in
	// the OTEL_EXPORTER_OTLP_PROTOCOL environment variable.
	errInvalidOTLPProtocol = errors.New("invalid OTLP protocol - should be one of ['grpc', 'http/protobuf']")

	// errDuplicateRegistration is returned when an duplicate registration is detected.
	errDuplicateRegistration = errors.New("duplicate registration")
)

// load returns tries to find the SpanExporter factory with the key and
// then execute the factory, returning the created SpanExporter.
// errUnknownExporter is returned if the registration is missing and the error from
// executing the factory if not nil.
func (r *registry) load(ctx context.Context, key string) (trace.SpanExporter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	factory, ok := r.names[key]
	if !ok {
		return nil, errUnknownExporter
	}
	return factory(ctx)
}

// store sets the factory for a key if is not already in the registry. errDuplicateRegistration
// is returned if the registry already contains key.
func (r *registry) store(key string, factory func(context.Context) (trace.SpanExporter, error)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.names[key]; ok {
		return fmt.Errorf("%w: %q", errDuplicateRegistration, key)
	}
	r.names[key] = factory
	return nil
}

// drop removes key from the registry if it exists, otherwise nothing.
func (r *registry) drop(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.names, key)
}

// RegisterSpanExporter sets the SpanExporter factory to be used when the
// OTEL_TRACES_EXPORTERS environment variable contains the exporter name. This
// will panic if name has already been registered.
func RegisterSpanExporter(name string, factory func(context.Context) (trace.SpanExporter, error)) {
	if err := envRegistry.store(name, factory); err != nil {
		// envRegistry.store will return errDuplicateRegistration if name is already
		// registered. Panic here so the user is made aware of the duplicate
		// registration, which could be done by malicious code trying to
		// intercept cross-cutting concerns.
		//
		// Panic for all other errors as well. At this point there should not
		// be any other errors returned from the store operation. If there
		// are, alert the developer that adding them as soon as possible that
		// they need to be handled here.
		panic(err)
	}
}

// spanExporter returns a span exporter using the passed in name
// from the list of registered SpanExporters. Each name must match an
// already registered SpanExporter. A default OTLP exporter is registered
// under both an empty string "" and "otlp".
// An error is returned for any unknown exporters.
func spanExporter(ctx context.Context, name string) (trace.SpanExporter, error) {
	exp, err := envRegistry.load(ctx, name)
	if err != nil {
		return nil, err
	}
	return exp, nil
}

// buildOTLPExporter creates an OTLP exporter using the environment variable
// OTEL_EXPORTER_OTLP_PROTOCOL to determine the exporter protocol.
// Defaults to http/protobuf protocol.
func buildOTLPExporter(ctx context.Context) (trace.SpanExporter, error) {
	proto := os.Getenv(otelExporterOTLPProtoEnvKey)
	if proto == "" {
		proto = "http/protobuf"
	}

	switch proto {
	case "grpc":
		return otlptracegrpc.New(ctx)
	case "http/protobuf":
		return otlptracehttp.New(ctx)
	default:
		return nil, errInvalidOTLPProtocol
	}
}
