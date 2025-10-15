package injectors

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type TracerInjector struct {
	ServiceName string
	provider    trace.TracerProvider
	// otel grpc exporter endpoint
	OtelExporterOtlpEndpoint string
	insecure                 bool
}

func (t *TracerInjector) Inject(stageName string) trace.Tracer {
	if t.provider == nil {
		t.provider = t.initTracerProvider()
	}
	tracer := t.provider.Tracer("pipeline_" + stageName)
	return tracer
}

func (t *TracerInjector) initTracerProvider() trace.TracerProvider {
	endpoint := t.OtelExporterOtlpEndpoint
	if endpoint == "" {
		fmt.Println("⚠️  OTEL_EXPORTER_OTLP_ENDPOINT non défini, traces désactivées")
		return noop.NewTracerProvider()
	}

	ctx := context.Background()

	options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}

	if t.insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx,
		options...,
	)

	if err != nil {
		fmt.Println("❌ Impossible de créer l’exporter OTLP:", err)
		return noop.NewTracerProvider()
	}

	res, _ := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(t.ServiceName)),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp
}

func (t *TracerInjector) Shutdown(ctx context.Context) {
	if provider, ok := t.provider.(*sdktrace.TracerProvider); ok {
		_ = provider.Shutdown(ctx)
	}
}
