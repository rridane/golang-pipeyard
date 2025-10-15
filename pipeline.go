package pipeline_provider

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
)

type Pipeline struct {
	rootCmd   *cobra.Command
	injectors []any
}

// PipelineConfig configure l'application
type PipelineConfig struct {
	Name             string
	ShortDescription string
	LongDescription  string
	Version          string
	Injectors        []any
}

// RegisterStage enregistre un stage
func (a *Pipeline) RegisterStage(name string, stage CommandStage) *Pipeline {

	cmd := &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("Exécute la commande %s", name),
		RunE:  a.createRunE(name, stage),
	}

	stage.Configure(cmd)
	a.rootCmd.AddCommand(cmd)
	return a
}

func (a *Pipeline) createRunE(stageName string, stage CommandStage) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		var injectorMethods []reflect.Value

		injectorMethods = a.validateInjectorAndGetMethods()

		m, found := getExecuteMethodType(stage)

		if !found {
			return nil
		}

		var funcArgs []reflect.Value

		funcArgs = append(funcArgs, reflect.ValueOf(ctx))
		funcArgs = append(funcArgs, reflect.ValueOf(cmd))
		dependencies, err := getDependencies(m, injectorMethods, stageName)

		if err != nil {
			return err
		}

		funcArgs = append(funcArgs, dependencies...)

		callExecuteCommand(stage, funcArgs)

		return nil
	}
}

func (a *Pipeline) Run() {
	if err := a.rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "❌ Erreur: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

// New crée un nouveau pipeline
func New(config PipelineConfig) *Pipeline {

	rootCmd := &cobra.Command{
		Use:     config.Name,
		Short:   config.ShortDescription,
		Long:    config.LongDescription,
		Version: config.Version,
	}

	app := &Pipeline{
		rootCmd:   rootCmd,
		injectors: config.Injectors,
	}

	return app
}
