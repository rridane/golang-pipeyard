package stack

import (
	"strings"
	"testing"
)

// Every step has to run even when an earlier one failed, and the script has
// to end non-zero all the same. A check that stops at the first failure makes
// its reader run it again to see the rest.
func TestScript_RunsEveryStepAndKeepsTheFailure(t *testing.T) {
	script := new(Script).
		Step("first --check").
		Step("second --check %s", Quote("a path")).
		String()

	for _, want := range []string{
		"status=0\n",
		"first --check || status=1\n",
		"second --check 'a path' || status=1\n",
		"exit $status\n",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("the script should contain %q:\n%s", want, script)
		}
	}

	if strings.Contains(script, "&&") {
		t.Fatalf("a step must not be chained with the next:\n%s", script)
	}
}

// The output has to say which command produced what, without the stage
// labelling it by hand.
func TestScript_AnnouncesEachStep(t *testing.T) {
	script := new(Script).Step("ruff check").String()

	if !strings.Contains(script, "echo '== ruff check'\n") {
		t.Fatalf("the step should announce itself:\n%s", script)
	}
}

func TestScript_WithNoStepSucceeds(t *testing.T) {
	if script := new(Script).String(); script != "status=0\nexit $status\n" {
		t.Fatalf("an empty script should just succeed, got %q", script)
	}
}

func TestQuote_SurvivesAQuoteInTheValue(t *testing.T) {
	if got := Quote("it's here"); got != `'it'\''s here'` {
		t.Fatalf("got %s", got)
	}
}
