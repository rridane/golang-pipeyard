package inj

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
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

// TelemetryLoggerInjector is a logger dependency injection with specific configuration
//
// Provides an implementation which can log as json and export to opentelemetry
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
}

// Inject crée un logger configuré pour le stage demandé
func (l *TelemetryLoggerInjector) Inject(stageName string) *zap.Logger {
	return l.newLogger(stageName)
}

// newLogger construit un logger combinant stdout + OTLP (optionnel)
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
			fmt.Println("⚠️  OTEL_EXPORTER_OTLP_ENDPOINT non défini, export des logs désactivé")
		} else {
			lp, err := l.initOtelLogProvider(stageName, l.OtelExporterOtlpEndpoint)
			if err != nil {
				fmt.Printf("⚠️  Impossible d'initialiser OTLP Log Exporter: %v\n", err)
			} else {
				cores = append(cores, NewOtelZapCore(lp))
				//fmt.Println("✅ OTLP Log Exporter activé :", endpoint)
			}
		}
	}

	// === Logger combiné ===
	logger := zap.New(zapcore.NewTee(cores...))

	fmt.Printf("📘 Niveau de log configuré: %s\n", level)

	return logger
}

// initOtelLogProvider configure le pipeline OpenTelemetry pour l’export des logs
func (l *TelemetryLoggerInjector) initOtelLogProvider(stageName, endpoint string) (*sdklog.LoggerProvider, error) {
	ctx := context.Background()

	options := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(endpoint),
	}

	if l.OtelInsecure {
		options = append(options, otlploggrpc.WithInsecure())
	}

	exporter, err := otlploggrpc.New(ctx,
		options...,
	)

	if err != nil {
		return nil, fmt.Errorf("échec création exporter OTLP: %w", err)
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
			fmt.Printf("⚠️ Erreur lors de l’arrêt du LoggerProvider: %v\n", err)
		}
	}
}

// otelZapCore implémente zapcore.Core et redirige les logs Zap vers OpenTelemetry Logs.
type otelZapCore struct {
	provider *sdklog.LoggerProvider
}

// NewOtelZapCore crée un core Zap qui exporte les logs vers le provider OTel.
func NewOtelZapCore(provider *sdklog.LoggerProvider) zapcore.Core {
	// Enregistre ce provider globalement pour que global.Logger() le voie
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

	// On récupère le logger depuis le provider (API publique)
	logger := o.provider.Logger("zap_bridge")

	// Création du record selon l'API log
	record := log.Record{}
	record.SetTimestamp(entry.Time)
	record.SetObservedTimestamp(time.Now())
	record.SetSeverityText(entry.Level.String())
	record.SetBody(log.StringValue(entry.Message))

	// Convertit les champs zap en attributs log.KeyValue
	for _, f := range fields {
		record.AddAttributes(log.String(f.Key, f.String))
	}

	// On émet le record via l’API publique (Emit renvoie void)
	logger.Emit(ctx, record)
	return nil
}

func (o *otelZapCore) Sync() error { return nil }
