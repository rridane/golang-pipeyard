package main

import (
	pipeline_provider "github.com/rridane/golang-pipewire"
	injectors "github.com/rridane/golang-pipewire/injectors"
)

func main() {

	pipe := pipeline_provider.New(pipeline_provider.PipelineConfig{
		Name:             "ci_pipeline",
		ShortDescription: "CI CLI – All stages",
		LongDescription:  "Interface CLI moderne pour gérer les pipelines ci",
		Version:          "1.0.0",
		Injectors:        []any{&injectors.DaggerInjector{}, &injectors.TracerInjector{}, &injectors.LoggerInjector{}, &injectors.K8sInjector{}},
	})

	pipe.
		RegisterStage("test", &TestSTage{}).
		RegisterStage("build", &BuildStage{}).
		RegisterStage("build", &BuildStage{}).
		RegisterStage("deploy", &DeployStage{
			defaultNamespace: "rct",
			defaultReplicas:  2,
		})
	pipe.Run()
}
