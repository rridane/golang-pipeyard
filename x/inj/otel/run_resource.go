package otel

import (
	"context"

	"github.com/rridane/golang-pipeyard/pi"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// runResource describes the run every record comes from, so a stage passes
// nothing and a log line still says where it came from.
//
// withChange is false for metrics and for traces. A resource attribute becomes
// a label in a Prometheus store, and a commit as a label is one series per
// commit — including through a spanmetrics connector, which copies a trace's
// resource into the metrics it derives from it. The change rides the span
// instead, where the observer puts it.
func runResource(ctx context.Context, serviceName string, withChange bool) *resource.Resource {

	run := pi.ReadIdentity()

	attrs := []attribute.KeyValue{semconv.ServiceName(serviceName)}

	// service.namespace is the group a service belongs to, which is the
	// project. Loki indexes it, so streams separate with no configuration.
	if run.Project != "" {
		attrs = append(attrs, semconv.ServiceNamespaceKey.String(run.Project))
	}

	if run.Branch != "" {
		attrs = append(attrs, attribute.String("branch", run.Branch))
	}

	if withChange {
		if run.CommitSHA != "" {
			attrs = append(attrs, attribute.String("commit_sha", run.CommitSHA))
		}
		if run.CommitTitle != "" {
			attrs = append(attrs, attribute.String("commit_title", run.CommitTitle))
		}
	}

	res, _ := resource.New(ctx, resource.WithAttributes(attrs...))
	return res
}
