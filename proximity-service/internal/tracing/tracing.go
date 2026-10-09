package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// localCollector is where traces go when OTEL_EXPORTER_OTLP_ENDPOINT is unset:
// a Jaeger on this machine (docker-compose.override.yml).
const localCollector = "http://localhost:4318/v1/traces"

// InitTracer sets up the OpenTelemetry tracer provider with an OTLP/HTTP
// exporter that sends to OTEL_EXPORTER_OTLP_ENDPOINT (Jaeger).  With
// OTEL_TRACES_EXPORTER=none nothing is exported and spans stay no-ops.
// Returns a shutdown function that should be deferred in main.
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	if os.Getenv("OTEL_TRACES_EXPORTER") == "none" {
		return func(context.Context) error { return nil }, nil
	}

	var opts []otlptracehttp.Option
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" && os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") == "" {
		opts = append(opts, otlptracehttp.WithEndpointURL(localCollector))
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
