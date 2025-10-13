package teams

import (
	"fmt"
	"github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/messagecard"
	"log"
	"os"
)

type TeamsUtils struct {
	WebHook string
}

type Notification struct {
	Summary string
	Title string
	Text                string
	NotificationButtons []NotificationButton
}

type NotificationButton struct {
	Title string
	Url string
}

func (t *TeamsUtils) SendNotification(notification Notification) {

	fmt.Println("Sending teams notification ...")

	mstClient := goteamsnotify.NewTeamsClient()

	// Destination for OpenUri action.

	msgCard := messagecard.NewMessageCard()

	msgCard.Title = notification.Title
	msgCard.Text = notification.Text
	msgCard.Summary = notification.Summary

	// Setup message card.
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

	// Send the message with default timeout/retry settings.
	if err := mstClient.Send(t.WebHook, msgCard); err != nil {
		log.Printf("failed to send message: %v", err)
		os.Exit(1)
	}

}

