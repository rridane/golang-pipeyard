// pipeline_test.go
package pipeline_provider

import (
	"context"
	"errors"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/metric/noop"
	"testing"
)

type MockStage struct {
	configureCalled bool
	executeCalled   bool
	shouldError     bool
}

func (m *MockStage) Configure(cmd *cobra.Command) {
	m.configureCalled = true
	cmd.Use = "mock"
}

type MockStageWithoutCmdUse struct {
	configureCalled bool
	executeCalled   bool
	shouldError     bool
}

func (m *MockStageWithoutCmdUse) Configure(cmd *cobra.Command) {
	m.configureCalled = true
}

func (m *MockStage) Execute(ctx context.Context, cmd *cobra.Command) error {
	m.executeCalled = true
	if m.shouldError {
		return errors.New("mock execution error")
	}
	return nil
}

func (m *MockStageWithoutCmdUse) Execute(ctx context.Context, cmd *cobra.Command) error {
	m.executeCalled = true
	if m.shouldError {
		return errors.New("mock execution error")
	}
	return nil
}

func TestNewPipelineWorks(t *testing.T) {
	config := PipelineConfig{
		Name:             "test",
		ShortDescription: "Test pipeline",
		LongDescription:  "Long test pipeline description",
		Version:          "1.0.0",
		Injectors:        []any{},
	}

	pipeline := New(config)

	if pipeline == nil {
		t.Fatal("New() returned nil pipeline")
	}

	if pipeline.rootCmd == nil {
		t.Error("Pipeline root command is nil")
	}
}

func TestRegisterStageHasCmdName(t *testing.T) {
	pipeline := New(PipelineConfig{Name: "test"})
	stage := &MockStage{}

	// Test registration
	result := pipeline.RegisterStage("test", stage)

	if result != pipeline {
		t.Error("RegisterStage() should return the pipeline for chaining")
	}

	found := false

	for _, cmd := range pipeline.rootCmd.Commands() {
		if cmd.Name() == "test" {
			found = true
			break
		}
	}

	withoutCmdUse := &MockStageWithoutCmdUse{}

	result.RegisterStage("test_without_cmd_use", withoutCmdUse)

	if result != pipeline {
		t.Error("RegisterStage() should return the pipeline for chaining")
	}

	found = false

	for _, cmd := range pipeline.rootCmd.Commands() {
		if cmd.Name() == "test_without_cmd_use" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Stage command was not registered")
	}
}

func TestPipelineRun(t *testing.T) {

	pipeline := New(PipelineConfig{Name: "test"})

	pipeline.rootCmd.SetArgs([]string{"--help"})

	pipeline.Run()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Pipeline.Run() panicked: %v", r)
		}
	}()

}

func TestCreateRunE(t *testing.T) {
	pipeline := New(PipelineConfig{
		Name:      "test",
		Injectors: []any{},
	})

	stage := &MockStage{}
	runE := pipeline.createRunE("test", stage)

	cmd := &cobra.Command{}
	args := []string{}

	err := runE(cmd, args)

	if err != nil {
		t.Errorf("createRunE() returned error: %v", err)
	}

	if !stage.executeCalled {
		t.Error("Stage Execute was not called")
	}
}

type MockStageWithInjectors struct {
	configureCalled bool
	executeCalled   bool
	shouldError     bool
}

func (m *MockStageWithInjectors) Configure(cmd *cobra.Command) {
	m.configureCalled = true
	cmd.Use = "mock"
}

func (m *MockStageWithInjectors) Execute(ctx context.Context, cmd *cobra.Command, test *noop.Float64Counter) error {
	m.executeCalled = true
	if m.shouldError {
		return errors.New("mock execution error")
	}
	return nil
}

func TestCreateRunE_WithErrorBecauseOfInjectors(t *testing.T) {
	pipeline := New(PipelineConfig{
		Name:      "test",
		Injectors: []any{},
	})

	stage := &MockStageWithInjectors{}
	runE := pipeline.createRunE("test", stage)

	cmd := &cobra.Command{}
	args := []string{}

	err := runE(cmd, args)

	if err == nil {
		t.Error("Expected error from stage execution, got nil")
	}
}
