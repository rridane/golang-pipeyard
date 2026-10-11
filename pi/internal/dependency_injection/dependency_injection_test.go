package dependency_injection

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type Budget struct{ Max int }

// an injector returning a value, not a pointer
type budgetInjector struct{}

func (i *budgetInjector) Inject(stageName string) Budget { return Budget{Max: 10} }

// an injector that cannot configure itself
type unconfiguredInjector struct{}

func (i *unconfiguredInjector) Inject(stageName string) *Budget { return nil }

type valueStage struct{}

func (s *valueStage) Execute(ctx context.Context, args struct{}, b Budget) error { return nil }

type pointerStage struct{}

func (s *pointerStage) Execute(ctx context.Context, args struct{}, b *Budget) error { return nil }

// An injector returning a value used to panic here: reflect.Value.IsNil is
// illegal on a struct, and the resolution loop called it unconditionally.
func TestGetDependencies_ValueReturningInjectorIsUsable(t *testing.T) {
	deps, err := GetDependencies(
		executeOf(t, &valueStage{}),
		[]reflect.Value{injectOf(t, &budgetInjector{})},
		"stage",
	)
	if err != nil {
		t.Fatalf("a value-returning injector must resolve, got: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("want 1 dependency, got %d", len(deps))
	}
	if got := deps[0].Interface().(Budget); got.Max != 10 {
		t.Fatalf("want Max 10, got %d", got.Max)
	}
}

// An injector whose configuration is incomplete returns nil, and the stage
// must be told what to do about it rather than receive a nil dependency.
func TestGetDependencies_NilInjectorIsReportedWithItsCause(t *testing.T) {
	_, err := GetDependencies(
		executeOf(t, &pointerStage{}),
		[]reflect.Value{injectOf(t, &unconfiguredInjector{})},
		"deploy",
	)
	if err == nil {
		t.Fatal("an injector returning nil must not pass for a resolved dependency")
	}
	for _, want := range []string{`"deploy"`, "PipelineConfig.Injectors", "returned nil"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}
}

func TestGetDependencies_PointerInjectorResolves(t *testing.T) {
	deps, err := GetDependencies(
		executeOf(t, &pointerStage{}),
		[]reflect.Value{reflect.ValueOf(func(stageName string) *Budget {
			return &Budget{Max: 42}
		})},
		"stage",
	)
	if err != nil {
		t.Fatal(err)
	}
	if deps[0].Interface().(*Budget).Max != 42 {
		t.Fatal("the injected pointer must reach the stage")
	}
}

func executeOf(t *testing.T, stage any) reflect.Method {
	t.Helper()
	method, ok := reflect.TypeOf(stage).MethodByName("Execute")
	if !ok {
		t.Fatal("test stage has no Execute method")
	}
	return method
}

func injectOf(t *testing.T, injector any) reflect.Value {
	t.Helper()
	method, err := ValidateDependencyInjector(injector)
	if err != nil {
		t.Fatalf("test injector rejected: %v", err)
	}
	return method
}
