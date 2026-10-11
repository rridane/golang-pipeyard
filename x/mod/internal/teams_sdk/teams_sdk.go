package teams_sdk

import (
	"fmt"
	"github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/messagecard"
)

type TeamsSDK struct {
	WebHook string
}

type TeamsNotification struct {
	Summary             string
	Title               string
	Text                string
	NotificationButtons []TeamsNotificationButton
}

type TeamsNotificationButton struct {
	Title string
	Url   string
}

// SendNotification posts the card to the webhook. Whether a missed
// notification should fail a stage is the stage's call, so the error is
// returned rather than acted on here.
func (t *TeamsSDK) SendNotification(notification TeamsNotification) error {

	card := messagecard.NewMessageCard()

	card.Title = notification.Title
	card.Text = notification.Text
	card.Summary = notification.Summary
	card.ThemeColor = "#DF813D"

	for _, button := range notification.NotificationButtons {
		action, err := t.openURIAction(button)
		if err != nil {
			return err
		}
		if err := card.AddPotentialAction(action); err != nil {
			return fmt.Errorf("add action %q to the card: %w", button.Title, err)
		}
	}

	if err := goteamsnotify.NewTeamsClient().Send(t.WebHook, card); err != nil {
		return fmt.Errorf("send the Teams notification: %w", err)
	}

	return nil
}

func (t *TeamsSDK) openURIAction(button TeamsNotificationButton) (*messagecard.PotentialAction, error) {

	action, err := messagecard.NewPotentialAction(messagecard.PotentialActionOpenURIType, button.Title)
	if err != nil {
		return nil, fmt.Errorf("create action %q: %w", button.Title, err)
	}

	action.PotentialActionOpenURI.Targets = []messagecard.PotentialActionOpenURITarget{
		{OS: "default", URI: button.Url},
	}

	return action, nil
}
