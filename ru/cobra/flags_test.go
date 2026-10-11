package cobra

import (
	"testing"
	"time"

	"github.com/rridane/golang-pipeyard/ru"
	"github.com/spf13/cobra"
)

// declaredDefaultsReachTheStage is the round trip that was broken: a default
// declared in Configure must be the value the stage actually receives.
func TestRoundTrip_DeclaredDefaultsReachTheStage(t *testing.T) {
	specs := []ru.Flag{
		{Name: "pkg", Type: ru.StringFlag, Default: "./..."},
		{Name: "jobs", Type: ru.IntFlag, Default: 4},
		{Name: "race", Type: ru.BoolFlag, Default: true},
		{Name: "timeout", Type: ru.DurationFlag, Default: 2 * time.Minute},
	}

	args := declareThenParse(t, specs)

	if got := args.String("pkg"); got != "./..." {
		t.Errorf("pkg: want %q, got %q", "./...", got)
	}
	if got := args.Int("jobs"); got != 4 {
		t.Errorf("jobs: want 4, got %d", got)
	}
	if got := args.Bool("race"); got != true {
		t.Errorf("race: want true, got %v", got)
	}
	if got := args.Duration("timeout"); got != 2*time.Minute {
		t.Errorf("timeout: want 2m0s, got %v", got)
	}
}

// A spec may state its defaults as strings; the stage must still get the value.
func TestRoundTrip_StringDefaultsAreCoerced(t *testing.T) {
	specs := []ru.Flag{
		{Name: "jobs", Type: ru.IntFlag, Default: "4"},
		{Name: "race", Type: ru.BoolFlag, Default: "true"},
		{Name: "timeout", Type: ru.DurationFlag, Default: "120s"},
	}

	args := declareThenParse(t, specs)

	if got := args.Int("jobs"); got != 4 {
		t.Errorf("jobs: want 4, got %d", got)
	}
	if got := args.Bool("race"); got != true {
		t.Errorf("race: want true, got %v", got)
	}
	if got := args.Duration("timeout"); got != 2*time.Minute {
		t.Errorf("timeout: want 2m0s, got %v", got)
	}
}

// A default that cannot be read must be reported, never silently dropped to zero.
func TestDeclare_UnreadableDefaultIsReported(t *testing.T) {
	cases := map[string]ru.Flag{
		"int":      {Name: "jobs", Type: ru.IntFlag, Default: "four"},
		"bool":     {Name: "race", Type: ru.BoolFlag, Default: "yes please"},
		"duration": {Name: "timeout", Type: ru.DurationFlag, Default: "2 minutes"},
		"unknown":  {Name: "path", Type: ru.FlagType("filepath"), Default: "/tmp"},
	}

	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Fatal("an unreadable default must panic at declaration")
				} else {
					t.Logf("reported: %v", recovered)
				}
			}()
			declareFlags(&cobra.Command{Use: "x"}, []ru.Flag{spec})
		})
	}
}

func declareThenParse(t *testing.T, specs []ru.Flag) ru.ParsedArgs {
	t.Helper()

	command := &cobra.Command{Use: "test"}
	declareFlags(command, specs)

	if err := command.Flags().Parse(nil); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	args, err := parseFlagsToParsedArgs(command, specs)
	if err != nil {
		t.Fatalf("parseFlagsToParsedArgs: %v", err)
	}
	return args
}
