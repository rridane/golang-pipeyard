package pi

import "testing"

// A commit message is a title and a body. Only the title belongs on a log line.
func TestReadIdentity_KeepsOnlyTheCommitTitle(t *testing.T) {
	t.Setenv(CommitTitleEnvVar, "keep the release id instead of its tag\n\nThe draft was not addressable.\n")

	if got, want := ReadIdentity().CommitTitle, "keep the release id instead of its tag"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestIdentity_ShortSHA(t *testing.T) {
	for sha, want := range map[string]string{
		"e8d9f62a41caa0a259da521a0d84cf5ab85faf97": "e8d9f62",
		"abc": "abc",
		"":    "",
	} {
		if got := (Identity{CommitSHA: sha}).ShortSHA(); got != want {
			t.Errorf("ShortSHA(%q) = %q, want %q", sha, got, want)
		}
	}
}
