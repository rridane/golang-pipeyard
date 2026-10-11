package otel

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

type TelemetryLoggerInjector struct {
	// service name displayed in otel logs and json logs
	ServiceName string
	// false and logs are only displayed as json, true to export
	// provide OtelExporterOltpEndpoint if true
	EnableOtelExport bool
	// otel grpc exporter endpoint
	OtelExporterOtlpEndpoint string
	// log level in debug, trace, info, warn, error
	LogLevel     zapcore.Level
	OtelInsecure bool
	// Protocol is ProtocolGRPC when empty.
	Protocol string
}

func (l *TelemetryLoggerInjector) Inject(stageName string) *zap.Logger {
	return l.newLogger(stageName)
}

// newLogger combines a stdout core with an optional OTLP one.
func (l *TelemetryLoggerInjector) newLogger(stageName string) *zap.Logger {
	level := l.LogLevel

	var cores []zapcore.Core

	encCfg := zap.NewDevelopmentEncoderConfig()
	encCfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
	consoleEncoder := zapcore.NewConsoleEncoder(encCfg)
	consoleCore := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level)
	cores = append(cores, consoleCore)

	if l.EnableOtelExport {
		if l.OtelExporterOtlpEndpoint == "" {
			fmt.Println("log export disabled: OtelExporterOtlpEndpoint is empty")
		} else {
			lp, err := l.initOtelLogProvider(stageName, l.OtelExporterOtlpEndpoint)
			if err != nil {
				fmt.Printf("OTLP log exporter unavailable: %v\n", err)
			} else {
				cores = append(cores, NewOtelZapCore(lp))
			}
		}
	}

	logger := zap.New(zapcore.NewTee(cores...))

	fmt.Printf("log level: %s\n", level)

	return logger
}

func (l *TelemetryLoggerInjector) newExporter(ctx context.Context, endpoint string) (sdklog.Exporter, error) {
	if l.Protocol == ProtocolHTTP {
		options := []otlploghttp.Option{otlploghttp.WithEndpoint(endpoint)}
		if l.OtelInsecure {
			options = append(options, otlploghttp.WithInsecure())
		}
		return otlploghttp.New(ctx, options...)
	}

	options := []otlploggrpc.Option{otlploggrpc.WithEndpoint(endpoint)}
	if l.OtelInsecure {
		options = append(options, otlploggrpc.WithInsecure())
	}
	return otlploggrpc.New(ctx, options...)
}

func (l *TelemetryLoggerInjector) initOtelLogProvider(stageName, endpoint string) (*sdklog.LoggerProvider, error) {
	ctx := context.Background()

	exporter, err := l.newExporter(ctx, endpoint)

	if err != nil {
		return nil, fmt.Errorf("create the OTLP log exporter: %w", err)
	}

	res, _ := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(l.ServiceName),
			semconv.ServiceNamespaceKey.String(stageName),
		),
	)

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	return lp, nil
}

func (l *TelemetryLoggerInjector) Shutdown(ctx context.Context) {
	globalProvider := global.GetLoggerProvider()
	if provider, ok := globalProvider.(*sdklog.LoggerProvider); ok {
		if err := provider.Shutdown(ctx); err != nil {
			fmt.Printf("logger provider not shut down cleanly: %v\n", err)
		}
	}
}

// otelZapCore bridges Zap records to OpenTelemetry logs.
type otelZapCore struct {
	provider *sdklog.LoggerProvider
}

func NewOtelZapCore(provider *sdklog.LoggerProvider) zapcore.Core {
	// Set globally so global.Logger() sees it.
	global.SetLoggerProvider(provider)
	return &otelZapCore{provider: provider}
}

func (o *otelZapCore) Enabled(l zapcore.Level) bool { return true }

func (o *otelZapCore) With(fields []zapcore.Field) zapcore.Core { return o }

func (o *otelZapCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if o.Enabled(entry.Level) {
		return ce.AddCore(entry, o)
	}
	return ce
}

func (o *otelZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	ctx := context.Background()

	logger := o.provider.Logger("zap_bridge")

	record := log.Record{}
	record.SetTimestamp(entry.Time)
	record.SetObservedTimestamp(time.Now())
	record.SetSeverityText(entry.Level.String())
	record.SetBody(log.StringValue(entry.Message))

	for _, f := range fields {
		record.AddAttributes(log.String(f.Key, f.String))
	}

	logger.Emit(ctx, record)
	return nil
}

func (o *otelZapCore) Sync() error { return nil }
