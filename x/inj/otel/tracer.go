package otel

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type TracerInjector struct {
	ServiceName string
	provider    trace.TracerProvider
	// otel exporter endpoint, as host:port without a scheme
	OtelExporterOtlpEndpoint string
	Insecure                 bool
	// Protocol is ProtocolGRPC when empty.
	Protocol string
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
		fmt.Println("tracing disabled: OtelExporterOtlpEndpoint is empty")
		return noop.NewTracerProvider()
	}

	ctx := context.Background()

	exporter, err := t.newExporter(ctx, endpoint)

	if err != nil {
		fmt.Println("create the OTLP trace exporter:", err)
		return noop.NewTracerProvider()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(runResource(ctx, t.ServiceName, true)),
	)
	otel.SetTracerProvider(tp)
	return tp
}

func (t *TracerInjector) newExporter(ctx context.Context, endpoint string) (sdktrace.SpanExporter, error) {
	if t.Protocol == ProtocolHTTP {
		options := []otlptracehttp.Option{otlptracehttp.WithEndpoint(endpoint)}
		if t.Insecure {
			options = append(options, otlptracehttp.WithInsecure())
		}
		return otlptracehttp.New(ctx, options...)
	}

	options := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(endpoint)}
	if t.Insecure {
		options = append(options, otlptracegrpc.WithInsecure())
	}
	return otlptracegrpc.New(ctx, options...)
}

func (t *TracerInjector) Shutdown(ctx context.Context) {
	if provider, ok := t.provider.(*sdktrace.TracerProvider); ok {
		_ = provider.Shutdown(ctx)
	}
}
