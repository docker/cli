package command

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/docker/distribution/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const exportTimeout = 50 * time.Millisecond

// TracerProvider is an extension of the trace.TracerProvider interface for CLI programs.
type TracerProvider interface {
	trace.TracerProvider
	ForceFlush(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// MeterProvider is an extension of the metric.MeterProvider interface for CLI programs.
type MeterProvider interface {
	metric.MeterProvider
	ForceFlush(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// TelemetryClient provides the methods for using OTEL tracing or metrics.
type TelemetryClient interface {
	// Resource returns the OTEL Resource configured with this TelemetryClient.
	// This resource may be created lazily, but the resource should be the same
	// each time this function is invoked.
	Resource() *resource.Resource

	// TracerProvider returns a TracerProvider. This TracerProvider will be configured
	// with the default tracing components for a CLI program along with any options given
	// for the SDK.
	TracerProvider(ctx context.Context, opts ...sdktrace.TracerProviderOption) TracerProvider

	// MeterProvider returns a MeterProvider. This MeterProvider will be configured
	// with the default metric components for a CLI program along with any options given
	// for the SDK.
	MeterProvider(ctx context.Context, opts ...sdkmetric.Option) MeterProvider
}

func (cli *DockerCli) Resource() *resource.Resource {
	return cli.res.Get()
}

func (cli *DockerCli) TracerProvider(ctx context.Context, opts ...sdktrace.TracerProviderOption) TracerProvider {
	allOpts := make([]sdktrace.TracerProviderOption, 0, len(opts)+2)
	allOpts = append(allOpts, sdktrace.WithResource(cli.Resource()))
	allOpts = append(allOpts, dockerSpanExporter(ctx, cli)...)
	allOpts = append(allOpts, opts...)
	return sdktrace.NewTracerProvider(allOpts...)
}

func (cli *DockerCli) MeterProvider(ctx context.Context, opts ...sdkmetric.Option) MeterProvider {
	allOpts := make([]sdkmetric.Option, 0, len(opts)+2)
	allOpts = append(allOpts, sdkmetric.WithResource(cli.Resource()))
	allOpts = append(allOpts, dockerMetricExporter(ctx, cli)...)
	allOpts = append(allOpts, opts...)
	return sdkmetric.NewMeterProvider(allOpts...)
}

// WithResourceOptions configures additional options for the default resource. The default
// resource will continue to include its default options.
func WithResourceOptions(opts ...resource.Option) CLIOption {
	return func(cli *DockerCli) error {
		cli.res.AppendOptions(opts...)
		return nil
	}
}

// WithResource overwrites the default resource and prevents its creation.
func WithResource(res *resource.Resource) CLIOption {
	return func(cli *DockerCli) error {
		cli.res.Set(res)
		return nil
	}
}

type telemetryResource struct {
	res  *resource.Resource
	opts []resource.Option
	once sync.Once
}

func (r *telemetryResource) Set(res *resource.Resource) {
	r.res = res
}

func (r *telemetryResource) Get() *resource.Resource {
	r.once.Do(r.init)
	return r.res
}

func (r *telemetryResource) init() {
	if r.res != nil {
		r.opts = nil
		return
	}

	opts := append(defaultResourceOptions(), r.opts...)
	res, err := resource.New(context.Background(), opts...)
	if err != nil {
		otel.Handle(err)
	}
	r.res = res

	// Clear the resource options since they'll never be used again and to allow
	// the garbage collector to retrieve that memory.
	r.opts = nil
}

func defaultResourceOptions() []resource.Option {
	return []resource.Option{
		resource.WithDetectors(serviceNameDetector{}),
		resource.WithAttributes(
			// Use a unique instance id so OTEL knows that each invocation
			// of the CLI is its own instance. Without this, downstream
			// OTEL processors may think the same process is restarting
			// continuously.
			semconv.ServiceInstanceID(uuid.Generate().String()),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	}
}

func (r *telemetryResource) AppendOptions(opts ...resource.Option) {
	if r.res != nil {
		return
	}
	r.opts = append(r.opts, opts...)
}

type serviceNameDetector struct{}

func (serviceNameDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	return resource.StringDetector(
		semconv.SchemaURL,
		semconv.ServiceNameKey,
		func() (string, error) {
			return filepath.Base(os.Args[0]), nil
		},
	).Detect(ctx)
}

// cliReader is an implementation of Reader that will automatically
// report to a designated Exporter when Shutdown is called.
type cliReader struct {
	sdkmetric.Reader
	exporter sdkmetric.Exporter
}

func newCLIReader(exp sdkmetric.Exporter) sdkmetric.Reader {
	reader := sdkmetric.NewManualReader(
		sdkmetric.WithTemporalitySelector(deltaTemporality),
	)
	return &cliReader{
		Reader:   reader,
		exporter: exp,
	}
}

func (r *cliReader) Shutdown(ctx context.Context) error {
	var rm metricdata.ResourceMetrics
	if err := r.Reader.Collect(ctx, &rm); err != nil {
		return err
	}

	// Place a pretty tight constraint on the actual reporting.
	// We don't want CLI metrics to prevent the CLI from exiting
	// so if there's some kind of issue we need to abort pretty
	// quickly.
	ctx, cancel := context.WithTimeout(ctx, exportTimeout)
	defer cancel()

	return r.exporter.Export(ctx, &rm)
}

// deltaTemporality sets the Temporality of every instrument to delta.
//
// This isn't really needed since we create a unique resource on each invocation,
// but it can help with cardinality concerns for downstream processors since they can
// perform aggregation for a time interval and then discard the data once that time
// period has passed. Cumulative temporality would imply to the downstream processor
// that they might receive a successive point and they may unnecessarily keep state
// they really shouldn't.
func deltaTemporality(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}
