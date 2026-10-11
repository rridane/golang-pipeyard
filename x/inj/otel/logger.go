package otel

import (
	"context"
	"fmt"
	"math"
	"os"
	"slices"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type TelemetryLoggerInjector struct {
	// ServiceName names the pipeline in the exported records.
	ServiceName string
	// EnableOtelExport adds the OTLP core to the console one. It needs
	// OtelExporterOtlpEndpoint.
	EnableOtelExport bool
	// OtelExporterOtlpEndpoint is host:port, without a scheme.
	OtelExporterOtlpEndpoint string
	// LogLevel is the console threshold; the OTLP core takes every record.
	LogLevel     zapcore.Level
	OtelInsecure bool
	// Protocol is ProtocolGRPC when empty.
	Protocol string

	provider *sdklog.LoggerProvider
	base     *zap.Logger
}

// Inject hands out one logger per stage, all sharing a single exporter. The
// stage name becomes a field, so every line says which stage wrote it. An
// empty name returns the bare logger, for a caller that is not a stage.
func (l *TelemetryLoggerInjector) Inject(stageName string) *zap.Logger {
	if l.base == nil {
		l.base = l.newLogger()
	}

	if stageName == "" {
		return l.base
	}

	return l.base.With(zap.String("stage", stageName))
}

// newLogger combines a stdout core with an optional OTLP one. It runs once: a
// provider per stage would leave every provider but the last unflushed, and
// records in an unflushed batch are dropped.
func (l *TelemetryLoggerInjector) newLogger() *zap.Logger {
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
			lp, err := l.initOtelLogProvider(l.OtelExporterOtlpEndpoint)
			if err != nil {
				fmt.Printf("OTLP log exporter unavailable: %v\n", err)
			} else {
				cores = append(cores, NewOtelZapCore(lp))
			}
		}
	}

	return zap.New(zapcore.NewTee(cores...))
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

func (l *TelemetryLoggerInjector) initOtelLogProvider(endpoint string) (*sdklog.LoggerProvider, error) {
	ctx := context.Background()

	exporter, err := l.newExporter(ctx, endpoint)

	if err != nil {
		return nil, fmt.Errorf("create the OTLP log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(runResource(ctx, l.ServiceName, true)),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	l.provider = lp

	return lp, nil
}

func (l *TelemetryLoggerInjector) Shutdown(ctx context.Context) {
	if l.provider == nil {
		return
	}
	if err := l.provider.Shutdown(ctx); err != nil {
		fmt.Printf("logger provider not shut down cleanly: %v\n", err)
	}
}

// otelZapCore bridges Zap records to OpenTelemetry logs.
type otelZapCore struct {
	provider *sdklog.LoggerProvider
	fields   []zapcore.Field
}

func NewOtelZapCore(provider *sdklog.LoggerProvider) zapcore.Core {
	// Set globally so global.Logger() sees it.
	global.SetLoggerProvider(provider)
	return &otelZapCore{provider: provider}
}

func (o *otelZapCore) Enabled(l zapcore.Level) bool { return true }

// With keeps the fields instead of dropping them, so a field attached once
// reaches every record that logger writes.
func (o *otelZapCore) With(fields []zapcore.Field) zapcore.Core {
	// Clipped, so two cores derived from the same parent cannot write into
	// each other's backing array.
	return &otelZapCore{
		provider: o.provider,
		fields:   append(slices.Clip(o.fields), fields...),
	}
}

func (o *otelZapCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if o.Enabled(entry.Level) {
		return ce.AddCore(entry, o)
	}
	return ce
}

func (o *otelZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {

	record := log.Record{}
	record.SetTimestamp(entry.Time)
	record.SetObservedTimestamp(time.Now())
	record.SetSeverity(severityOf(entry.Level))
	record.SetSeverityText(entry.Level.String())
	record.SetBody(log.StringValue(entry.Message))

	for _, f := range slices.Concat(o.fields, fields) {
		if value, ok := attributeValue(f); ok {
			record.AddAttributes(log.KeyValue{Key: f.Key, Value: value})
		}
	}

	o.provider.Logger("zap_bridge").Emit(context.Background(), record)
	return nil
}

func (o *otelZapCore) Sync() error { return nil }

// attributeValue keeps a field's type. Zap packs a bool, an integer, a float
// and a duration into Field.Integer and leaves Field.String empty, so reading
// String for every field turns everything but a string into "".
func attributeValue(f zapcore.Field) (log.Value, bool) {
	switch f.Type {

	case zapcore.StringType:
		return log.StringValue(f.String), true

	case zapcore.BoolType:
		return log.BoolValue(f.Integer == 1), true

	case zapcore.Int8Type, zapcore.Int16Type, zapcore.Int32Type, zapcore.Int64Type,
		zapcore.Uint8Type, zapcore.Uint16Type, zapcore.Uint32Type, zapcore.Uint64Type,
		zapcore.UintptrType:
		return log.Int64Value(f.Integer), true

	// Float32 and Float64 are packed with different bit widths.
	case zapcore.Float64Type:
		return log.Float64Value(math.Float64frombits(uint64(f.Integer))), true

	case zapcore.Float32Type:
		return log.Float64Value(float64(math.Float32frombits(uint32(f.Integer)))), true

	case zapcore.DurationType:
		return log.StringValue(time.Duration(f.Integer).String()), true

	// Integer holds the instant, Interface the location.
	case zapcore.TimeType:
		at := time.Unix(0, f.Integer)
		if location, ok := f.Interface.(*time.Location); ok && location != nil {
			at = at.In(location)
		}
		return log.StringValue(at.Format(time.RFC3339Nano)), true

	case zapcore.TimeFullType:
		if at, ok := f.Interface.(time.Time); ok {
			return log.StringValue(at.Format(time.RFC3339Nano)), true
		}

	case zapcore.ByteStringType:
		if raw, ok := f.Interface.([]byte); ok {
			return log.StringValue(string(raw)), true
		}

	case zapcore.ErrorType:
		if err, ok := f.Interface.(error); ok && err != nil {
			return log.StringValue(err.Error()), true
		}

	case zapcore.SkipType, zapcore.NamespaceType:
		return log.Value{}, false
	}

	// Stringer, Reflect, and whatever zap.Any was given.
	return log.StringValue(fmt.Sprint(f.Interface)), true
}

// severityOf maps Zap levels onto the OTel scale, which is what a log backend
// filters on. Without it every record is severity 0, "undefined".
func severityOf(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return log.SeverityFatal
	}
	return log.SeverityUndefined
}
