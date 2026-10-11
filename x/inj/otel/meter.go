package otel

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"time"
)

type MeterInjector struct {
	ServiceName string
	provider    metric.MeterProvider
	// otel exporter endpoint, as host:port without a scheme
	OtelExporterOtlpEndpoint string
	Insecure                 bool
	// Protocol is ProtocolGRPC when empty.
	Protocol string
}

func (m *MeterInjector) Inject(stageName string) metric.Meter {
	if m.provider == nil {
		m.provider = m.initMeterProvider()
	}
	return m.provider.Meter("pipeline_" + stageName)
}

func (m *MeterInjector) initMeterProvider() metric.MeterProvider {
	endpoint := m.OtelExporterOtlpEndpoint
	if endpoint == "" {
		fmt.Println("metrics disabled: OtelExporterOtlpEndpoint is empty")
		return noop.NewMeterProvider()
	}

	ctx := context.Background()

	exporter, err := m.newExporter(ctx, endpoint)
	if err != nil {
		fmt.Println("create the OTLP metric exporter:", err)
		return noop.NewMeterProvider()
	}

	res, _ := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(m.ServiceName)),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	return mp
}

func (m *MeterInjector) newExporter(ctx context.Context, endpoint string) (sdkmetric.Exporter, error) {
	if m.Protocol == ProtocolHTTP {
		options := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(endpoint)}
		if m.Insecure {
			options = append(options, otlpmetrichttp.WithInsecure())
		}
		return otlpmetrichttp.New(ctx, options...)
	}

	options := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
	if m.Insecure {
		options = append(options, otlpmetricgrpc.WithInsecure())
	}
	return otlpmetricgrpc.New(ctx, options...)
}

func (m *MeterInjector) Shutdown(ctx context.Context) {
	if provider, ok := m.provider.(*sdkmetric.MeterProvider); ok {
		if err := provider.Shutdown(ctx); err != nil {
			fmt.Printf("meter provider not shut down cleanly: %v\n", err)
		}
	}
}
