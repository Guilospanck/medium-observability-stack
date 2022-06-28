package telemetry

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Creates Jaeger exporter
func exporterToJaeger() (*jaeger.Exporter, error) {
	const OPEN_TELEMETRY_COLLECTOR_URL = "http://localhost:14278/api/traces" // DOCKER-COMPOSE
	// const OPEN_TELEMETRY_COLLECTOR_URL = "http://otel-collector-service.telemetry.svc:14278/api/traces" // KUBERNETES
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(OPEN_TELEMETRY_COLLECTOR_URL)))
}

// Returns a new OpenTelemetry resource describing this application.
func newResource(ctx context.Context) *resource.Resource {
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String("OBSERVABILITY-MEDIUM-SERVICE-EXAMPLE"),
			attribute.String("environment", "DEVELOPMENT"),
		),
	)
	if err != nil {
		log.Fatalf("%s: %v", "Failed to create resource", err)
	}

	return res
}

// Helper function to define sampling.
// When in development mode, AlwaysSample is defined,
// otherwise, sample based on Parent and IDRatio will be used.
func getSampler() trace.Sampler {
	ENV := os.Getenv("GO_ENV")

	switch ENV {
	case "development":
		return trace.AlwaysSample()
	case "production":
		return trace.ParentBased(trace.TraceIDRatioBased(0.5))
	default:
		return trace.AlwaysSample()
	}
}

// Initiates OpenTelemetry provider sending data to OpenTelemetry Collector.
func InitProviderWithJaegerExporter(ctx context.Context) (func(context.Context) error, error) {
	exp, err := exporterToJaeger()
	if err != nil {
		log.Fatalf("error: %s", err.Error())
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(getSampler()),
		trace.WithBatcher(exp),
		trace.WithResource(newResource(ctx)),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
