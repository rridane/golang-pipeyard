package pi

// The environment a pipeline reads, under names Pipeyard owns. A CI engine
// names these things its own way, and mapping its variables onto these belongs
// in that engine's configuration — so no package here knows what any engine
// calls them.
const (
	// MountDirEnvVar overrides the mount directory, which differs between a
	// laptop run and a containerised one.
	MountDirEnvVar = "PIPEYARD_MOUNT_DIR"

	// ProjectEnvVar and BranchEnvVar say which pipeline a record belongs to.
	ProjectEnvVar = "PIPEYARD_PROJECT"
	BranchEnvVar  = "PIPEYARD_BRANCH"

	// CommitSHAEnvVar, CommitTimeEnvVar and CommitTitleEnvVar identify the
	// change under test. The commit time is what lead time for changes is
	// measured from; the title is what makes a log line say what it was
	// testing.
	CommitSHAEnvVar   = "PIPEYARD_COMMIT_SHA"
	CommitTimeEnvVar  = "PIPEYARD_COMMIT_TIME"
	CommitTitleEnvVar = "PIPEYARD_COMMIT_TITLE"
)
