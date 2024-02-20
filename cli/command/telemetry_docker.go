package command

import (
	"context"
	"fmt"
	"net/url"
	"path"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const otelContextFieldName = "otel"

// dockerExporterOTLPEndpoint retrieves the OTLP endpoint used for the docker reporter
// from the current context.
func dockerExporterOTLPEndpoint(cli Cli) (endpoint string, secure bool) {
	meta, err := cli.ContextStore().GetMetadata(cli.CurrentContext())
	if err != nil {
		otel.Handle(err)
		return "", false
	}

	var otelCfg any
	switch m := meta.Metadata.(type) {
	case DockerContext:
		otelCfg = m.AdditionalFields[otelContextFieldName]
	case map[string]any:
		otelCfg = m[otelContextFieldName]
	}

	if otelCfg == nil {
		return "", false
	}

	otelMap, ok := otelCfg.(map[string]any)
	if !ok {
		otel.Handle(errors.Errorf(
			"unexpected type for field %q: %T (expected: %T)",
			otelContextFieldName,
			otelCfg,
			otelMap,
		))
		return "", false
	}

	// keys from https://opentelemetry.io/docs/concepts/sdk-configuration/otlp-exporter-configuration/
	endpoint, ok = otelMap["OTEL_EXPORTER_OTLP_ENDPOINT"].(string)
	if !ok {
		return "", false
	}

	// Parse the endpoint. The docker config expects the endpoint to be
	// in the form of a URL to match the environment variable, but this
	// option doesn't correspond directly to WithEndpoint.
	//
	// We pretend we're the same as the environment reader.
	u, err := url.Parse(endpoint)
	if err != nil {
		otel.Handle(errors.Errorf("docker otel endpoint is invalid: %s", err))
		return "", false
	}

	switch u.Scheme {
	case "unix":
		// Unix sockets are a bit weird. OTEL seems to imply they
		// can be used as an environment variable and are handled properly,
		// but they don't seem to be as the behavior of the environment variable
		// is to strip the scheme from the endpoint, but the underlying implementation
		// needs the scheme to use the correct resolver.
		//
		// We'll just handle this in a special way and add the unix:// back to the endpoint.
		endpoint = fmt.Sprintf("unix://%s", path.Join(u.Host, u.Path))
	case "https":
		secure = true
		fallthrough
	case "http":
		endpoint = path.Join(u.Host, u.Path)
	}
	return endpoint, secure
}

func dockerSpanExporter(ctx context.Context, cli Cli) []sdktrace.TracerProviderOption {
	endpoint, secure := dockerExporterOTLPEndpoint(cli)
	if endpoint == "" {
		return nil
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	if !secure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		otel.Handle(err)
		return nil
	}
	return []sdktrace.TracerProviderOption{sdktrace.WithBatcher(exp, sdktrace.WithExportTimeout(exportTimeout))}
}

func dockerMetricExporter(ctx context.Context, cli Cli) []sdkmetric.Option {
	endpoint, secure := dockerExporterOTLPEndpoint(cli)
	if endpoint == "" {
		return nil
	}

	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
	}
	if !secure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	exp, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		otel.Handle(err)
		return nil
	}
	return []sdkmetric.Option{sdkmetric.WithReader(newCLIReader(exp))}
}
