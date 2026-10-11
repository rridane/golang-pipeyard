package inj

import (
	"github.com/rridane/golang-pipeyard/x/mod"
)

type MailerInjector struct {
	Config mod.MailConfig
}

func (i *MailerInjector) Inject(stageName string) *mod.MailSender {
	return mod.NewMailSender(i.Config)
}
