package pipeline_provider

import (
	"context"
	"github.com/spf13/cobra"
	"reflect"
	"testing"
)

type ServiceA struct{ Name string }
type ServiceB struct{ ID int }

type InjectorA struct{ called bool }

func (i *InjectorA) Inject(stageName string) *ServiceA {
	i.called = true
	return &ServiceA{Name: "InjectedA:" + stageName}
}

type InjectorB struct{ called bool }

func (i *InjectorB) Inject(stageName string) *ServiceB {
	i.called = true
	return &ServiceB{ID: 42}
}

type InjectorNil struct{}

func (i *InjectorNil) Inject(stageName string) *ServiceA { return nil }

type StageWithDeps struct{}

func (s *StageWithDeps) Configure(cmd *cobra.Command) {

}

func (s *StageWithDeps) Execute(_ context.Context, _ *cobra.Command, a *ServiceA, b *ServiceB) error {
	return nil
}

func TestGetDependencies_Success(t *testing.T) {
	stage := &StageWithDeps{}
	method, found := getExecuteMethodType(stage)
	if !found {
		t.Fatal("Méthode Execute introuvable")
	}

	injA := &InjectorA{}
	injB := &InjectorB{}
	injectors := []reflect.Value{
		reflect.ValueOf(injA).MethodByName("Inject"),
		reflect.ValueOf(injB).MethodByName("Inject"),
	}

	args, err := getDependencies(method, injectors, "test-stage")
	if err != nil {
		t.Fatalf("Erreur inattendue: %v", err)
	}

	if len(args) != 2 {
		t.Fatalf("Attendu 2 dépendances injectées, obtenu %d", len(args))
	}

	// Vérifie que les bons types ont été injectés
	if _, ok := args[0].Interface().(*ServiceA); !ok {
		t.Error("Première dépendance n’est pas du type *ServiceA")
	}
	if _, ok := args[1].Interface().(*ServiceB); !ok {
		t.Error("Deuxième dépendance n’est pas du type *ServiceB")
	}

	// Vérifie que les bons injecteurs ont été appelés
	if !injA.called || !injB.called {
		t.Error("Tous les injecteurs n’ont pas été appelés")
	}
}

type StageMissing struct{}

func (s *StageMissing) Execute(_ context.Context, _ *cobra.Command, a *ServiceA, b *ServiceB) error {
	return nil
}

func (s *StageMissing) Configure(cmd *cobra.Command) {
}

func TestGetDependencies_MissingDependency(t *testing.T) {

	stage := &StageMissing{}
	method, found := getExecuteMethodType(stage)
	if !found {
		t.Fatal("Méthode Execute introuvable")
	}

	injA := &InjectorA{} // Pas de B
	injectors := []reflect.Value{reflect.ValueOf(injA).MethodByName("Inject")}

	args, err := getDependencies(method, injectors, "stage-test")

	if err == nil {
		t.Fatal("Erreur attendue pour dépendance manquante")
	}
	if len(args) != 0 {
		t.Errorf("Aucune dépendance ne devrait être injectée, obtenu %d", len(args))
	}
}

type StageWithOneDep struct{}

func (s *StageWithOneDep) Execute(_ context.Context, _ *cobra.Command, a *ServiceA) error { return nil }

func (s *StageWithOneDep) Configure(cmd *cobra.Command) {
}

func TestGetDependencies_IgnoreNilInjector(t *testing.T) {

	stage := &StageWithOneDep{}
	method, _ := getExecuteMethodType(stage)

	injNil := &InjectorNil{}
	injectors := []reflect.Value{reflect.ValueOf(injNil).MethodByName("Inject")}

	args, err := getDependencies(method, injectors, "stage-nil")

	if err == nil {
		t.Fatal("Erreur attendue pour dépendance non injectée")
	}
	if len(args) != 0 {
		t.Errorf("Aucune dépendance ne devrait être injectée, obtenu %d", len(args))
	}
}
