package teams_sdk

import (
	"fmt"
	"github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/messagecard"
	"log"
	"os"
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

func (t *TeamsSDK) SendNotification(notification TeamsNotification) {

	fmt.Println("Sending teams notification ...")

	mstClient := goteamsnotify.NewTeamsClient()

	msgCard := messagecard.NewMessageCard()

	msgCard.Title = notification.Title
	msgCard.Text = notification.Text
	msgCard.Summary = notification.Summary

	msgCard.ThemeColor = "#DF813D"

	for _, b := range notification.NotificationButtons {

		ac, err := messagecard.NewPotentialAction(
			messagecard.PotentialActionOpenURIType,
			b.Title,
		)

		if err != nil {
			log.Fatal("error encountered when creating new action:", err)
		}

		ac.PotentialActionOpenURI.Targets =
			[]messagecard.PotentialActionOpenURITarget{
				{
					OS:  "default",
					URI: b.Url,
				},
			}

		if err := msgCard.AddPotentialAction(ac); err != nil {
			log.Fatal("error encountered when adding action to message card:", err)
		}

	}

	if err := mstClient.Send(t.WebHook, msgCard); err != nil {
		log.Printf("failed to send message: %v", err)
		os.Exit(1)
	}

}
