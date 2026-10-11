package featureflag

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
		fmt.Printf("Unleash not configured for %s: UNLEASH_URL and UNLEASH_API_TOKEN are both required\n", stageName)
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
		fmt.Printf("Unleash not initialised: %v\n", err)
		return nil
	}

	if u.BlockUntilReady {
		client.WaitForReady()
	}

	return client
}
