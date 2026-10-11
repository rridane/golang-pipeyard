package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
	"github.com/rridane/golang-pipeyard/x/mod"
	"go.uber.org/zap"
)

type ReleaseStage struct{}

func (s *ReleaseStage) Configure() *pi.StageSpec {
	spec := pi.NewStageSpec("release")
	spec.SetDescription("Build the pipeline image and push it")
	spec.AddFlag("version", ru.StringFlag, "semver version, without a leading v", "", true)
	spec.AddFlag("image", ru.StringFlag, "image repository", "ghcr.io/rridane/pipeyard-ci", false)
	spec.AddFlag("push", ru.BoolFlag, "push to the registry", false, false)
	spec.AddFlag("insecure", ru.BoolFlag, "registry without TLS", false, false)
	return spec
}

func (s *ReleaseStage) Execute(ctx context.Context, args ru.ParsedArgs, cb *mod.ContainerBuilder, log *zap.Logger) error {
	release, err := newRelease(args)
	if err != nil {
		return err
	}

	image, err := cb.Build(ctx, mod.BuildRequest{
		Dockerfile: release.recipe(),
		ContextDir: ".",
		Target:     "runtime",
		BuildArgs:  map[string]string{"VERSION": release.version},
	})
	if err != nil {
		return fmt.Errorf("build the pipeline image: %w", err)
	}

	if err := image.ExtractTo(ctx, "/usr/local/bin/pipeyard-ci", "./dist"); err != nil {
		return fmt.Errorf("extract the binary: %w", err)
	}
	log.Info("binary written", zap.String("path", "./dist/pipeyard-ci"))

	if !release.push {
		log.Info("image built, not pushed", zap.Strings("tags", release.tags()))
		return nil
	}

	// The exact version first, then the floating tag. The second push reuses
	// the cache, so moving :1 costs nothing once :1.4.0 is up.
	for _, tag := range release.tags() {
		if err := release.publish(ctx, image, tag); err != nil {
			return err
		}
		log.Info("pushed", zap.String("tag", tag))
	}
	return nil
}

type release struct {
	version  string
	image    string
	push     bool
	insecure bool
}

func newRelease(args ru.ParsedArgs) (release, error) {
	version := strings.TrimPrefix(args.String("version"), "v")

	if strings.Count(version, ".") != 2 {
		return release{}, fmt.Errorf("version %q is not a semver x.y.z", args.String("version"))
	}

	return release{
		version:  version,
		image:    args.String("image"),
		push:     args.Bool("push"),
		insecure: args.Bool("insecure"),
	}, nil
}

// tags pairs the exact version with its major, so consumers can pin x.y.z or
// follow the floating tag and get the next release without a change.
func (r release) tags() []string {
	major := strings.SplitN(r.version, ".", 2)[0]
	return []string{
		r.image + ":" + r.version,
		r.image + ":" + major,
	}
}

func (r release) publish(ctx context.Context, image *mod.Artifact, tag string) error {
	if r.insecure {
		return image.PushInsecure(ctx, tag)
	}
	return image.Push(ctx, tag)
}

// recipe carries Go in the runtime stage on purpose: the fmt, vet and test
// stages shell out to the toolchain, so the image has to provide it.
func (r release) recipe() mod.Dockerfile {
	return mod.Dockerfile{
		Args: r,
		Template: `
			FROM golang:1.24-alpine AS builder
			ARG VERSION
			WORKDIR /src
			COPY go.mod go.sum ./
			RUN --mount=type=cache,target=/go/pkg/mod go mod download
			COPY . .
			RUN --mount=type=cache,target=/go/pkg/mod \
			    --mount=type=cache,target=/root/.cache/go-build \
			    CGO_ENABLED=0 go build -trimpath \
			      -ldflags "-s -w -X main.version=${VERSION}" \
			      -o /out/pipeyard-ci ./ci

			FROM golang:1.24-alpine AS runtime
			RUN adduser -S -D -H ci
			COPY --from=builder /out/pipeyard-ci /usr/local/bin/pipeyard-ci
			USER ci
			WORKDIR /workspace
			LABEL org.opencontainers.image.source="https://github.com/rridane/golang-pipeyard"
			LABEL org.opencontainers.image.version="{{.Version}}"
			ENTRYPOINT ["pipeyard-ci"]
		`,
	}
}

// Version is read by the template above.
func (r release) Version() string { return r.version }
