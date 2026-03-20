package pi

type Ctx struct {
	// MountDir is ./mounts on local and /mounts in pipeyard container
	MountDir   string
	CurrentDir string
}
