package golang

import (
	"testing"
	"time"

	"github.com/rridane/golang-pipeyard/pi"
	"github.com/rridane/golang-pipeyard/ru"
)

func TestTestStage_Command(t *testing.T) {
	stage := &TestStage{}

	args := ru.ParsedArgs{}
	args.SetString("pkg", "./pi/")
	args.SetBool("race", true)
	args.SetDuration("timeout", 5*time.Minute)

	if got, want := stage.command(args), "go test -timeout 5m0s -race ./pi/"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	args.SetBool("race", false)

	if got, want := stage.command(args), "go test -timeout 5m0s ./pi/"; got != want {
		t.Errorf("without the race detector: got %q, want %q", got, want)
	}
}

func TestBuildStage_Command(t *testing.T) {
	args := ru.ParsedArgs{}
	args.SetString("pkg", "./...")

	if got, want := (&BuildStage{}).command(args), "go build ./..."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVetStage_Command(t *testing.T) {
	args := ru.ParsedArgs{}
	args.SetString("pkg", "./x/...")

	if got, want := (&VetStage{}).command(args), "go vet ./x/..."; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// gofmt takes the directories one after another, and each is quoted: a
// directory with a space in its name would otherwise be read as two.
func TestFormatStage_Command(t *testing.T) {
	stage := &FormatStage{}

	if got, want := stage.command([]string{"pi", "ru", "x"}), "gofmt -l 'pi' 'ru' 'x'"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := stage.command([]string{"my sources"}), "gofmt -l 'my sources'"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A field of the struct is the default of a flag, which is the whole contract
// between how a repository wires a stage and what its command line offers.
func TestStages_FieldsBecomeFlagDefaults(t *testing.T) {
	test := (&TestStage{Packages: "./pi/...", Timeout: "90s", WithoutRace: true}).Configure()

	for _, tc := range []struct {
		flag string
		want any
	}{
		{"pkg", "./pi/..."},
		{"timeout", "90s"},
		{"race", false},
	} {
		flag, found := test.Flag(tc.flag)
		if !found {
			t.Fatalf("no flag %q", tc.flag)
		}
		if flag.Default != tc.want {
			t.Errorf("--%s defaults to %v, want %v", tc.flag, flag.Default, tc.want)
		}
	}
}

// Left empty, every stage still offers a usable command line: a stack that
// has to be configured before it runs is a stack nobody tries.
func TestStages_ZeroValueIsUsable(t *testing.T) {
	for _, stage := range []pi.Stage{&BuildStage{}, &VetStage{}, &TestStage{}} {
		spec := stage.Configure()

		flag, found := spec.Flag("pkg")
		if !found {
			t.Errorf("%s: no --pkg", spec.Command())
			continue
		}
		if flag.Default != DefaultPackages {
			t.Errorf("%s: --pkg defaults to %v, want %q", spec.Command(), flag.Default, DefaultPackages)
		}
	}
}

// The race detector is on unless the repository says otherwise, because the
// bugs it finds are the ones review does not.
func TestTestStage_RaceIsOnByDefault(t *testing.T) {
	flag, found := (&TestStage{}).Configure().Flag("race")
	if !found {
		t.Fatal("no --race")
	}
	if flag.Default != true {
		t.Errorf("--race defaults to %v, want true", flag.Default)
	}
}
