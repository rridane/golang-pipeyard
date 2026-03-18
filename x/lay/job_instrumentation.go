package lay

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"time"
)

type JobInstrumentation struct {
	Ctx          context.Context
	StartTime    time.Time
	Span         trace.Span
	RunCounter   metric.Int64Counter
	DurationHist metric.Float64Histogram
	Logger       *zap.Logger
	StageName    string
}

// StartNewJobInstrumentation initialise le tracing, la métrique et le logging pour un stage
func StartNewJobInstrumentation(
	ctx context.Context,
	stageName string,
	tracer *trace.Tracer,
	meter metric.Meter,
	logger *zap.Logger,
) *JobInstrumentation {

	start := time.Now()

	// Crée un span de trace
	ctx, span := (*tracer).Start(ctx, stageName+".execute",
		trace.WithAttributes(attribute.String("stage", stageName)),
	)

	// Crée des instruments génériques
	runCounter, _ := meter.Int64Counter("job_total_calls")
	durationHist, _ := meter.Float64Histogram("job_duration_seconds")

	// Log de début
	logger.Info("🚀 Stage démarré", zap.String("stage", stageName))

	return &JobInstrumentation{
		Ctx:          ctx,
		StartTime:    start,
		Span:         span,
		RunCounter:   runCounter,
		DurationHist: durationHist,
		Logger:       logger,
		StageName:    stageName,
	}
}

// End termine proprement le span et enregistre les métriques
func (ji *JobInstrumentation) End(err error) {
	elapsed := time.Since(ji.StartTime)

	status := "success"
	if err != nil {
		status = "error"
		ji.Span.SetStatus(codes.Error, err.Error())
	} else {
		ji.Span.SetStatus(codes.Ok, "OK")
	}

	// Ajout des métriques génériques
	attrs := []attribute.KeyValue{
		attribute.String("stage_name", ji.StageName),
		attribute.String("status", status),
	}
	ji.RunCounter.Add(ji.Ctx, 1, metric.WithAttributes(attrs...))
	ji.DurationHist.Record(ji.Ctx, elapsed.Seconds(), metric.WithAttributes(attrs...))

	// Ajout des attributs dans la trace
	ji.Span.SetAttributes(
		attribute.String("stage_name", ji.StageName),
		attribute.String("status", status),
		attribute.Float64("duration_sec", elapsed.Seconds()),
	)

	// Log final
	if err != nil {
		ji.Logger.Error("❌ Stage échoué",
			zap.String("stage", ji.StageName),
			zap.Float64("duration_sec", elapsed.Seconds()),
			zap.String("trace_id", ji.Span.SpanContext().TraceID().String()),
			zap.Error(err),
		)
	} else {
		ji.Logger.Info("✅ Stage terminé",
			zap.String("stage", ji.StageName),
			zap.Float64("duration_sec", elapsed.Seconds()),
			zap.String("trace_id", ji.Span.SpanContext().TraceID().String()),
		)
	}

	fmt.Printf("⏱️ Durée du stage %s: %.3fs\n", ji.StageName, elapsed.Seconds())
	ji.Span.End()
}
