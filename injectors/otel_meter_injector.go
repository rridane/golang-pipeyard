package injectors

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
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
	// otel grpc exporter endpoint
	OtelExporterOtlpEndpoint string
	Insecure                 bool
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
		fmt.Println("⚠️  OTEL_EXPORTER_OTLP_ENDPOINT non défini, métriques désactivées")
		return noop.NewMeterProvider()
	}

	ctx := context.Background()

	options := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}

	if m.Insecure {
		options = append(options, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx,
		options...,
	)
	if err != nil {
		fmt.Println("❌ Impossible de créer le metric exporter:", err)
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

func (m *MeterInjector) Shutdown(ctx context.Context) {
	if provider, ok := m.provider.(*sdkmetric.MeterProvider); ok {
		if err := provider.Shutdown(ctx); err != nil {
			fmt.Printf("⚠️ Erreur lors de l’arrêt du MeterProvider: %v\n", err)
		}
	}
}
