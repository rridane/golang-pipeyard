package sys

import "github.com/rridane/golang-pipeyard/pi"

// IdentityInjector gives a stage the run it is part of: which project, which
// branch, which commit. It reads the environment, under the names pi declares.
type IdentityInjector struct{}

func (i *IdentityInjector) Inject(stageName string) pi.Identity {
	return pi.ReadIdentity()
}
