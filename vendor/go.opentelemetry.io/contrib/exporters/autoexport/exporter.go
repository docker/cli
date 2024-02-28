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
	"os"

	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	otelTracesExportersEnvKey = "OTEL_TRACES_EXPORTER"
)

type config struct {
	fallbackExporter trace.SpanExporter
}

func newConfig(ctx context.Context, opts ...Option) (config, error) {
	cfg := config{}
	for _, opt := range opts {
		cfg = opt.apply(cfg)
	}

	// if no fallback exporter is configured, use otlp exporter
	if cfg.fallbackExporter == nil {
		exp, err := spanExporter(context.Background(), "otlp")
		if err != nil {
			return cfg, err
		}
		cfg.fallbackExporter = exp
	}
	return cfg, nil
}

// Option applies an autoexport configuration option.
type Option interface {
	apply(config) config
}

type optionFunc func(config) config

func (fn optionFunc) apply(cfg config) config {
	return fn(cfg)
}

// WithFallbackSpanExporter sets the fallback exporter to use when no exporter
// is configured through the OTEL_TRACES_EXPORTER environment variable.
func WithFallbackSpanExporter(exporter trace.SpanExporter) Option {
	return optionFunc(func(cfg config) config {
		cfg.fallbackExporter = exporter
		return cfg
	})
}

// NewSpanExporter returns a configured [go.opentelemetry.io/otel/sdk/trace.SpanExporter]
// defined using the environment variables described below.
//
// OTEL_TRACES_EXPORTER defines the traces exporter; supported values:
//   - "none" - "no operation" exporter
//   - "otlp" (default) - OTLP exporter; see [go.opentelemetry.io/otel/exporters/otlp/otlptrace]
//
// OTEL_EXPORTER_OTLP_PROTOCOL defines OTLP exporter's transport protocol;
// supported values:
//   - "grpc" - protobuf-encoded data using gRPC wire format over HTTP/2 connection;
//     see: [go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc]
//   - "http/protobuf" (default) -  protobuf-encoded data over HTTP connection;
//     see: [go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp]
//
// An error is returned if an environment value is set to an unhandled value.
//
// Use [RegisterSpanExporter] to handle more values of OTEL_TRACES_EXPORTER.
//
// Use [WithFallbackSpanExporter] option to change the returned exporter
// when OTEL_TRACES_EXPORTER is unset or empty.
//
// Use [IsNoneSpanExporter] to check if the retured exporter is a "no operation" exporter.
func NewSpanExporter(ctx context.Context, opts ...Option) (trace.SpanExporter, error) {
	// prefer exporter configured via environment variables over exporter
	// passed in via exporter parameter
	envExporter, err := makeExporterFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	if envExporter != nil {
		return envExporter, nil
	}
	config, err := newConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return config.fallbackExporter, nil
}

// makeExporterFromEnv returns a configured SpanExporter defined by the OTEL_TRACES_EXPORTER
// environment variable.
// nil is returned if no exporter is defined for the environment variable.
func makeExporterFromEnv(ctx context.Context) (trace.SpanExporter, error) {
	expType := os.Getenv(otelTracesExportersEnvKey)
	if expType == "" {
		return nil, nil
	}
	return spanExporter(ctx, expType)
}
