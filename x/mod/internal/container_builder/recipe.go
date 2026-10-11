package container_builder

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

// Dockerfile is a build recipe, either raw or parameterised.
//
//	Args nil     → Template is rendered as is, with no interpretation
//	Args not nil → Template is treated as a text/template
//
// Template indentation is free: it is normalised on render, so the recipe can
// be written aligned with the Go code that carries it.
type Dockerfile struct {
	Template string
	Args     any
}

// Render produces a valid Dockerfile, usable as is with `docker build`.
func (d Dockerfile) Render() string {
	body := d.body()

	if d.Args == nil {
		return body
	}

	tmpl, err := template.New("dockerfile").Parse(body)
	if err != nil {
		panic(fmt.Errorf("invalid dockerfile template: %w", err))
	}

	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, d.Args); err != nil {
		panic(fmt.Errorf("render dockerfile template: %w", err))
	}
	return strings.TrimSpace(rendered.String())
}

// String lets one recipe be injected into another's template.
// That is the whole composition mechanism.
func (d Dockerfile) String() string { return d.Render() }

// Stages delegates parsing to BuildKit, the only authority on the syntax.
func (d Dockerfile) Stages() ([]instructions.Stage, error) {
	ast, err := parser.Parse(strings.NewReader(d.Render() + "\n"))
	if err != nil {
		return nil, fmt.Errorf("unreadable dockerfile: %w", err)
	}
	stages, _, err := instructions.Parse(ast.AST, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid dockerfile: %w", err)
	}
	return stages, nil
}

// body strips the template's common indentation.
func (d Dockerfile) body() string {
	lines := strings.Split(strings.Trim(d.Template, "\n"), "\n")

	indent := -1
	for _, l := range lines {
		content := strings.TrimLeft(l, " \t")
		if content == "" {
			continue
		}
		if n := len(l) - len(content); indent < 0 || n < indent {
			indent = n
		}
	}
	if indent <= 0 {
		return strings.TrimSpace(strings.Join(lines, "\n"))
	}

	for i, l := range lines {
		if len(l) >= indent {
			lines[i] = l[indent:]
		} else {
			lines[i] = strings.TrimLeft(l, " \t")
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
