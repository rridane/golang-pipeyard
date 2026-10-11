package notify

import (
	"fmt"

	"github.com/rridane/golang-pipeyard/x/mod"
)

type TeamsInjector struct {
	WebHook string
}

func (i *TeamsInjector) Inject(stageName string) *mod.TeamsSDK {
	if i.WebHook == "" {
		fmt.Printf("Teams not configured for %s: WebHook is required\n", stageName)
		return nil
	}

	return &mod.TeamsSDK{WebHook: i.WebHook}
}
