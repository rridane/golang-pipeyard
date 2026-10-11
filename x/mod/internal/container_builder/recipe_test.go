package container_builder

import (
	"strings"
	"testing"
)

func TestRender_WithoutArgsKeepsTemplateAsIs(t *testing.T) {
	recipe := Dockerfile{Template: "\t\tFROM alpine:3.20\n\t\tRUN echo '{{ pas un tmpl }}'"}

	rendered := recipe.Render()

	if !strings.Contains(rendered, "{{ pas un tmpl }}") {
		t.Fatalf("Args nil must render the template with no interpretation, got:\n%s", rendered)
	}
	if strings.HasPrefix(rendered, "\t") {
		t.Fatalf("common indentation must be stripped, got:\n%q", rendered)
	}
}

func TestRender_WithArgsInterpolates(t *testing.T) {
	type params struct{ Base string }

	rendered := Dockerfile{
		Args:     params{Base: "alpine:3.20"},
		Template: "FROM {{.Base}}",
	}.Render()

	if rendered != "FROM alpine:3.20" {
		t.Fatalf("want %q, got %q", "FROM alpine:3.20", rendered)
	}
}

func TestStages_DelegatesParsingToBuildKit(t *testing.T) {
	recipe := Dockerfile{Template: `
		FROM golang:1.24 AS builder
		RUN go build ./...

		FROM alpine:3.20 AS runtime
		COPY --from=builder /app /app
	`}

	stages, err := recipe.Stages()
	if err != nil {
		t.Fatal(err)
	}
	if len(stages) != 2 {
		t.Fatalf("want 2 stages, got %d", len(stages))
	}
	if stages[0].Name != "builder" || stages[1].Name != "runtime" {
		t.Fatalf("unexpected stage names: %q %q", stages[0].Name, stages[1].Name)
	}
}

func TestComposition_RecipeInjectsIntoAnother(t *testing.T) {
	builderStage := Dockerfile{Template: "FROM golang:1.24 AS builder\nRUN go build ./..."}
	runtimeStage := Dockerfile{Template: "FROM alpine:3.20 AS runtime\nUSER nobody"}

	assembled := Dockerfile{
		Template: "{{.Builder}}\n\n{{.Runtime}}",
		Args:     map[string]any{"Builder": builderStage, "Runtime": runtimeStage},
	}

	stages, err := assembled.Stages()
	if err != nil {
		t.Fatal(err)
	}
	if len(stages) != 2 {
		t.Fatalf("composition must produce 2 stages, got %d", len(stages))
	}
}
