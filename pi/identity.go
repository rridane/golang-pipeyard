package pi

import (
	"os"
	"strings"
)

// Identity is what a run is: which project, which branch, which commit. A
// stage receives it by declaring it as a dependency, like anything else.
type Identity struct {
	Project     string
	Branch      string
	CommitSHA   string
	CommitTime  string
	CommitTitle string
}

// ShortSHA is the commit as a human writes it.
func (i Identity) ShortSHA() string {
	if len(i.CommitSHA) < 7 {
		return i.CommitSHA
	}
	return i.CommitSHA[:7]
}

// ReadIdentity resolves the identity from the environment. It is exported so
// that an injector, an observer and a stage all describe the same run.
func ReadIdentity() Identity {
	return Identity{
		Project:    os.Getenv(ProjectEnvVar),
		Branch:     os.Getenv(BranchEnvVar),
		CommitSHA:  os.Getenv(CommitSHAEnvVar),
		CommitTime: os.Getenv(CommitTimeEnvVar),
		// A commit message is a title and then a body. Only the title belongs
		// on a log line.
		CommitTitle: firstLine(os.Getenv(CommitTitleEnvVar)),
	}
}

func firstLine(message string) string {
	title, _, _ := strings.Cut(strings.TrimSpace(message), "\n")
	return strings.TrimSpace(title)
}
