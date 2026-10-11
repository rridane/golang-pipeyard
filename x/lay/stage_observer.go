package lay

import (
	"strconv"
	"time"

	"context"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

// StageObserver records every stage as one span, one counter sample and one
// duration sample. Wire it into PipelineConfig.Observer and every stage is
// instrumented without a line of code in the stages.
type StageObserver struct {
	Tracer trace.Tracer
	Meter  metric.Meter
	Logger *zap.Logger

	// Flags names the stage flags whose value is recorded. Without them a
	// deploy to staging and one to production land in the same series.
	//
	// Keep them low-cardinality — an environment, a target, a boolean. A
	// version or an image reference creates one metric series per release.
	Flags []string

	// Shutdown flushes the exporters. Wire the injectors' Shutdown here: a
	// batching exporter in a short-lived command drops everything otherwise.
	Shutdown []func(ctx context.Context)
}

func (o *StageObserver) Close(ctx context.Context) error {
	for _, flush := range o.Shutdown {
		flush(ctx)
	}
	return nil
}

func (o *StageObserver) StageStarted(ctx context.Context, stage string, args ru.ParsedArgs) (context.Context, pi.StageFinished) {

	started := time.Now()

	// The commit identity goes on the span only. As a metric label a commit
	// timestamp would create one series per commit.
	series := o.seriesAttributes(stage, args)
	spanAttrs := append(o.commitAttributes(), series...)

	ctx, span := o.tracer().Start(ctx, stage+".execute", trace.WithAttributes(spanAttrs...))
	calls, duration := o.instruments()

	o.logger().Info(o.opening(stage), zap.String("stage", stage))

	return ctx, func(err error) {
		elapsed := time.Since(started)
		status := attribute.String("status", o.status(err))

		calls.Add(ctx, 1, metric.WithAttributes(append(series, status)...))
		duration.Record(ctx, elapsed.Seconds(), metric.WithAttributes(append(series, status)...))

		o.closeSpan(span, err, elapsed, append(spanAttrs, status))
		o.report(stage, err, elapsed, span)
	}
}

// seriesAttributes are recorded on both the span and the metrics, so they must
// stay low-cardinality.
func (o *StageObserver) seriesAttributes(stage string, args ru.ParsedArgs) []attribute.KeyValue {

	attrs := []attribute.KeyValue{attribute.String("stage", stage)}

	// On the series, not only on the resource: a Prometheus store keeps
	// resource attributes out of the labels, so a branch on the resource
	// cannot be filtered on.
	if branch := pi.ReadIdentity().Branch; branch != "" {
		attrs = append(attrs, attribute.String("branch", branch))
	}

	for _, flag := range o.Flags {
		if value, ok := args.StringOk(flag); ok {
			attrs = append(attrs, attribute.String("flag."+flag, value))
			continue
		}
		if value, ok := args.BoolOk(flag); ok {
			attrs = append(attrs, attribute.Bool("flag."+flag, value))
			continue
		}
		if value, ok := args.IntOk(flag); ok {
			attrs = append(attrs, attribute.Int("flag."+flag, value))
			continue
		}
		if value, ok := args.DurationOk(flag); ok {
			attrs = append(attrs, attribute.String("flag."+flag, value.String()))
		}
	}

	return attrs
}

// commitAttributes identify the change under test. They land on the span, which
// is where lead time for changes is computed from.
func (o *StageObserver) commitAttributes() []attribute.KeyValue {

	run := pi.ReadIdentity()

	var attrs []attribute.KeyValue

	if run.CommitSHA != "" {
		attrs = append(attrs, attribute.String("commit_sha", run.CommitSHA))
	}
	if run.CommitTime != "" {
		attrs = append(attrs, attribute.String("commit_time", run.CommitTime))
	}
	if run.CommitTitle != "" {
		attrs = append(attrs, attribute.String("commit_title", run.CommitTitle))
	}
	return attrs
}

// opening names the stage and the change it is running against, so one line
// per run is readable on its own — in a log viewer with its fields collapsed,
// in the runner's console, in a grep.
func (o *StageObserver) opening(stage string) string {
	run := pi.ReadIdentity()

	message := stage + " started"

	if run.Branch != "" {
		message += " on " + run.Branch
	}
	if sha := run.ShortSHA(); sha != "" {
		message += " @" + sha
	}
	if run.CommitTitle != "" {
		message += " " + strconv.Quote(run.CommitTitle)
	}

	return message
}

func (o *StageObserver) closeSpan(span trace.Span, err error, elapsed time.Duration, attrs []attribute.KeyValue) {
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(append(attrs, attribute.Float64("duration_sec", elapsed.Seconds()))...)
	span.End()
}

// report logs the outcome with the trace id, so a line in the CI output can be
// matched to a trace.
func (o *StageObserver) report(stage string, err error, elapsed time.Duration, span trace.Span) {
	fields := []zap.Field{
		zap.String("stage", stage),
		zap.Float64("duration_sec", elapsed.Seconds()),
	}

	// Only when a real tracer is wired, otherwise it is a line of zeroes.
	if span.SpanContext().IsValid() {
		fields = append(fields, zap.String("trace_id", span.SpanContext().TraceID().String()))
	}

	if err != nil {
		o.logger().Error(stage+" failed", append(fields, zap.Error(err))...)
		return
	}
	o.logger().Info(stage+" finished", fields...)
}

func (o *StageObserver) status(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}

// A half-configured observer records what it can instead of failing: a
// pipeline with no OTel endpoint still runs.
func (o *StageObserver) tracer() trace.Tracer {
	if o.Tracer != nil {
		return o.Tracer
	}
	return tracenoop.NewTracerProvider().Tracer("pipeyard")
}

func (o *StageObserver) instruments() (metric.Int64Counter, metric.Float64Histogram) {
	meter := o.Meter
	if meter == nil {
		meter = metricnoop.NewMeterProvider().Meter("pipeyard")
	}

	calls, err := meter.Int64Counter("stage_runs_total")
	if err != nil {
		o.logger().Warn("stage_runs_total unavailable", zap.Error(err))
		calls = metricnoop.Int64Counter{}
	}

	duration, err := meter.Float64Histogram("stage_duration_seconds")
	if err != nil {
		o.logger().Warn("stage_duration_seconds unavailable", zap.Error(err))
		duration = metricnoop.Float64Histogram{}
	}

	return calls, duration
}

func (o *StageObserver) logger() *zap.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return zap.NewNop()
}

var (
	_ pi.StageObserver       = (*StageObserver)(nil)
	_ pi.StageObserverCloser = (*StageObserver)(nil)
)
