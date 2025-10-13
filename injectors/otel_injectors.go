package injectors

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	noopTracer "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"os"
)

type MeterInjector struct {
	provider metric.MeterProvider
}

func (m *MeterInjector) Inject(stageName string) metric.Meter {
	if m.provider != nil {
		return m.provider.Meter("pipeline_" + stageName)
	}
	return noop.NewMeterProvider().Meter("pipeline")
}

type LoggerInjector struct{}

func (l *LoggerInjector) Inject(stageName string) *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		return zap.NewExample()
	}
	return logger.With(zap.String("stage", stageName))
}

type TracerInjector struct {
	serviceName string
}

func (t *TracerInjector) Inject(stageName string) *trace.Tracer {
	provider := getTracerProvider(t.serviceName)
	if provider == nil {
		tracer := noopTracer.NewTracerProvider().Tracer("pipeline")
		return &tracer
	}
	tracer := provider.Tracer("pipeline")
	return &tracer
}

func getTracerProvider(serviceName string) *sdktrace.TracerProvider {

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return nil
	}

	var exporter sdktrace.SpanExporter

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp
}
