package pi

import "os"

// MountDirEnvVar overrides the mount directory, which differs between a laptop
// run and a containerised one.
const MountDirEnvVar = "PIPEYARD_MOUNT_DIR"

const defaultMountDir = "./mounts"

// Ctx is the pipeline's view of the filesystem it runs on. A stage receives it
// by declaring it as a dependency, like any other.
type Ctx struct {
	MountDir   string
	CurrentDir string
}

// newCtx resolves the directories once, at pipeline creation.
func newCtx() Ctx {
	mounts := os.Getenv(MountDirEnvVar)
	if mounts == "" {
		mounts = defaultMountDir
	}

	current, err := os.Getwd()
	if err != nil {
		current = "."
	}

	return Ctx{MountDir: mounts, CurrentDir: current}
}

// ctxInjector hands the pipeline context to any stage that asks for it. It is
// an ordinary injector, so it goes through the same resolution as the rest.
type ctxInjector struct {
	ctx Ctx
}

func (i *ctxInjector) Inject(stageName string) Ctx { return i.ctx }
