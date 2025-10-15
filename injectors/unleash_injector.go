package injectors

import (
	"fmt"
	"github.com/Unleash/unleash-go-sdk/v5"
	"net/http"
)

type UnleashInjector struct {
	ServiceName     string
	UnleashURL      string
	UnleashAPIToken string
	UnleashAppName  string
	BlockUntilReady bool
}

func (u *UnleashInjector) Inject(stageName string) *unleash.Client {
	if u.UnleashURL == "" || u.UnleashAPIToken == "" {
		fmt.Printf("❌ UNLEASH_URL ou UNLEASH_API_TOKEN manquant pour %s\n", stageName)
		return nil
	}

	app := u.UnleashAppName
	if app == "" {
		app = u.ServiceName
	}

	client, err := unleash.NewClient(
		unleash.WithAppName(app),
		unleash.WithUrl(u.UnleashURL),
		unleash.WithCustomHeaders(http.Header{
			"Authorization": []string{u.UnleashAPIToken},
		}),
		unleash.WithListener(&unleash.DebugListener{}),
	)
	if err != nil {
		fmt.Printf("⚠️ Impossible d’initialiser Unleash: %v\n", err)
		return nil
	}

	if u.BlockUntilReady {
		client.WaitForReady()
	}

	return client
}
