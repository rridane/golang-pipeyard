package pi

import "os"

// Identity is what a run is: which project, which branch, which commit. A
// stage receives it by declaring it as a dependency, like anything else.
type Identity struct {
	Project    string
	Branch     string
	CommitSHA  string
	CommitTime string
}

// ReadIdentity resolves the identity from the environment. It is exported so
// that an injector, an observer and a stage all describe the same run.
func ReadIdentity() Identity {
	return Identity{
		Project:    os.Getenv(ProjectEnvVar),
		Branch:     os.Getenv(BranchEnvVar),
		CommitSHA:  os.Getenv(CommitSHAEnvVar),
		CommitTime: os.Getenv(CommitTimeEnvVar),
	}
}
